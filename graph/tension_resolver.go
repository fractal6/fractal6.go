package graph

import (
    "fmt"
    "context"
    "github.com/99designs/gqlgen/graphql"

    "fractale/fractal6.go/graph/model"
    "fractale/fractal6.go/db"
    webauth"fractale/fractal6.go/web/auth"
    . "fractale/fractal6.go/tools"
)


////////////////////////////////////////////////
// Tension Resolver
////////////////////////////////////////////////


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

    // History and notification Logics --
    // In order to notify user on the given event, we need to know their ids to pass and link them
    // to the notification (UserEvent edge) function. To do so we first cut the history from the original
    // input, and push then the history (see the PushHistory function).
    ctx = context.WithValue(ctx, "cut_history", true)
    // Execute query
    data, err := next(ctx)
    if err != nil { return data, err }
    if data.(*model.AddTensionPayload) == nil {
        return nil, LogErr("add tension", fmt.Errorf("no tension added."))
    }
    tension := data.(*model.AddTensionPayload).Tension[0]
    id := tension.ID

    // Validate and process Blob Event
    ok, _,  err := tensionEventHook(uctx, id, input.History, nil)
    if !ok || err != nil {
        // Delete the tension just added
        e := db.GetDB().DeepDelete("tension", id)
        if e != nil { panic(e) }
    }
    if err != nil {
        return data, err
    }
    if ok {
        err = PushHistory(uctx, id, input.History)
        e := PushEventNotifications(id, input.History)
        if e != nil { panic(e) }
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
    var blob *model.BlobRef
    var contract *model.Contract
    var ok bool
    if input.Set != nil {
        if len(input.Set.Blobs) > 0 {
            blob = input.Set.Blobs[0]
        }
        ok, contract, err = tensionEventHook(uctx, ids[0], input.Set.History, blob)
        if err != nil { return nil, err }
        if ok {
            // History and notification Logics --
            // In order to notify user on the given event, we need to know their ids to pass and link them
            // to the notification (UserEvent edge) function. To do so we first cut the history from the original
            // input, and push then the history (see the PushHistory function).
            ctx = context.WithValue(ctx, "cut_history", true)
            // Execute query
            data, err := next(ctx)
            if err != nil { return data, err }
            err = PushHistory(uctx, ids[0], input.Set.History)
            e := PushEventNotifications(ids[0], input.Set.History)
            if e != nil { panic(err) }
            return data, err
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


