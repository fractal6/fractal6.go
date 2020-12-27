package graph

import (
    "fmt"
    "context"
    "github.com/99designs/gqlgen/graphql"

    "zerogov/fractal6.go/graph/model"
    "zerogov/fractal6.go/web/auth"
    "zerogov/fractal6.go/db"
    . "zerogov/fractal6.go/tools"
)

// Add Tension Hook
func addTensionHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    // Retrieve userCtx from token
    uctx, err := auth.UserCtxFromContext(ctx)
    if err != nil { return nil, LogErr("Access denied", err) }

    // Get input
    data, err := next(ctx)
    if err != nil { return nil, err }

    // Validate input
    input := data.([]*model.AddTensionInput)
    if len(input) != 1 {
        return nil, LogErr("Add tension error", fmt.Errorf("Only one tension supported in input."))
    }

    // Check that user as the given emitter role
    emitterid := input[0].Emitterid
    if !userPlayRole(uctx, emitterid) {
        // if not check for bot access
        r_, err := db.GetDB().GetFieldByEq("Node.nameid", emitterid, "Node.role_type")
        if err != nil { return nil, LogErr("Internal error", err) }
        isArchived, err := db.GetDB().GetFieldByEq("Node.nameid", emitterid, "Node.isArchived")
        if err != nil { return nil, LogErr("Internal error", err) }
        if r_ == nil { return data, err }
        if model.RoleType(r_.(string)) != model.RoleTypeBot || isArchived.(bool) {
            return nil, LogErr("Access denied", fmt.Errorf("you do not own this node"))
        }
    }

    return data, err
}

// Update Tension hook:
// * add the id field in the context for further inspection in new resolver
func updateTensionHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    filter := obj.(model.JsonAtom)["input"].(model.JsonAtom)["filter"].(model.JsonAtom)
    ids := filter["id"].([]interface{})
    if len(ids) != 1 {
        return nil, LogErr("Update tension error", fmt.Errorf("Only one tension supported in input."))
    }

    ctx = context.WithValue(ctx, "id", ids[0].(string))
    return next(ctx)
}

// Add Tension Post Hook
func addTensionPostHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    // Retrieve userCtx from token
    uctx, err := auth.UserCtxFromContext(ctx)
    if err != nil { return nil, LogErr("Access denied", err) }

    // Get Input
    data, err := next(ctx)
    if err != nil { return nil, err }

    // Validate input
    rc := graphql.GetResolverContext(ctx)
    input := rc.Args["input"].([]*model.AddTensionInput)[0]
    tid := data.(*model.AddTensionPayload).Tension[0].ID
    if tid == "" {
        return nil, LogErr("field missing", fmt.Errorf("id field is required in tension payload"))
    }

    // Validate and process Blob Event
    ok, err := tensionEventHook(uctx, tid, input.History, nil)
    if err != nil || !ok {
        // Delete the tension just added
        e := db.GetDB().DeleteNodes(tid)
        if e != nil { panic(e) }
    }

    if err != nil  { return nil, err }
    if ok { return data, err }

    return nil, LogErr("Access denied", fmt.Errorf("Contact a coordinator to access this ressource."))
}

// Update Tension Post Hook
func updateTensionPostHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    // Retrieve userCtx from token
    uctx, err := auth.UserCtxFromContext(ctx)
    if err != nil { return nil, LogErr("Access denied", err) }

    // Validate input
    rc := graphql.GetResolverContext(ctx)
    input := rc.Args["input"].(model.UpdateTensionInput)
    tids := input.Filter.ID
    if len(tids) == 0 {
        return nil, LogErr("field missing", fmt.Errorf("id field is required in tension filter."))
    }

    // Validate Blob Event
    var bid *string
    if input.Set != nil  {
        if len(input.Set.Blobs) > 0 {
            bid = input.Set.Blobs[0].ID
        }
        ok, err := tensionEventHook(uctx, tids[0], input.Set.History, bid)
        if err != nil  { return nil, err }
        if ok {
            return next(ctx)
        } else {
            return nil, LogErr("Access denied", fmt.Errorf("Contact a coordinator to access this ressource."))
        }
    }

    return next(ctx)
}

// Update Comment hook
// * add the id field in the context for further inspection in new resolver
func updateCommentHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    filter := obj.(model.JsonAtom)["input"].(model.JsonAtom)["filter"].(model.JsonAtom)
    ids := filter["id"].([]interface{})
    if len(ids) != 1 {
        return nil, LogErr("Update comment error", fmt.Errorf("Only one comment supported in input."))
    }

    ctx = context.WithValue(ctx, "id", ids[0].(string))
    return next(ctx)
}
