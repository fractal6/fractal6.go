# Refactoring Recommendations

Major improvement opportunities identified across the fractal6.go codebase, ordered by impact.

## Summary: Priority Matrix

| # | Improvement | Impact | Effort | Priority |
|---|-------------|--------|--------|----------|
| 2 | Test coverage for graph/db | High | High | Critical |
| 6 | DQL template injection safety | High | Medium | Critical |
| 5 | Structured error handling | High | Medium | High |
| 1 | Split dql.go | Medium | Low | High |
| 4 | ~~Remove dot-imports~~ | Medium | Low | High |
| 8 | Extract business logic from hooks | High | High | High |
| 3 | Replace reflection with generics | Medium | Medium | Medium (mostly done) |
| 14 | Database interface abstraction | Medium | Medium | Medium |
| 10 | Request-level observability | Medium | Medium | Medium |
| 7 | Consolidate hook registration | Low | Low | Medium |
| 11 | Simplify meta() type switch | Low | Low | Medium |
| 9 | Modernize dependencies | Medium | Medium | Low (mapstructure direct dependency removed) |
| 12 | Reduce hardcoded payloads | Low | Medium | Low |
| 13 | Clean up TODO/DEBUG | Low | Low | Low |
| 15 | Schema documentation | Medium | Medium | Low |


---

## 1. Split `db/dql.go` (~2800 lines)

**Problem:** `dql.go` is the largest source file. It mixes DQL query templates, payload strings, query execution methods, and data transformation utilities in a single file.

**Recommendation:**
- ~~Extract DQL templates into a `db/dql_templates.go` (or use embedded `.dql` files with `//go:embed`)~~
- ~~Extract payload definitions into `db/payloads.go`~~
- ~~Keep execution methods (`Meta()`, `Gamma()`, `QueryDql()`, generic helpers `Meta[T]`/`Gamma[T]`/`First[T]`) in `dql.go`~~
- ~~Extract data mapping helpers (`CleanCompositeName`, `Map2Struct` wrappers) into `db/mappers.go`~~ (Done: `DecodeDql[T]` generic helper in `internal/tools/dql_decode.go` replaces all DQL mapstructure boilerplate)

**Benefit:** Easier navigation, clear separation between query definitions and execution logic. The existing `@refactor` comment at line 38 already acknowledges this need.

---

## 2. Increase Test Coverage

**Problem:** Only 6 test files exist, all in `tools/` and `web/auth/`. Zero tests for `graph/` (resolver logic, authorization, event processing) and `db/` (query building, data mapping). These are the most critical packages.

**Recommendation:**
- Add integration tests for the tension event pipeline (`TensionEventHook`, `ProcessEvent`)
- Add unit tests for `FieldAuthorization.go` rule functions (`isOwner`, `unique`, `oneByOne`, etc.)
- Add unit tests for `tension_auth.go` authorization hook checks
- Add tests for DQL template rendering (ensure `{{.nameid}}` substitution works correctly)
- Add tests for `codec/` encoding/decoding functions
- Consider a test harness with a test Dgraph instance for integration testing

**Benefit:** Currently, authorization logic changes have no safety net. The EMAP and directive system are central to security and correctness.

---

## 3. Replace Reflection with Type-Safe Patterns

**Problem:** The `meta()` function in `resolver.go` (lines 307-381) uses heavy reflection to:
- Extract field values from objects (`reflect.ValueOf(obj).Elem().FieldByName(...)`)
- Convert `[]map[string]interface{}` results to typed slices
- Dynamically dispatch based on return type (`reflect.TypeOf`, `reflect.MakeSlice`)

**Recommendation:**
- ~~Use Go generics (1.18+) for the type conversion layer~~ (Done: `DecodeDql[T]` generic in `internal/tools/dql_decode.go`)
- Register typed handlers per @meta field instead of relying on runtime type switching
- ~~Replace `Map2Struct` reflection with explicit struct mapping functions for known types (`Event`, `EventCount`)~~ (Done: `Map2Struct` now uses `json.Marshal/Unmarshal`, `meta()` uses `reflect.New` + JSON)
- ~~Migrate standalone `Map2Struct` callers to `DecodeDql[T]`~~ (Done: all call sites migrated; `Map2Struct` deleted)
- ~~Factorize `Meta1`/`Gamma1` into generic package-level `db.Meta[T]`/`db.Gamma[T]` + `First[T]`~~ (Done: `Meta1`/`Gamma1` deleted; all call sites migrated to `First(db.Meta[T](...))` or `First(db.Gamma[T](...))`; `First[T]` and `DecodeDql[T]` moved to `internal/tools/dql_decode.go`; `DecodeDqlSlice` removed — unified on `DecodeDql[[]T]`)

