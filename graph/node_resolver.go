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

//
// Node Resolver
//

// Update Node hook
// - add the nameid field in the context for further inspection in new resolver
func updateNodeHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    filter := obj.(model.JsonAtom)["input"].(model.JsonAtom)["filter"].(model.JsonAtom)
    nameid_ := filter["nameid"].(model.JsonAtom)["eq"]
    if nameid_ != nil {
        ctx = context.WithValue(ctx, "nameid", nameid_.(string))
    }

    return next(ctx)
}

//
// Label Resolver
//

// Add Label hook | Must pass all
// + chech that user rights
func addLabelHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    // Retrieve userCtx from token
    uctx, err := auth.UserCtxFromContext(ctx)
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
    if len(label.Nodes) != 1 { return nil, LogErr("Input error", fmt.Errorf("One circle required.")) }
    nid := label.Nodes[0].Nameid
    charac := GetNodeCharacStrict()
    ok, err := CheckUserRights(uctx, *nid, &charac)
    if err != nil { return nil, LogErr("Internal error", err) }
    if ok {
        return next(ctx)
    }

    return nil, LogErr("Access denied", fmt.Errorf("Contact a coordinator to access this ressource."))
}

// Update Label hook | Must past all
// + check the user rights
func updateLabelHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    // Retrieve userCtx from token
    uctx, err := auth.UserCtxFromContext(ctx)
    if err != nil { return nil, LogErr("Access denied", err) }

    // Validate input
    filter := obj.(model.JsonAtom)["input"].(model.JsonAtom)["filter"].(model.JsonAtom)
    ids := filter["id"].([]interface{})
    if len(ids) != 1 {
        return nil, LogErr("Update label error", fmt.Errorf("Only one label supported in input."))
    }
    lid := ids[0].(string)
    ctx = context.WithValue(ctx, "id", lid)

    nodes, err := db.GetDB().GetSubFieldById(lid, "Label.nodes", "Node.nameid")
    if err != nil { return nil, LogErr("Internal error", err) }
    if nodes == nil { return nil, LogErr("Access denied", fmt.Errorf("You do not own this ressource.")) }
    fmt.Println(nodes)
    charac := GetNodeCharacStrict()
    for _, nid := range nodes.([]interface{}) {
        ok, err := CheckUserRights(uctx, nid.(string), &charac)
        if err != nil { return nil, LogErr("Internal error", err) }
        if ok { return next(ctx) }
    }

    return nil, LogErr("Access denied", fmt.Errorf("You do not own this ressource."))
}

