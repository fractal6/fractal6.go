package graph

import (
    "fmt"
    "context"
    "github.com/99designs/gqlgen/graphql"

    "zerogov/fractal6.go/graph/model"
    "zerogov/fractal6.go/graph/auth"
    webauth "zerogov/fractal6.go/web/auth"
    . "zerogov/fractal6.go/tools"
)

////////////////////////////////////////////////
// Node Resolver
////////////////////////////////////////////////

// ras

////////////////////////////////////////////////
// Label Resolver
////////////////////////////////////////////////

// Add Label - Must be Coordo
func addLabelHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    // Get User context
    uctx, err := webauth.GetUserContext(ctx)
    if err != nil { return nil, LogErr("Access denied", err) }

    // Validate input
    inputs := graphql.GetResolverContext(ctx).Args["input"].([]*model.AddLabelInput)
    // Authorization
    // Check that user satisfy strict condition (coordo roles on node linked)
    charac := GetNodeCharacStrict()
    for _, input := range inputs {
        if len(input.Nodes) == 0 { return nil, LogErr("Access denied", fmt.Errorf("A node must be given.")) }
        node := input.Nodes[0]
        ok, err := auth.HasCoordoRole(uctx, *node.Nameid, &charac)
        if err != nil { return nil, LogErr("Internal error", err) }
        if !ok {
            return nil, LogErr("Access denied", fmt.Errorf("Contact a coordinator to access this ressource."))
        }
    }
    return next(ctx)
}

// Update Label - Must be coordo
func updateLabelHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    // Get User context
    uctx, err := webauth.GetUserContext(ctx)
    if err != nil { return nil, LogErr("Access denied", err) }

    // Validate input
    input := graphql.GetResolverContext(ctx).Args["input"].(model.UpdateLabelInput)
    var nodes []*model.NodeRef
    if input.Set != nil {
        if len(input.Set.Nodes) == 0 { return nil, LogErr("Access denied", fmt.Errorf("A node must be given.")) }
        nodes = append(nodes,  input.Set.Nodes[0])
    }
    if input.Remove != nil {
        if len(input.Remove.Nodes) == 0 { return nil, LogErr("Access denied", fmt.Errorf("A node must be given.")) }
        nodes = append(nodes,  input.Remove.Nodes[0])
    }

    // Authorization
    // Check that user satisfy strict condition (coordo roles on node linked)
    charac := GetNodeCharacStrict()
    for _, node := range nodes {
        ok, err := auth.HasCoordoRole(uctx, *node.Nameid, &charac)
        if err != nil { return nil, LogErr("Internal error", err) }
        if !ok {
            return nil, LogErr("Access denied", fmt.Errorf("Contact a coordinator to access this ressource."))
        }
    }
    return next(ctx)
}

