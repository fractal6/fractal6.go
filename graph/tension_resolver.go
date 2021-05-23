package graph

import (
    "fmt"
    "context"
    "github.com/99designs/gqlgen/graphql"

    "zerogov/fractal6.go/graph/model"
    "zerogov/fractal6.go/graph/auth"
    "zerogov/fractal6.go/db"
    webauth"zerogov/fractal6.go/web/auth"
    . "zerogov/fractal6.go/tools"
)

////////////////////////////////////////////////
// Tension Resolver
////////////////////////////////////////////////

// Add Tension Hook
func addTensionHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    // Retrieve userCtx from token
    uctx, err := webauth.UserCtxFromContext(ctx)
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
    if !auth.UserPlayRole(uctx, emitterid) {
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

// Update Tension hook
func updateTensionHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    ctx, _, err := setContextWith(ctx, obj, "id")
    if err != nil { return nil, LogErr("Update tension error", err) }
    return next(ctx)
}

// Add Tension Post Hook
func addTensionPostHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    // Retrieve userCtx from token
    uctx, err := webauth.UserCtxFromContext(ctx)
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
    ok, _,  err := tensionEventHook(uctx, tid, input.History, nil)
    if err != nil || !ok {
        // Delete the tension just added
        e := db.GetDB().DeepDelete("tension", tid)
        if e != nil { panic(e) }
    }

    if err != nil  { return nil, err }
    if ok { return data, err }

    return nil, LogErr("Access denied", fmt.Errorf("Contact a coordinator to access this ressource."))
}

// Update Tension Post Hook
func updateTensionPostHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    // Retrieve userCtx from token
    uctx, err := webauth.UserCtxFromContext(ctx)
    if err != nil { return nil, LogErr("Access denied", err) }

    // Validate input
    rc := graphql.GetResolverContext(ctx)
    input := rc.Args["input"].(model.UpdateTensionInput)
    tids := input.Filter.ID
    if len(tids) == 0 {
        return nil, LogErr("field missing", fmt.Errorf("id field is required in tension filter."))
    }

    // Validate Blob Event prior the mutation
    var bid *string
    var contract *model.Contract
    var ok bool
    if input.Set != nil  {
        if len(input.Set.Blobs) > 0 {
            bid = input.Set.Blobs[0].ID
        }
        ok, contract, err = tensionEventHook(uctx, tids[0], input.Set.History, bid)
        if err != nil  { return nil, err }
        if ok {
            return next(ctx)
        } else if contract != nil {
            var t model.UpdateTensionPayload
            t.Tension = []*model.Tension{&model.Tension{
                Contracts: []*model.Contract{contract},
            }}
            return &t, err
        } else {
            return nil, LogErr("Access denied", fmt.Errorf("Contact a coordinator to access this ressource."))
        }
    }

    return next(ctx)
}

////////////////////////////////////////////////
// Comment Resolver
////////////////////////////////////////////////

// Update Comment hook
func updateCommentHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    ctx, _, err := setContextWith(ctx, obj, "id")
    if err != nil { return nil, LogErr("Update comment error", err) }
    return next(ctx)
}

////////////////////////////////////////////////
// Contract Resolver
////////////////////////////////////////////////

// -- todo auth method for contract mutations
// -- user @alter_hasRole ?

// Add Contract hook
func addContractHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    // * Get the tid or error
    // * get the tensionHook
    // * call ProcessEvent(tid, nil) (in tension_op)
    return next(ctx)
}

// Update Contract hook
func updateContractHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    ctx, _, err := setContextWith(ctx, obj, "id")
    if err != nil { return nil, LogErr("Update contract error", err) }
    return next(ctx)
}

// Delete Contract hook
func deleteContractHookPost(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    // Retrieve userCtx from token
    _, err := webauth.UserCtxFromContext(ctx)
    if err != nil { return nil, LogErr("Access denied", err) }

    // Validate input
    rc := graphql.GetResolverContext(ctx)
    filter := rc.Args["filter"].(model.ContractFilter)
    ids := filter.ID
    if len(ids) != 1 {
        return nil, LogErr("delete contract error", fmt.Errorf("Only one contract supported in input."))
    }

    // Deep delete
    err = db.GetDB().DeepDelete("contract", ids[0], )
    if err != nil { return nil, LogErr("Delete contract error", err) }

    var d model.DeleteContractPayload
    d.Contract = []*model.Contract{&model.Contract{ID: ids[0]}}
    return &d, err

    //data, err := next(ctx)
    //if err != nil { return nil, err }
    //return data, err
}