**Current state:** The DQL data mapping layer is now fully generic. The remaining reflection is in `meta()` (resolver.go) which dynamically dispatches `@meta` field results — this requires either per-field typed handlers or staying with reflection since `@meta` operates on schema-declared types unknown at compile time.

**Benefit:** Compile-time type safety, clearer error messages, better performance, easier debugging.

---


## 5. Structured Error Handling

**Problem:** Error handling uses `LogErr()` which captures stack traces but returns flat error strings. There is no error type hierarchy. Some operations fail silently (authorization checks return empty responses instead of errors). Error messages mix French and English.

**Recommendation:**
- Define sentinel errors for common cases: `ErrUnauthorized`, `ErrNotFound`, `ErrValidation`, `ErrConflict`
- Use `fmt.Errorf("...: %w", err)` wrapping consistently to maintain error chains
- Replace silent failures with explicit authorization errors where appropriate
- Use `errors.Is()` / `errors.As()` for error checking instead of string matching
- Standardize all error messages in English

**Benefit:** Enables proper error handling by callers, better observability, cleaner API error responses.

---

## 6. DQL Template Injection Safety

**Problem:** DQL queries use raw string template substitution (`RawFormat` with `text/template`):
```go
var(func: eq(Node.nameid, "{{.nameid}}"))
```
If a `nameid` value contains DQL metacharacters or quotes, this could produce malformed queries. While values largely come from internal sources (Dgraph UIDs, validated nameids), this is a fragile pattern.

**Recommendation:**
- Use Dgraph's parameterized query variables (`$nameid: string`) instead of string interpolation where possible
- For DQL queries that must use template substitution, add an explicit sanitization step
- Document which query parameters are user-controlled vs. system-generated

**Benefit:** Defense in depth against injection, clearer security boundaries.

---

## 7. Consolidate Resolver Hook Registration

**Problem:** `graph/resolver.go` `Init()` registers ~60+ hook directives manually. Many share the same handler (e.g., multiple `nothing` hooks, multiple `setContextWithID` hooks). Adding a new type requires adding entries in multiple places.

**Recommendation:**
- Group hooks by handler function using a registration helper:
  ```go
  registerHooks(c, setContextWithID,
      "Hook_addTensionInput", "Hook_updateTensionInput",
      "Hook_addContractInput", ...)
  ```
- Or use a declarative map:
  ```go
  hookMap := map[string]DirectiveFunc{
      "Hook_addTensionInput": setContextWithID,
      "Hook_addTension":      addTensionHook,
  }
  ```
- Consider code generation from annotations if the pattern grows further

**Benefit:** Reduces boilerplate, makes hook registration self-documenting, harder to miss a registration.

---

## 8. Extract Business Logic from Resolver Hooks

**Problem:** Resolver hook functions (e.g., `addTensionHook`, `updateContractHook`) mix GraphQL plumbing (context extraction, type assertions, directive chain) with business logic (event processing, authorization, notifications). This makes business logic hard to test in isolation.

**Recommendation:**
- Extract pure business logic into service-layer functions that accept typed parameters and return typed results
- Keep resolver hooks as thin adapters that extract data from context and delegate to service functions
- Example:
  ```go
  // Service layer (testable)
  func ProcessTensionEvents(uctx *model.UserCtx, tid string, events []*model.EventRef, blob *model.BlobRef) error

  // Hook (thin adapter)
  func addTensionHook(ctx, obj, next) (interface{}, error) {
      uctx := auth.GetUserContext(ctx)
      // ... extract data ...
      return ProcessTensionEvents(uctx, tid, events, blob)
  }
  ```

**Benefit:** Enables unit testing of business logic without GraphQL context, clearer separation of concerns.

---

## 9. Modernize Dependency Versions

