package graph

import (
    "fmt"
    "context"
    "github.com/99designs/gqlgen/graphql"

    "zerogov/fractal6.go/graph/model"
    "zerogov/fractal6.go/db"
    webauth "zerogov/fractal6.go/web/auth"
    . "zerogov/fractal6.go/tools"
)

////////////////////////////////////////////////
// Node Resolver
////////////////////////////////////////////////

// Update Node hook
func updateNodeHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    ctx, _, err := setContextWith(ctx, obj, "nameid")
    if err != nil { return nil, LogErr("Update node error", err) }
    return next(ctx)
}

////////////////////////////////////////////////
// Label Resolver
////////////////////////////////////////////////

// Add Label hook | Must pass all
// + chech that user rights
func addLabelHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    // Retrieve userCtx from token
    uctx, err := webauth.UserCtxFromContext(ctx)
    if err != nil { return nil, LogErr("Access denied", err) }

    // Get input
    data, err := next(ctx)
    if err != nil { return nil, err }

    // Validate input
    input := data.([]*model.AddLabelInput)
    if len(input) != 1 {
        return nil, LogErr("Add label error", fmt.Errorf("Only one label supported in input."))
    }
    label := input[0]

    // Check rights
    if len(label.Nodes) != 1 { return nil, LogErr("Input error", fmt.Errorf("One and only one circle is required.")) }
    nid := label.Nodes[0].Nameid
    charac := GetNodeCharacStrict()
    ok, err := CheckUserRights(uctx, *nid, &charac)
    if err != nil { return nil, LogErr("Internal error", err) }
    if ok {
        return data, err
    }

    return nil, LogErr("Access denied", fmt.Errorf("Contact a coordinator to access this ressource."))
}

// Update Label hook | Must pass all
// + check the user rights
func updateLabelHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    // Retrieve userCtx from token
    uctx, err := webauth.UserCtxFromContext(ctx)
    if err != nil { return nil, LogErr("Access denied", err) }

    // Validate input
    var isNew bool
    ctx, lid, err := setContextWith(ctx, obj, "id")
    if err != nil {
        return nil, LogErr("Update tension error", err)
    }
    if lid == "" { // assumes rootnameid and name are given
        filter := obj.(model.JsonAtom)["input"].(model.JsonAtom)["filter"].(model.JsonAtom)
        rootnameid := getNestedObj(filter, "rootnameid.eq").(string)
        name := getNestedObj(filter, "name.eq").(string)
        filterName := "Label.rootnameid"
        ids, err := db.GetDB().GetIDs("Label.name", name, &filterName, &rootnameid)
        if err != nil { return nil, LogErr("Internal error", err) }
        lid = ids[0]
        isNew = true
    }

    ctx = context.WithValue(ctx, "id", lid)

    // Get input
    data, err := next(ctx)
    if err != nil { return nil, err }
    input := data.(model.UpdateLabelInput)

    // Check rights
    charac := GetNodeCharacStrict()
    nodes, err := db.GetDB().GetSubFieldById(lid, "Label.nodes", "Node.nameid")
    if err != nil { return nil, LogErr("Internal error", err) }

    // Add label a in a Circle
    if isNew {
        // Similar than AddLabel
        if len(input.Set.Nodes) != 1 { return nil, LogErr("Input error", fmt.Errorf("One circle required.")) }
        nid := input.Set.Nodes[0].Nameid
        ok, err := CheckUserRights(uctx, *nid, &charac)
        if err != nil { return nil, LogErr("Internal error", err) }
        if ok {
            return data, err
        }

    } else if nodes == nil {
        return nil, LogErr("Access denied", fmt.Errorf("You do not own this ressource."))
    }

    // Update label
    for _, nid := range nodes.([]interface{}) {
        ok, err := CheckUserRights(uctx, nid.(string), &charac)
        if err != nil { return nil, LogErr("Internal error", err) }
        if ok {
            if obj.(model.JsonAtom)["input"].(model.JsonAtom)["remove"] != nil {
                // user is removing node
                fmt.Println("remove label here !!! check if removed from tension")
            }
            //return data, err
            return next(ctx)
        }
    }

    return nil, LogErr("Access denied", fmt.Errorf("You do not own this ressource."))
}

