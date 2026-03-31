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
- [Resolver Architecture: Schema-to-Runtime Pipeline](#resolver-architecture-schema-to-runtime-pipeline)
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
│   ├── dql.go                  # DQL query templates, execution, and generic helpers
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
POST /api                              - GraphQL endpoint

# Auth API - User
POST /auth/signup                      - User registration
POST /auth/validate                    - Email verification
POST /auth/login                       - User login
GET  /auth/logout                      - User logout
POST /auth/tokenack                    - Token acknowledgment
POST /auth/resetpasswordchallenge      - Password reset challenge (captcha)
POST /auth/resetpassword               - Password reset
POST /auth/resetpassword2              - Password reset (step 2)
POST /auth/uuidcheck                   - UUID check
POST /auth/updatepassword              - Update password

# Auth API - Organisation
POST /auth/createorga                  - Create organisation
POST /auth/setusercanjoin              - Set user-can-join flag
POST /auth/setguestcancreatetension    - Set guest-can-create-tension flag
POST /auth/setlexicon                  - Set organisation lexicon
POST /auth/makeowner                   - Transfer ownership

# REST API - Queries (with visibility filtering)
POST /q/nodes/sub                      - Query sub-nodes
POST /q/members/sub                    - Query sub-members
POST /q/labels/top                     - Query top labels
POST /q/labels/sub                     - Query sub labels
POST /q/roles/top                      - Query top roles
POST /q/roles/sub                      - Query sub roles
POST /q/projects/sub                   - Query sub-projects
POST /q/tensions/{mode}                - Filtered tension queries (light, int, ext, all)
POST /q/tensions/count                 - Tension count query

# Webhooks
POST /notifications                    - MTA webhook (email replies)
POST /mailing                          - Mailing webhook
POST /postal_webhook                   - Postal webhook

# Dev & Static
GET  /playground                       - GraphQL playground (dev only)
GET  /ping                             - Health check (dev only)
GET  /assets/*                         - Static assets
GET  /*                                - Frontend SPA
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
| `TensionTemplate` | Pre-filled tension template scoped to circles |
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
meta() converts maps to []*model.Event via reflect.New + json.Marshal/Unmarshal
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

### Collaborator-Based Access (per-resource)

Some resource types support direct user-level access that bypasses the node-based coordinator hierarchy. This is used when external users (who may not be organisation members) need read/write access to a specific resource.

**Currently applies to:** `Project`

The `Project` type has a `collaborators: [User!]` field. When a user is listed as a collaborator, they are granted access at two levels:

- **Dgraph `@auth` rules** (`schema/auth/query-project.gql`, `alter-project.gql`): Include a rule that checks if the requesting user's `$USERNAME` matches any entry in `Project.collaborators`. This grants query and mutation access at the database level.
- **Hook-level auth** (`web/auth/gbac.go:CheckProjectAuth`): Checks collaborator membership first (cheap username match) before falling back to the standard node-based coordinator authorization. This means a collaborator gets access even without any role in the organisation tree.

```
CheckProjectAuth(uctx, projectid)
    |
    ├── isProjectCollaborator? ──yes──> ALLOW
    |
    └── checkProjectNodeAuth (coordinator check on linked nodes)
            |
            └── CheckNodesAuth -> HasCoordoAuth -> ...
```

This pattern can be extended to other resource types that need direct user invitations without requiring organisation membership.

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
- Methods: `QueryDql()`, `MutateWithQueryDql()`, `Meta()`, `Gamma()`

### Generic DQL Helpers

The `db` package provides generic package-level functions that compose cleanly for typed DQL access:

```go
// Execute a named DQL query/mutation, decode results into typed values
users, err := db.Meta[model.User]("getWatchers", maps)

// Execute a custom DQL query/mutation, decode results into typed values
col, err := First(db.Gamma[ProjectColumnLoc](QueryColumnLoc, maps))

// Extract the first result (composes with Meta/Gamma)
tension, err := First(db.Meta[model.Tension]("getTensionSimple", maps))
```

`db.Meta[T]` and `db.Gamma[T]` call `GetDB()` internally and use `DecodeDql[[]T]` (from `internal/tools/dql_decode.go`) for JSON-based type conversion. `First[T]` and `DecodeDql[T]` live in `internal/tools/dql_decode.go` as pure generic helpers with no db dependency. These coexist with the untyped `(dg Dgraph).Meta()` / `(dg Dgraph).Gamma()` methods, which are still used for mutations that discard results or need raw map access (e.g., `resolver.go` reflection-based `meta()`).

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

## Resolver Architecture: Schema-to-Runtime Pipeline

This section details the subtle interactions between the GraphQL schema, the code generation pipeline, and the resolver files that bring the API to life.

### Schema Build Pipeline

The schema goes through multiple transformation stages before it can be used. Each stage produces a distinct artifact:

```
schema/graphql/fractal6.graphql       (1) Source schema with <<placeholder>> auth rules
         |
         v
schema/gqlauth.py + schema/auth/*.gql (2) Replace <<rule-name>> with Dgraph @auth rules
         |
         v
schema/fractal6-gen.graphql           (3) Schema with real @auth rules injected
         |
         ├──> gqlast.py --dgraph      (4a) Strip custom directives, produce Dgraph-compatible schema
         |         |
         |         v
         |    schema/dgraph_schema.graphql  -> pushed to Dgraph via HTTP API
         |
         |    Dgraph generates its own Query/Mutation types based on the schema
         |         |
         |         v
         |    schema/dgraph_out.graphql     (fetched via get-graphql-schema)
         |
         v
schema/gqlast.py + directives.graphql (4b) Merge Dgraph output + custom directives
         |                                  + propagate @x_*/@w_* to input types
         v
schema/schema.graphql                 (5) Final unified schema for gqlgen
         |
         v
gqlgen generate (go generate ./...)   (6) Produce Go code
         |
         ├──> graph/generated/        Auto-generated execution engine (DO NOT EDIT)
         ├──> graph/model/models_gen.go  Type definitions
         └──> graph/schema.resolvers.go  Resolver stubs
```

This pipeline is orchestrated by `make genall` (root Makefile) which calls:
1. `make dgraph_schema` -> `schema/Makefile:dgraph_in` (auth_schema + gqlast --dgraph)
2. `make dgraph` -> pushes schema to Dgraph
3. `make dgraph_out` -> fetches Dgraph-generated schema
4. `make gqlgen_schema` -> `schema/Makefile:gqlgen_in` (gqlast merge)
5. `make generate` -> `go generate ./...` (gqlgen)

### Auth Rule Injection (`schema/Makefile:auth_schema`)

The source schema uses placeholders like `<<query-node>>` inside `@auth()` directives:

```graphql
type Node @auth(
  query: <<query-node>>,
  add: <<is-root>>,
  update: <<is-root>>,
  delete: <<is-root>>
)
```

The `gqlauth.py` script scans for `<<name>>` patterns and replaces each with the contents of `schema/auth/{name}.gql`. For example, `<<query-node>>` loads `schema/auth/query-node.gql`, which contains Dgraph RBAC/GBAC rules using JWT claims (`$USERNAME`, `$USERTYPE`, `$ROOTIDS`, `$OWNIDS`):

```graphql
{ or: [
    { rule: "{ $USERTYPE: {eq: \"Root\"} }" },
    { rule: """query ($OWNIDS: [String]) {
        queryNode(filter: {visibility: {eq: Public}, or: [{rootnameid: {in: $OWNIDS}}]}) { id }
    }""" },
    { rule: """query ($ROOTIDS: [String]) {
        queryNode(filter: {visibility: {eq: Private}, and: [{rootnameid: {in: $ROOTIDS}}]}) { id }
    }""" },
    ...
  ]
}
```

There are ~20 auth rule files covering queries, adds, updates, and deletes for each protected type. These rules are evaluated by Dgraph's alpha server at query time, filtering results based on the user's JWT claims.

### Generated Resolvers (`graph/schema.resolvers.go`)

gqlgen generates `schema.resolvers.go` with a resolver stub for every Query and Mutation defined in the schema. Each type gets up to 6 stubs: `Add*`, `Update*`, `Delete*`, `Get*`, `Query*`, `Aggregate*`.

By default, all stubs `panic("not implemented")`. The developer activates a resolver by replacing the panic with a call to the appropriate **Dgraph bridge**:

```go
// NOT ACTIVATED - panics at runtime
func (r *mutationResolver) AddNode(ctx context.Context, input []*model.AddNodeInput, upsert *bool) (*model.AddNodePayload, error) {
    panic(fmt.Errorf("not implemented"))
}

// ACTIVATED - delegates to Dgraph bridge
func (r *mutationResolver) AddLabel(ctx context.Context, input []*model.AddLabelInput) (*model.AddLabelPayload, error) {
    errors = r.DgraphAddBridge(ctx, input, nil, &data)
    return data, errors
}
```

This is a deliberate design choice: only types that need to be exposed through the API are wired up. Internal types (Node, Blob, Event, etc.) are manipulated exclusively through business logic hooks, never directly via their generated mutations.

**Currently activated resolvers:**

| Bridge | Activated Types |
|--------|----------------|
| `DgraphAddBridge` | Label, RoleExt, TensionTemplate, Project, ProjectColumn, ProjectCard, Tension, Reaction, Contract, Vote |
| `DgraphUpdateBridge` | Label, RoleExt, TensionTemplate, Project, ProjectColumn, ProjectCard, Tension, Comment, ProjectDraft, User, Contract, UserEvent |
| `DgraphDeleteBridge` | ProjectColumn, ProjectCard, Comment, Reaction, Contract |
| `DgraphBridgeRaw` | GetNode, GetTension, GetUser, GetProject, GetTensionTemplate, GetProjectColumn, GetContract, AggregateProject |
| `DgraphQueryBridge` | QueryNode, QueryLabel, QueryTensionTemplate, QueryTension, QueryUser |

### Dgraph Bridges (`graph/dgraph_resolver.go`)

The bridges are the critical link between gqlgen resolvers and Dgraph. They translate the GraphQL operation context into database calls:

```go
func (r *mutationResolver) DgraphAddBridge(ctx context.Context, input interface{}, upsert *bool, data interface{}) error
func (r *mutationResolver) DgraphUpdateBridge(ctx context.Context, input interface{}, data interface{}) error
func (r *mutationResolver) DgraphDeleteBridge(ctx context.Context, filter interface{}, data interface{}) error
func (r *queryResolver) DgraphQueryBridge(ctx context.Context, filter, order any, first, offset *int, data any) error
func (r *queryResolver) DgraphBridgeRaw(ctx context.Context, data interface{}) error  // @deprecated
```

**How bridges work:**

1. **Extract context**: `getUserQueryType()` pulls the authenticated user (`UserCtx`) and determines the operation type (`add`, `update`, `delete`, `query`, `get`) by parsing the GraphQL field name (e.g., `addLabel` -> type=`add`, name=`Label`).

2. **Reconstruct query projection**: `GetQueryGraph(ctx)` walks the gqlgen field collection tree to build the return field selection string that tells Dgraph which fields to return.

3. **Delegate to `db/` layer**: Calls `db.AddExtra()`, `db.UpdateExtra()`, etc., which build the full GraphQL request to Dgraph's HTTP API (including the user's JWT for `@auth` enforcement).

4. **Post-process**: `postGqlProcess()` handles two subtleties:
   - **Silent error swallowing**: If Dgraph returns both data and an error (common when `@auth` rules filter some but not all results), the error is silently ignored. This is necessary because gqlgen drops data when an error is present.
   - **`@meta_patch` execution**: Checks Redis for pending post-mutation DQL operations (set by the `@w_meta_patch` directive) and executes them.

**Raw bridge (`DgraphBridgeRaw`)**: An older, deprecated approach that forwards the client's raw GraphQL query directly to Dgraph. It bypasses directive-modified inputs (changes made by `@w_*` or `@x_*` are lost) but preserves the full query structure. Still used for `Get*` queries and some aggregates where input modification is not needed.

### The Resolver Hub (`graph/resolver.go`)

`resolver.go` is the dependency injection center. Its `Init()` function creates the `gen.Config` with:

1. **The Resolver struct** - holds a pointer to the Dgraph client (`*db.Dgraph`)
2. **~10 field directive implementations** - `@hidden`, `@private`, `@meta`, `@x_*`, `@w_*`, etc.
3. **~60 hook registrations** - mapping each `@hook_` directive to its implementation function

The hook registration follows a naming convention:

```go
// Input hooks: run BEFORE the mutation, in the directive chain
c.Directives.Hook_addTensionInput  = nothing          // no pre-processing
c.Directives.Hook_updateTensionInput = setUpdateContextInfo  // extract IDs + set/remove flags

// Mutation hooks: wrap the ENTIRE resolver (before AND after the DB call)
c.Directives.Hook_addTension  = addTensionHook        // complex event processing
c.Directives.Hook_updateTension = updateTensionHook    // complex event processing
```

The `nothing` function is a pass-through (`return next(ctx)`) used for hooks that don't need custom logic. Common input hooks include:
- `setContextWithID` - Extracts `id`, `nameid`, `rootnameid`, `username` from the input and stores them in the context for downstream directives (used by `@isOwner`, `@unique`, `@private`)
- `setUpdateContextInfo` - Like `setContextWithID` but also records whether `set` and `remove` are present (used by `@hasEvent`)

### Resolver Files: Business Logic Hooks

Each `*_resolver.go` file contains the hook implementations for a domain:

| File | Hooks | Domain |
|------|-------|--------|
| `tension_resolver.go` | `addTensionHook`, `updateTensionHook` | Tension lifecycle + event processing |
| `node_resolver.go` | `addNodeArtefactHook`, `updateNodeArtefactHook` | Label/RoleExt/TensionTemplate/Project auth + CRUD |
| `contract_resolver.go` | `addContractInputHook`, `addContractHook`, `updateContractHook`, `deleteContractHook` | Contract lifecycle + voting |
| `card_resolver.go` | `addProjectCardHook`, `updateProjectCardHook`, `deleteProjectCardHook` | Card position management |
| `column_resolver.go` | `addProjectColumnHook`, `updateProjectColumnHook`, `deleteProjectColumnHook` | Column position management |
| `draft_resolver.go` | `updateProjectDraftHook` | Draft author ownership |
| `reaction_resolver.go` | `addReactionInputHook` | Reaction ID generation |

**Mutation hook pattern** (wraps the full resolver):

```go
func addTensionHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    // 1. PRE-PROCESSING: authenticate, validate, extract data
    ctx, uctx, err := auth.GetUserContext(ctx)
    inputs := graphql.GetResolverContext(ctx).Args["input"].([]*model.AddTensionInput)
    history := inputs[0].History
    inputs[0].History = nil   // cut history (will be pushed separately)

    // 2. EXECUTE: call next(ctx) which eventually hits DgraphAddBridge
    data, err := next(ctx)

    // 3. POST-PROCESSING: event processing, notifications
    ok, _, err := TensionEventHook(uctx, id, history, nil)
    PublishTensionEvent(model.EventNotif{...})

    return data, err
}
```

### The Full Runtime Flow

When a GraphQL request arrives, the execution flows through these layers in order:

```
1. HTTP Middleware (JWT decode -> UserCtx in context)
         |
