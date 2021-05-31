package graph

import (
    "fmt"
    "context"
    "github.com/99designs/gqlgen/graphql"

    "zerogov/fractal6.go/graph/model"
    "zerogov/fractal6.go/db"
    webauth"zerogov/fractal6.go/web/auth"
    . "zerogov/fractal6.go/tools"
)


////////////////////////////////////////////////
// Tension Resolver
////////////////////////////////////////////////
//

// Add Tension - Hook
func addTensionHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    // Get User context
    uctx, err := webauth.GetUserContext(ctx)
    if err != nil { return nil, LogErr("Access denied", err) }

    // Validate Input
    inputs := graphql.GetResolverContext(ctx).Args["input"].([]*model.AddTensionInput)
    if len(inputs) != 1 {
        return nil, LogErr("add tension", fmt.Errorf("One and only one tension allowed."))
    }
    if !PayloadContains(ctx, "id") {
        return nil, LogErr("field missing", fmt.Errorf("id field is required in tension payload"))
    }
    input := inputs[0]

    // Execute query
    data, err := next(ctx)
    if err != nil { return nil, err }
    if data.(*model.AddTensionPayload) == nil {
        return nil, LogErr("add tension", fmt.Errorf("no tension added."))
    }
    id := data.(*model.AddTensionPayload).Tension[0].ID

    // Validate and process Blob Event
    ok, _,  err := tensionEventHook(uctx, id, input.History, nil)
    if !ok || err != nil {
        // Delete the tension just added
        e := db.GetDB().DeepDelete("tension", id)
        if e != nil { panic(e) }
    }
    if ok || err != nil {
        return data, err
    }
    return nil, LogErr("Access denied", fmt.Errorf("Contact a coordinator to access this ressource."))
}

// Update Tension - Hook
func updateTensionHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    // Get User context
    uctx, err := webauth.GetUserContext(ctx)
    if err != nil { return nil, LogErr("Access denied", err) }

    // Validate input
    input := graphql.GetResolverContext(ctx).Args["input"].(model.UpdateTensionInput)
    ids := input.Filter.ID
    if len(ids) != 1 {
        return nil, LogErr("update tension", fmt.Errorf("One and only one tension allowed."))
    }

    // Validate Event prior the mutation
    var bid *string
    var contract *model.Contract
    var ok bool
    if input.Set != nil {
        if len(input.Set.Blobs) > 0 {
            bid = input.Set.Blobs[0].ID
        }
        ok, contract, err = tensionEventHook(uctx, ids[0], input.Set.History, bid)
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

    return nil, LogErr("Access denied", fmt.Errorf("Input remove not implemented."))
}


