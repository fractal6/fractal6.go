# Fractale Backend Architecture

This document describes the architecture of **fractal6.go**, the backend and business logic layer for [Fractale](https://fractale.co), a platform for self-organisation.

## Table of Contents

- [Overview](#overview)
- [Directory Structure](#directory-structure)
- [GraphQL Layer](#graphql-layer)
- [Custom Directive System](#custom-directive-system)
- [The @meta Directive: Computed Fields](#the-meta-directive-computed-fields)
- [Authorization Architecture](#authorization-architecture)
- [Tension Event System](#tension-event-system)
- [Database Layer (Dgraph)](#database-layer-dgraph)
- [Hook System](#hook-system)
- [Notification Pipeline](#notification-pipeline)
- [Web Layer and Routing](#web-layer-and-routing)
- [Configuration](#configuration)
- [Key Data Flows](#key-data-flows)

---

## Overview

Fractale models organisations as trees: **Circles** (branches) contain **Roles** (leaves). Both have a **Mandate** document defining purpose and rules. Communication happens through **Tensions** (structured issues linking users and organisations). Changes that require peer validation use **Contracts** (voting mechanisms).

**Tech Stack:**
- **Language:** Go 1.21+ (toolchain 1.22)
- **GraphQL:** gqlgen v0.17.49
- **Database:** Dgraph (GraphQL + DQL over gRPC/HTTP)
- **HTTP Router:** chi/v5
- **Auth:** JWT (RSA-256) via lestrrat-go/jwx/v2
- **Cache/PubSub:** Redis (go-redis/v8)
- **Config:** Viper + Cobra CLI

---

## Directory Structure

```
fractal6.go/
├── cmd/                        # CLI commands (server, notifier, user mgmt)
├── db/                         # Database layer (Dgraph client, DQL/GQL queries)
│   ├── dgraph.go               # Client setup, connection management, JWT for Dgraph
│   ├── dql.go                  # DQL query templates and execution (~2800 lines)
│   ├── gql.go                  # GraphQL query/mutation API to Dgraph
│   └── tensionQuery.go         # Tension query builder with filtering/sorting
├── graph/                      # GraphQL resolvers and business logic
│   ├── generated/              # Auto-generated gqlgen code (DO NOT EDIT)
│   ├── model/                  # GraphQL types (models_gen.go is auto-generated)
│   ├── codec/                  # Data encoding/decoding (nameid codecs)
│   ├── resolver.go             # Resolver init, directive registration, meta implementation
│   ├── schema.resolvers.go     # Auto-generated resolver signatures
│   ├── FieldAuthorization.go   # @x_* directive implementations
│   ├── FieldTransform.go       # @w_* directive implementations
│   ├── tension_auth.go         # Event authorization map (EMAP) and hooks
│   ├── tension_op.go           # Tension event processing pipeline
│   ├── node_op.go              # Node creation/update operations
│   ├── contract_op.go          # Contract handling and voting logic
│   ├── notifications.go        # Redis event publishing
│   ├── *_resolver.go           # Type-specific resolver hooks
│   └── dgraph_resolver.go      # Dgraph bridge utilities
├── schema/                     # GraphQL schema definitions
│   ├── graphql/                # Schema SDL files
│   │   ├── fractal6.graphql    # Main schema with types, enums, directives
│   │   └── directives.graphql  # Custom directive definitions
│   ├── auth/                   # Dgraph authorization rule templates
│   ├── gram/                   # Auto-generated grammars (DO NOT EDIT)
│   └── gqlast.py               # Schema parser & directive propagation engine
├── web/                        # HTTP layer
│   ├── auth/                   # JWT, RBAC, GBAC, validation
│   ├── handlers/               # HTTP endpoint handlers
│   ├── middleware/              # JWT, CORS, context, recovery middleware
│   ├── sessions/               # Redis session management
│   └── email/                  # Email integration
├── tools/                      # Shared utilities (strings, errors, crypto, etc.)
├── config.toml                 # Application configuration
├── gqlgen.yml                  # gqlgen code generation config
└── main.go                     # Entry point
```

---

## Web Layer and Routing

### HTTP Router (chi/v5)

```
POST /api                  - GraphQL endpoint
POST /auth/signup          - User registration
POST /auth/login           - User login
POST /auth/resetpassword   - Password reset
POST /auth/verificate      - Email verification
POST /q/sub_nodes          - Query sub-nodes
POST /q/sub_members        - Query sub-members
POST /q/top_labels         - Query top labels
POST /q/tensions_*         - Filtered tension queries
POST /notifications        - MTA webhook (email replies)
GET  /playground           - GraphQL playground (dev only)
GET  /assets/*             - Static assets
GET  /*                    - Frontend SPA
```

### Middleware Stack

1. `RequestID` - Unique request identifier
2. `RealIP` - Extract real client IP
3. `CORS` - Cross-origin resource sharing
4. `JWT Verifier` - Token signature validation
5. `JWT Decode` - Extract claims into context
6. `Logger` - Request logging
7. `Panic Recovery` - Graceful error recovery
8. `Timeout` - 60 second request timeout

---

## GraphQL Layer

### Schema

The schema is defined in `schema/graphql/fractal6.graphql`. Core types:

| Type | Purpose |
|------|---------|
| `Node` | Circle or Role in the organisation tree |
| `Tension` | Issue/communication between nodes (implements `Post`) |
| `Comment` | Discussion entry on a tension (implements `Post`) |
| `Event` | Immutable history entry for a tension (implements `Post`) |
| `Blob` | Document storage for node data changes (implements `Post`) |
| `Contract` | Validation/voting mechanism (implements `Post`) |
| `Vote` | A participant's vote on a contract (implements `Post`) |
| `User` | Platform user with roles, subscriptions, events |
| `Label` | Categorisation tag for tensions |
| `RoleExt` | Template role definition reusable across circles |
| `Project` | Kanban-style project with columns, cards, fields |

### Code Generation

1. Schema SDL in `schema/graphql/*.graphql`
2. `schema/gqlast.py` parses schema and propagates directives to input types
3. `gqlgen` generates Go code into `graph/generated/` and `graph/model/`
4. Custom resolvers wired in `graph/resolver.go`

Run `make generate` or `make genall` for the full pipeline.

---

## Custom Directive System

The schema uses a rich set of custom directives processed at three levels: schema parsing, code generation, and runtime.

### Directive Categories

**Output Field Directives** (query results):
| Directive | Purpose | File |
|-----------|---------|------|
| `@hidden` | Field can never be read | `resolver.go` |
| `@private` | Only the owning user can read | `resolver.go` |
| `@meta(f, k)` | Computed field via DQL query (k is a list of keys) | `resolver.go` |
| `@isContractValidator` | Boolean: can current user validate? | `contract_resolver.go` |

**Input Field Authorization** (`@x_*`):
| Directive | Purpose | File |
|-----------|---------|------|
| `@x_add` / `@x_alter` | Authorization on add/update mutations | `FieldAuthorization.go` |
| `@x_set` / `@x_remove` / `@x_patch` | Authorization on set/remove/patch | `FieldAuthorization.go` |
| `@x_ro` / `@x_patch_ro` | Read-only (blocks mutation) | `FieldAuthorization.go` |

Authorization rules (the `r` parameter):
- `isOwner` - Only the resource creator can modify
- `unique` - Uniqueness constraint within a scope (field `f`)
- `oneByOne` - Array mutations limited to one element
- `hasEvent` - Required events must exist in tension history
- `tensionTypeCheck` - Validates tension type permissions
- `ref` - Only ID references allowed (no deep creates)
- `minLen` / `maxLen` - String length validation

**Input Field Transformations** (`@w_*`):
| Directive | Purpose | File |
|-----------|---------|------|
| `@w_add` / `@w_alter` | Transform on add/update | `FieldTransform.go` |
| `@w_meta_patch(f, k)` | Trigger DQL mutation post-hook | `resolver.go` |

Transform actions (the `a` parameter):
- `lower` - Convert to lowercase
- `now` - Set to current timestamp

**Type-Level Hooks** (`@hook_`):
Applied to types, auto-generates pre/post mutation hooks (e.g., `@hook_` on `Tension` generates `Hook_addTensionInput` and `Hook_addTension`).

### Directive Propagation

The `schema/gqlast.py` parser propagates directives from type fields to generated input types:
- `Add*Input` receives `@w_(add|alter)` and `@x_(add|alter)` directives
- `*Patch` inputs receive all `@w_*` and `@x_*` with `@x_patch_ro` default
- `*Filter` inputs receive `@w_alter` only
- `*Ref` inputs receive `@w_*` and `@x_*` (without plain alter)

---

## The @meta Directive: Computed Fields

The `@meta` directive enables declarative computed fields backed by DQL queries. This is a key architectural pattern for fields that require complex aggregation or traversal.

### Schema Declaration

```graphql
directive @meta(f: String!, k: [String!]) on FIELD_DEFINITION
```

- `f` - Name of a DQL query template
- `k` - List of key fields to extract from the parent object as query parameters. The first key is required (primary key); subsequent keys are optional and skipped when nil/empty.

In addition to `k` keys, `meta()` also collects **field arguments** (e.g., `query: String`) and adds them to the template map. This allows DQL templates to use both parent object fields and query-time parameters.

### Usage Examples

```graphql
type Node {
  events_history(query: String): [Event!] @meta(f:"getNodeHistory", k:["nameid"])
}

type User {
  event_count: EventCount @meta(f:"getEventCount", k:["username"])
}
```

### Request Flow

```
GraphQL Query (e.g., query { getNode { events_history { ... } } })
    |
    v
Generated resolver: _Node_events_history()
    |
    v
Directive chain: directive0 (return obj.EventsHistory) -> directive1 (unmarshal @meta args)
    |
    v
meta() function [graph/resolver.go:307-381]
    |-- Calls next(ctx) to get initial value
    |-- Iterates over key list, extracts each value via reflection (handles *string pointers)
    |-- Collects field arguments (e.g., query) into template map
    |-- Calls db.GetDB().Meta("getNodeHistory", {"nameid": "<value>", "query": "<optional>"})
    |
    v
Meta() function [db/dql.go:1191-1229]
    |-- Looks up "getNodeHistory" in dqlQueries map
    |-- Calls QueryDql() which templates the query ({{.nameid}} -> actual value)
    |
    v
Dgraph DQL execution via gRPC
    |
    v
JSON response unmarshaled -> []map[string]interface{}
    |
    v
meta() converts maps to []*model.Event via reflection + Map2Struct
    |
    v
Result marshaled to GraphQL response
```

### DQL Template Example: getNodeHistory

```dql
{
    # Find the node and its direct children
    var(func: eq(Node.nameid, "{{.nameid}}")) {
        n1 as uid
        n2 as Node.children
    }

    # Collect all tension history events for these nodes
    # When query argument is set, filter tensions by title/message match
    var(func: uid(n1, n2)) {
        Node.tensions_in {{if .query}}@filter(
            anyoftext(Tension.title, "{{.query}}")
            OR anyoftext(Post.message, "{{.query}}")){{end}} {
            h as Tension.history
        }
    }

    # Return the 25 most recent events (excluding BlobCreated)
    all(func: uid(h), first:25, orderdesc: Post.createdAt)
        @filter(NOT eq(Event.event_type, "BlobCreated")) {
        Post.createdAt
        Post.createdBy { User.username }
        Event.event_type
        Event.tension {
            uid
            Tension.title
            Tension.receiver { Node.name Node.nameid }
        }
    }
}
```

This query traverses the node tree, collects tension events across children, and returns a unified, sorted activity feed. When the `query` field argument is provided, the Go template conditionally adds a `@filter` that narrows tensions to those matching the search term in title or message (using Dgraph's `anyoftext` full-text search).

### Other @meta Fields

| Field | Template | Keys | Returns |
|-------|----------|------|---------|
| `Node.events_history` | `getNodeHistory` | `nameid` + `query` field arg (optional) | `[Event!]` - Recent activity, filterable by text |
| `User.event_count` | `getEventCount` | `username` | `EventCount` - Unread counts |
| `User.markAllAsRead` | `markAllAsRead` | `username` | Mutation via `@w_meta_patch` |

---

## Authorization Architecture

Authorization is enforced at four layers:

### Layer 1: Dgraph Schema Rules (`@auth`)

Built-in Dgraph authorization. Rules defined in `schema/auth/` and embedded in the schema:

```graphql
type Node @auth(
  query: <<query-node>>,
  add: <<is-root>>,
  update: <<is-root>>,
  delete: <<is-root>>
)
```

JWT claims (`DgraphClaims`) carry `Username`, `UserType`, `Rootids` (member orgs), and `Ownids` (owner orgs).

### Layer 2: Field-Level Directives (`@x_*`)

Custom directive middleware in `graph/FieldAuthorization.go`. Applied per-field on inputs:

```graphql
title: String! @x_alter(r:"minLen", n:1) @x_alter(r:"maxLen", n:100)
```

### Layer 3: Type-Level Hooks (`@hook_`)

Pre/post mutation hooks in resolver files. Perform complex authorization checks:

```go
// In node_resolver.go: addNodeArtefactHook
// Checks coordinator authority, rootnameid compliance, etc.
```

### Layer 4: Event Authorization (EMAP)

The `EventsMap` in `graph/tension_op.go` maps each `TensionEvent` to authorization rules:

```go
model.TensionEventBlobPushed: EventMap{
    Auth:       TargetCoordoHook | AssigneeHook,
    Validation: model.ContractTypeAnyCoordoTarget,
    Action:     PushBlob,
}
```

**Authorization Hooks** (bitflag composition):

| Hook | Meaning |
|------|---------|
| `PassingHook` | No auth required |
| `OwnerHook` | Must be org owner |
| `MemberHook` | Must be org member |
| `MemberStrictHook` | Member or guest with rights |
| `SourceCoordoHook` | Coordinator of emitter circle |
| `TargetCoordoHook` | Coordinator of receiver circle |
| `AuthorHook` | Creator of the tension |
| `AssigneeHook` | Assigned to the tension |
| `CandidateHook` | Contract candidate |

---

## Tension Event System

Tensions are the core communication primitive. Every change to a tension creates an `Event` entry, forming an immutable history.

### Event Processing Pipeline

```
updateTension mutation
    |
    v
TensionEventHook(uctx, tid, events, blob)  [tension_resolver.go]
    |
    v  (for each event in the batch)
ProcessEvent(uctx, tension, event, blob, contract, doCheck, doProcess)  [tension_op.go]
    |
    |-- doCheck: EMAP[event].Check()  -> authorization validation
    |-- doProcess: EMAP[event].Action() -> execute side effect
    |
    v
leaveTrace() -> updates parent node timestamps
    |
    v
PublishTensionEvent() -> Redis pub/sub for notifications
```

### Key Event Types and Actions

| Event | Auth | Action | Description |
|-------|------|--------|-------------|
| `BlobPushed` | TargetCoordo/Assignee | `PushBlob` | Apply blob changes to node |
| `UserJoined` | Member | `UserJoin` | Add user to circle |
| `UserLeft` | Member/Author | `UserLeave` | Remove user from circle |
| `MemberLinked` | TargetCoordo | `ChangeFirstLink` | Assign user to role |
| `Moved` | Author/Coordo | `MoveTension` | Move tension between nodes |
| `Authority` | TargetCoordo | `ChangeAuthority` | Change node governance mode |
| `Visibility` | TargetCoordo | `ChangeVisibility` | Change node visibility |

### Contract Validation

When an event requires peer validation, a `Contract` is created:

| Contract Type | Who Validates |
|---------------|---------------|
| `AnyCoordoDual` | Coordinators from both source and target |
| `AnyCandidates` | Specific nominated candidates |
| `AnyCoordoSource` | Emitter circle coordinator |
| `AnyCoordoTarget` | Receiver circle coordinator |

---

## Database Layer (Dgraph)

### Dual Query System

The database layer uses two Dgraph interfaces:

**GraphQL API** (`db/gql.go`) - HTTP to Dgraph's GraphQL endpoint:
- Used for standard CRUD operations
- Dgraph `@auth` rules enforced automatically
- Methods: `Query()`, `Get()`, `Add()`, `Update()`, `Delete()`

**DQL (Dgraph Query Language)** (`db/dql.go`) - gRPC to Dgraph:
- Used for complex aggregations, recursive traversals, computed fields
- Bypasses GraphQL auth (custom authorization needed)
- Template-based queries with `{{.key}}` substitution
- ~40+ named query templates in `dqlQueries` map
- Methods: `QueryDql()`, `MutateWithQueryDql()`, `Meta()`

### Connection Setup

```go
// gRPC for DQL queries
dg.grpcAddr = fmt.Sprintf("%s:%s", host, portGrpc)

// HTTP for GraphQL queries
dg.gqlAddr = fmt.Sprintf("http://%s:%s/graphql", host, portGraphql)
```

### JWT for Dgraph

The backend generates JWTs for Dgraph auth containing user claims:

```go
type DgraphClaims struct {
    Username string
    UserType model.UserType
    Rootids  []string  // orgs where user is member
    Ownids   []string  // orgs where user is owner
}
```

---

## Hook System

Hooks provide pre/post processing for GraphQL mutations. They are generated from the `@hook_` directive.

### Hook Types

**Input Hooks** (`Hook_*Input`) - Before mutation:
- Validate and enrich input data
- Set context values for downstream processing
- Example: `setContextWithID` extracts the mutation target ID

**Mutation Hooks** (`Hook_add*`, `Hook_update*`, `Hook_delete*`) - After mutation:
- Execute business logic side effects
- Trigger notifications
- Cascade updates
- Example: `addTensionHook` processes events and publishes notifications

### Hook Registration

All hooks are wired in `graph/resolver.go` `Init()`:

```go
c.Directives.Hook_addTensionInput = setContextWithID
c.Directives.Hook_addTension = addTensionHook
c.Directives.Hook_updateTensionInput = setContextWithID
c.Directives.Hook_updateTension = updateTensionHook
// ... ~60 hooks total
```

---

## Notification Pipeline

### Architecture

```
API Server                          Notifier Daemon
    |                                    |
    | PublishTensionEvent()              |
    |----> Redis PubSub Channel ------->|
    |      "api-tension-notification"   |
    |                                   | Process event
    |                                   | Build notification list
    |                                   | Create UserEvent records
    |                                   | Send emails (via Postal MTA)
    |                                   |
```

### Event Types

- `EventNotif` - Tension event notifications
- `ContractNotif` - Contract voting notifications
- `NotifNotif` - Generic notifications with arbitrary links

### Subscriber Resolution

Notifications are sent to:
- Tension subscribers
- Tension assignees
- Receiver circle coordinators
- Emitter circle coordinators (for created tensions)
- Contract candidates

---

## Configuration

Configuration managed via `config.toml` with Viper:

```toml
[server]
domain = "fractale.co"
hostname = "localhost"
port = "8484"

[db]
hostname = "localhost"
port_graphql = "8080"
port_grpc = "9080"

[graphql]
complexity_limit = 200
introspection = false

[mailer]
admin_email = "..."
```

Build modes (`DEV`/`PROD`) injected via Makefile flags.

---

## Key Data Flows

### Tension Creation

```
HTTP POST /api (createTension mutation)
  -> JWT middleware (extract UserCtx)
  -> Hook_addTensionInput (set context ID)
  -> Field directives (@w_add: lowercase, @x_add: validate)
  -> Dgraph @auth rules (membership check)
  -> DB mutation (gql.go Add())
  -> Hook_addTension:
       -> Process events (TensionEventHook)
       -> Authorize each event (EMAP check)
       -> Execute event actions
       -> Publish to Redis (PublishTensionEvent)
  -> Return AddTensionPayload
```

### Contract Voting

```
HTTP POST /api (addVote mutation)
  -> Authorize user has contract rights
  -> Create Vote record
  -> Hook_addVote:
       -> Check if contract is complete
       -> If passes: execute contract's event (ProcessEvent)
       -> Update tension with event data
       -> Publish completion notification
  -> Return AddVotePayload
```

### Node History Query (via @meta)

```
HTTP POST /api (query { getNode { events_history { ... } } })
  -> JWT middleware
  -> Dgraph @auth query filter on Node
  -> @meta directive triggers:
       -> Extract nameid (required) from Node object + query field argument (optional)
       -> Execute getNodeHistory DQL template with parameter map
       -> Traverse node + children tensions (filtered by query if set)
       -> Collect + sort 25 most recent events
       -> Map to []*model.Event
  -> Return events_history field
```