2. gqlgen Execution Engine (graph/generated/)
         |
    ┌────┴────────────────────────────────────────────────┐
    │  DIRECTIVE CHAIN (runs in order for each field):    │
    │                                                      │
    │  a) @hook_*Input  (setContextWithID, etc.)          │
    │     └─ Enriches context with IDs for downstream     │
    │                                                      │
    │  b) @w_* transforms (lower, now)                    │
    │     └─ Modifies input values in-place               │
    │                                                      │
    │  c) @x_* authorization (isOwner, unique, ref, etc.) │
    │     └─ Validates or rejects the field               │
    │                                                      │
    │  d) @hook_add*/update*/delete* (mutation hooks)     │
    │     └─ Wraps the resolver with pre/post logic       │
    └─────────────────────────────────────────────────────┘
         |
3. Resolver function (schema.resolvers.go)
    └─ Calls DgraphAddBridge / DgraphUpdateBridge / etc.
         |
4. Dgraph Bridge (dgraph_resolver.go)
    └─ Reconstructs query, delegates to db layer
         |
5. Database layer (db/gql.go)
    └─ Sends GraphQL to Dgraph with user JWT
         |
6. Dgraph Alpha Server
    └─ Evaluates @auth rules, executes mutation/query
         |
7. Post-processing (postGqlProcess)
    └─ Silent error handling + @meta_patch execution
         |
8. Response returned to client
```

**Important subtlety about hook ordering**: Mutation hooks (`@hook_addTension`) wrap the *entire* resolver, meaning they execute *around* step 3-7. The hook calls `next(ctx)` which triggers the bridge, which calls Dgraph. The hook's post-processing code runs *after* the database operation completes. This is how `addTensionHook` can cut the history from the input before the mutation, then push it separately after the tension is created.

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
