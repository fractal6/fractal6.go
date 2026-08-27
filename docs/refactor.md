# Refactoring backlog

Open improvement opportunities, ordered by priority. Items are dropped from this file
once done — it is a backlog, not a changelog.

| # | Item | Impact | Effort |
|---|------|--------|--------|
| 1 | Test coverage for `graph/` and `db/` | High | High |
| 2 | DQL template injection safety (residual) | Medium | Medium |
| 3 | Eliminate the dual bridge system | High | Medium |
| 4 | Replace the Redis `@meta_patch` hack | High | Low |
| 5 | Structured error handling | High | Medium |
| 6 | Extract business logic from hooks | High | High |
| 7 | Consolidate hook registration | Medium | Low |
| 8 | Type-safe input extraction | Medium | Low |
| 9 | Standardize resolver activation | Medium | Low |
| 10 | Request-level observability | Medium | Medium |
| 11 | Database interface abstraction | Medium | Medium |
| 12 | Modernize dependencies | Medium | Medium |
| 13 | Unify the artefact resolver pattern | Medium | Medium |
| 14 | Reduce hardcoded DQL payloads | Low | Medium |
| 15 | Resolve scattered `@DEBUG` / `@obsolete` annotations | Low | Low |
| 16 | Schema documentation | Low | Medium |

## 1. Test coverage for `graph/` and `db/`

Integration tests cover `db/` queries and the HTTP handlers, but the tension event
pipeline (`TensionEventHook`, `ProcessEvent`), the EMAP auth hooks in
`tension_auth.go` and most `xw_directive.go` rules still have no safety net —
these are the security-critical paths.

## 2. DQL template injection safety (residual)

The two exploitable paths are already closed: `db.ValidateUids` guards every
client-supplied id before it reaches a `uid()` root, and `QuoteString` escapes user
patterns spliced into string literals (see [text search](search.md)).

What remains is the other `text/template` interpolations (`{{.nameid}}` and friends),
which carry internally-generated or already-validated values and are therefore not
known to be exploitable — just unguarded by construction. Closing it means moving
them to Dgraph query variables (`$nameid`) where the template shape allows, and
labelling the parameters that are user-controlled so the next template author knows
which need escaping.

## 3. Eliminate the dual bridge system

`DgraphBridgeRaw` forwards the client's raw query string, losing directive
modifications, and regex-strips the `history` field via a `cut_history` context flag.
The `Get*` queries are its last callers. A `DgraphGetBridge` built on the existing
`GetQueryGraph` infrastructure would let the hook simply nil out `input.Set.History`
and remove the regex entirely.

## 4. Replace the Redis `@meta_patch` hack

`@w_meta_patch` stashes function/key/value in Redis under `username + "meta_patch_*"`
with a 5s TTL, read back in `postGqlProcess`. Concurrent requests from the same user
overwrite each other. The operation is request-scoped — pass it through
`context.Context` instead.

## 5. Structured error handling

No error type hierarchy: `LogErr()` returns flat strings, some authorization failures
return empty responses instead of errors, and messages mix French and English. Define
sentinel errors (`ErrUnauthorized`, `ErrNotFound`, `ErrValidation`, `ErrConflict`),
wrap with `%w`, check with `errors.Is/As`, and standardise on English so the frontend
can distinguish cases.

## 6. Extract business logic from hooks

Resolver hooks mix GraphQL plumbing (context extraction, type assertions, directive
chain) with business logic (event processing, auth, notifications). Extracting the
logic into service functions taking typed parameters would make it unit-testable
without a GraphQL context, leaving hooks as thin adapters.

## 7. Consolidate hook registration

`resolver.go:Init()` registers ~60 hooks by hand and most are the pass-through
`nothing`. Defaulting every `Hook_*` field via a reflection loop and registering only
the custom ones would cut `Init()` by ~80%.

## 8. Type-safe input extraction

The generic `ExtractInputs[T]` / `ExtractInput[T]` helpers exist in `resolver.go` but are
used inconsistently: `node_resolver.go` uses them while `tension_resolver.go` and
`contract_resolver.go` still do raw `Args["input"].(...)` assertions that panic on
failure. Unify on the helpers, and give them error returns instead of panicking.

## 9. Standardize resolver activation

`schema.resolvers.go` has ~130 `panic("not implemented")` bodies mixed in with live
bridge calls and one-off implementations, with no way to see what is exposed without
reading the whole file. Some of them *should* be wired (e.g. `DeleteLabel`). A
registration table would document it, and returning a GraphQL error rather than
panicking would stop an unexpected call from taking down the server.

## 10. Request-level observability

Logging is `fmt.Println` in DEV mode. Prometheus metrics exist
(`web/handlers/instrumentation.go`) but there is no structured logging, no DQL query
timing, and no audit trail of authorization denials. `slog` plus the existing
`RequestID` middleware covers most of it.

## 11. Database interface abstraction

`Resolver` holds a concrete `*db.Dgraph`. A `db.Store` interface over the core
operations would allow mock implementations in tests.

## 12. Modernize dependencies

`go-redis/redis/v8` (v9 is current, context-first API) and `dgraph-io/dgo/v200`
(legacy versioning) are both behind.

## 13. Unify the artefact resolver pattern

Label, RoleExt and Project share one authorization shape (artefacts attached to nodes,
coordinator auth) but go through ad-hoc `AddArtefactInput` / `UpdateArtefactInput` proxy
types in `node_resolver.go` that partially duplicate the generated models. A generic
hook factory constrained on `GetNodes()` / `GetRootnameid()` would remove the proxies
and the `typeName` switching in `updateNodeArtefactHook`.

## 14. Reduce hardcoded DQL payloads

`db/dql_payloads.go` duplicates field selections from the schema by hand; a schema
change requires updating them manually with no compile-time signal. Generate them
during `make generate`, or at minimum test the field names against the live schema.

## 15. Resolve scattered annotations

`@DEBUG`, `@debug`, `@future`, `@obsolete` and `@refactor` markers accumulate in the
schema and code (a dozen in `fractal6.graphql` alone), several waiting on Dgraph
features like nested filters. Audit each one: resolve, convert to a tracked issue, or
delete.

## 16. Schema documentation

`fractal6.graphql` has almost no `"""description"""` strings, despite being the
primary contract with the frontend. `TensionEvent` in particular is undocumented.
