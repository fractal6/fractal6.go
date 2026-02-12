# Resolver Refactoring Proposals

This document proposes improvements to simplify, unify, and improve the consistency of the resolver architecture in `fractal6.go`.

## Summary & Priority

| # | Proposal | Effort | Impact | Priority |
|---|----------|--------|--------|----------|
| 1 | Eliminate dual bridge system | Medium | High (removes fragile regex, unifies flow) | High |
| 4 | Replace Redis meta_patch hack | Low | High (fixes race condition) | High |
| 2 | Convention-over-configuration hooks | Low | Medium (reduces boilerplate) | Medium |
| 5 | Standardize resolver activation | Low | Medium (clarity + no panics) | Medium |
| 6 | Type-safe input extraction | Low | Medium (safety + consistency) | Medium |
| 7 | Decouple history cut hack | — | High (included in #1) | — |
| 3 | Generic artefact resolver | Medium | Medium (reduces duplication) | Low |
| 8 | Consistent error types | Medium | Medium (better UX) | Low |


---

## 1. Eliminate the Dual Bridge System

**Problem**: Two bridge patterns coexist — `DgraphQueryBridge`/`DgraphAddBridge`/etc. and the deprecated `DgraphBridgeRaw`. The raw bridge forwards the client's original query string and loses directive modifications, while the structured bridges properly reconstruct queries. Currently, `Get*` queries (GetNode, GetTension, GetUser, GetProject, GetContract) still use the raw bridge.

**Impact**: The raw bridge uses regex hacking to strip history (`regexp.MustCompile(`,?\s*history\s*:\s*\[...`)) and manual variable manipulation. This is fragile and bypasses the directive chain for input values.

**Proposal**:
- Implement a `DgraphGetBridge` (the stub exists but panics) that works like `DgraphQueryBridge` but for single-object lookups.
- Migrate all `DgraphBridgeRaw` call sites to structured bridges.
- Remove `DgraphQueryResolverRaw` entirely.

**Effort**: Medium. The `GetQueryGraph` infrastructure already exists; the main work is ensuring the `Get*` queries pass the right filter/ID parameters through the structured bridge.

---

## 2. Consolidate Hook Registration with Convention-over-Configuration

**Problem**: `resolver.go:Init()` registers ~60 hooks manually. Most are `nothing` (pass-through). Every new schema type requires adding 8-10 new lines even when no custom logic is needed.

```go
c.Directives.Hook_getProjectCardInput = nothing
c.Directives.Hook_queryProjectCardInput = nothing
c.Directives.Hook_addProjectCardInput = nothing
c.Directives.Hook_updateProjectCardInput = nothing
c.Directives.Hook_deleteProjectCardInput = nothing
c.Directives.Hook_addProjectCard = addProjectCardHook
c.Directives.Hook_updateProjectCard = updateProjectCardHook
c.Directives.Hook_deleteProjectCard = deleteProjectCardHook
```

**Proposal**:
- Set `nothing` as the default handler for all `@hook_` directives (requires a small gqlgen config change or a reflection-based initializer).
- Only register hooks that have actual custom logic.
- This would reduce `Init()` from ~170 lines to ~30 lines.

**Example** (using reflection to set defaults):
```go
func Init() gen.Config {
    c := gen.Config{Resolvers: &Resolver{db: db.GetDB()}}

    // Set all Hook_* fields to nothing by default
    setDefaultHooks(&c.Directives, nothing)

    // Only register custom hooks
    c.Directives.Hook_addTension = addTensionHook
    c.Directives.Hook_updateTension = updateTensionHook
    c.Directives.Hook_addLabel = addNodeArtefactHook
    // ... only ~20 custom hooks

    return c
}
```

**Effort**: Low. A simple reflection loop over the `Directives` struct fields whose names start with `Hook_`.

---

## 3. Unify the Artefact Resolver Pattern

**Problem**: Label, RoleExt, and Project share the same authorization pattern (artefacts linked to nodes, requiring coordinator auth) but use ad-hoc struct types (`AddArtefactInput`, `UpdateArtefactInput`, `FilterArtefactInput`) defined in `node_resolver.go` to handle them uniformly. These types partially duplicate the generated model types and rely on `ExtractInputs`/`ExtractInput` to bridge the gap.

**Proposal**:
- Define a generic `ArtefactHook[T]` interface or use Go generics to create a single reusable hook factory:

```go
func artefactAddHook[T interface{ GetNodes() []*model.NodeRef; GetRootnameid() string }](
    ctx context.Context, obj interface{}, next graphql.Resolver,
) (interface{}, error) {
    ctx, uctx, err := auth.GetUserContext(ctx)
    if err != nil {
        return nil, LogErr("Access denied", err)
    }
    var inputs []T
    ExtractInputs(ctx, &inputs)
    for _, input := range inputs {
        // ... shared auth logic
    }
    return next(ctx)
}
```

- This eliminates the `AddArtefactInput`/`UpdateArtefactInput` proxy types and the special-case `typeName` switching in `updateNodeArtefactHook`.

**Effort**: Medium. Requires defining interfaces on the generated model types (or adding methods via extensions).

---

## 4. Replace the Redis `@meta_patch` Hack

**Problem**: The `@w_meta_patch` directive stores function name/key/value in Redis with a 5-second TTL, then `postGqlProcess` retrieves and executes them. This is a race condition waiting to happen (concurrent requests from the same user could overwrite each other's Redis keys since keys are `username + "meta_patch_f"`).

**Proposal**:
- Pass the meta_patch operations through `context.Context` instead of Redis. Context is request-scoped and inherently safe for concurrent requests.

```go
type metaPatchOp struct {
    F string
    K string
    V string
}

func meta_patch(ctx context.Context, obj interface{}, next graphql.Resolver, f string, k *string) (interface{}, error) {
    op := metaPatchOp{F: f, K: *k, V: extractValue(ctx, obj, *k)}
    ctx = context.WithValue(ctx, "meta_patch_op", &op)
    return next(ctx)
}

func postGqlProcess(ctx context.Context, db *db.Dgraph, data interface{}, errors error) error {
    // ...
    if op, ok := ctx.Value("meta_patch_op").(*metaPatchOp); ok {
        db.Meta(op.F, map[string]string{op.K: op.V})
    }
    // ...
}
```

**Effort**: Low. Straightforward refactor, no external dependencies changed.

---

## 5. Standardize Resolver Activation

**Problem**: `schema.resolvers.go` mixes three patterns for resolver bodies:
1. `panic("not implemented")` — intentionally disabled operations
2. `DgraphAddBridge`/`DgraphUpdateBridge` calls — activated operations
3. Special one-off implementations (e.g., `QueryBuildInfo` returns a hardcoded version)

There's no easy way to see which operations are live vs. disabled without reading the entire file. Some disabled operations *should* work (e.g., `DeleteLabel`) but simply haven't been wired up.

**Proposal**:
- Add a comment header block listing all activated/disabled resolvers.
- Or, generate a registration table:

```go
// resolverRegistry defines which operations are exposed through the API.
// Operations not listed here will return "not implemented" errors.
var resolverRegistry = map[string]bool{
    "AddLabel":     true,
    "UpdateLabel":  true,
    "DeleteLabel":  false,  // TODO: implement cascade delete
    // ...
}
```

- Consider replacing panics with proper GraphQL errors (`return nil, fmt.Errorf("operation not supported")`) to avoid crashing the server on unexpected calls.

**Effort**: Low.

---

## 6. Type-Safe Input Extraction

**Problem**: Several resolver hooks extract inputs via untyped `graphql.GetResolverContext(ctx).Args["input"]` with type assertions that panic on failure:

```go
inputs := graphql.GetResolverContext(ctx).Args["input"].([]*model.AddTensionInput)
input := graphql.GetResolverContext(ctx).Args["input"].(model.UpdateTensionInput)
```

The generic helpers `ExtractInputs[T]` and `ExtractInput[T]` exist in `resolver.go` but aren't used consistently — `tension_resolver.go` and `contract_resolver.go` use raw type assertions while `node_resolver.go` uses the helpers.

**Proposal**:
- Use `ExtractInputs[T]`/`ExtractInput[T]` everywhere.
- Add error handling to these helpers (currently they panic via `ExtractSlice`/`StructMap`).

```go
// Before (tension_resolver.go):
inputs := graphql.GetResolverContext(ctx).Args["input"].([]*model.AddTensionInput)

// After:
var inputs []*model.AddTensionInput
ExtractInputs(ctx, &inputs)
```

**Effort**: Low. Mechanical replacement across ~5 files.

---

## 7. Decouple the History Cut Hack

**Problem**: Tension hooks use a `context.WithValue(ctx, "cut_history", true)` flag that is consumed deep in `DgraphQueryResolverRaw` to regex-strip the `history` field from the raw query. This is the most fragile part of the codebase — a regex operating on a GraphQL query string.

**Proposal**: This is solved automatically by Proposal 1 (eliminating `DgraphBridgeRaw`). With the structured bridge, the hook simply sets `input.Set.History = nil` before calling `next(ctx)`, and the bridge reconstructs the query without history. No regex needed.

**Effort**: Included in Proposal 1.

---

## 8. Consistent Error Handling Pattern

**Problem**: Error handling is inconsistent across resolver hooks:
- Some use `LogErr("Access denied", err)` (returns a wrapped error)
- Some use `fmt.Errorf("not implemented")` (bare error)
- Some return `(nil, err)`, others `(data, err)` allowing partial results

**Proposal**:
- Define domain error types:

```go
type AccessDeniedError struct{ Msg string }
type NotImplementedError struct{ Op string }
type ValidationError struct{ Field, Rule string }
```

- Use consistent wrapping:

```go
return nil, &AccessDeniedError{Msg: "coordinator rights required"}
```

- This enables the frontend to distinguish error types and show appropriate UI.

**Effort**: Medium. Requires defining error types and updating ~30 return sites.

---