**Problem:** Several dependencies are on older versions:
- `dgraph-io/dgo/v200` - Uses Dgraph's v200 client (legacy versioning)
- `go-redis/redis/v8` - v9 is current with context-first API
- `go.mod` declares Go 1.21 with toolchain 1.22 - could target 1.22 directly

**Recommendation:**
- Evaluate upgrading to `go-redis/redis/v9` for improved context handling
- Check if Dgraph client has a newer release compatible with current Dgraph version
- Set `go 1.22` directly in go.mod since that's the toolchain used
- Review all dependencies for security patches

**Benefit:** Security patches, performance improvements, access to newer Go features.

---

## 10. Add Request-Level Observability

**Problem:** Logging is basic (`fmt.Println` for DEV mode query names). Prometheus metrics exist (`web/handlers/instrumentation.go`) but there's no structured logging, no request tracing, and no visibility into DQL query performance.

**Recommendation:**
- Replace `fmt.Println` with structured logging (e.g., `slog` from stdlib in Go 1.21+)
- Add DQL query timing metrics (template name, duration, result count)
- Add trace IDs through the request lifecycle (already have `RequestID` middleware)
- Log authorization decisions (especially denials) for security auditing
- Add Dgraph gRPC connection pool metrics

**Benefit:** Production debugging, performance profiling, security audit trail.

---

## 11. Simplify the `meta()` Type Switch

**Problem:** The `meta()` function handles three return type categories (`*int`, slices, default structs) with nested reflection. The code is dense and hard to follow.

**Recommendation:**
- Register return type converters per @meta field name at init time:
  ```go
  metaConverters = map[string]func([]map[string]interface{}) (interface{}, error){
      "getNodeHistory": convertToEvents,
      "getEventCount":  convertToEventCount,
  }
  ```
- Each converter is a simple, testable function that knows its target type
- Fall back to reflection-based conversion for unregistered fields

**Benefit:** Debuggable, testable, explicit instead of reflective.

---

## 12. Reduce Hardcoded DQL Payload Strings

**Problem:** `db/dql.go` contains ~20 multi-line payload string variables (`userCtxPayload`, `tensionHookPayload`, `contractHookPayload`, etc.) that duplicate field selections from the schema. If the schema changes, these payloads must be updated manually.

**Recommendation:**
- Use `//go:embed` to load payloads from `.dql` files that can be validated
- Or generate payload strings from the schema during `make generate`
- At minimum, add a test that validates payload field names against the current schema

**Benefit:** Reduces drift between schema and DQL payloads, catches errors earlier.

---

## 13. Clean Up TODO/DEBUG Comments

**Problem:** The schema and code contain many `@DEBUG`, `@debug`, `@future`, `@obsolete`, and `@refactor` annotations, some dating back to early development:
- `@DEBUG: Waiting Nested filter in Dgraph` (multiple places in schema)
- `@debug: Aggregate count result` (User type)
- `@obsolete ?!` (NodeFragment fields)
- `@refactor: modularize generic function` (dql.go line 38)

**Recommendation:**
- Audit each annotation: resolve, convert to GitHub issues, or remove if obsolete
- Replace inline `@DEBUG` with tracked issues so they don't accumulate
- Remove `@obsolete` markers if the code is truly unused

**Benefit:** Cleaner codebase, tracked technical debt instead of scattered annotations.

---

## 14. Add Interface Abstractions for Database Layer

**Problem:** The `Resolver` struct holds a concrete `*db.Dgraph` pointer. All database operations go through concrete method calls. This tightly couples the GraphQL layer to Dgraph.

**Recommendation:**
- Define a `db.Store` interface with the core operations (`Query`, `Get`, `Add`, `Update`, `Delete`, `Meta`)
- Have `Dgraph` implement this interface
- Accept the interface in `Resolver` and hook functions
- This enables mock implementations for testing

**Benefit:** Testability, potential for alternative backends, cleaner dependency boundaries.

---

## 15. Schema Documentation

**Problem:** The GraphQL schema (`fractal6.graphql`) has minimal field documentation. Most types and fields lack `"""description"""` strings. The schema is the primary API contract with the frontend.

**Recommendation:**
- Add descriptions to all public types and their key fields
- Document enum values (especially `TensionEvent` - the most complex enum)
- Document authorization requirements per type (which roles can query/mutate)
- Generate API documentation from the annotated schema

**Benefit:** Self-documenting API, better developer experience for frontend and API consumers.

---

