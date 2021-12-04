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
// Artefact Resolver (Label, RoleExt...)
////////////////////////////////////////////////

type AddArtefactInput struct {
	Nodes       []*model.NodeRef    `json:"nodes,omitempty"`
}

type UpdateArtefactInput struct {
	Set    *AddArtefactInput  `json:"set,omitempty"`
	Remove *AddArtefactInput  `json:"remove,omitempty"`
}


// Add "Artefeact" - Must be Coordo
func addNodeArtefactHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    // Get User context
    uctx, err := webauth.GetUserContext(ctx)
    if err != nil { return nil, LogErr("Access denied", err) }

    // Validate input
    var inputs []*AddArtefactInput
    StructMap(graphql.GetResolverContext(ctx).Args["input"], inputs)

    // Authorization
    // Check that user satisfy strict condition (coordo roles on node linked)
    mode := model.NodeModeCoordinated
    for _, input := range inputs {
        if len(input.Nodes) == 0 { return nil, LogErr("Access denied", fmt.Errorf("A node must be given.")) }
        node := input.Nodes[0]
        ok, err := auth.HasCoordoRole(uctx, *node.Nameid, &mode)
        if err != nil { return nil, LogErr("Internal error", err) }
        if !ok {
            return nil, LogErr("Access denied", fmt.Errorf("Contact a coordinator to access this ressource."))
        }
    }
    return next(ctx)
}

// Update "Artefact" - Must be coordo
func updateNodeArtefactHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    // Get User context
    uctx, err := webauth.GetUserContext(ctx)
    if err != nil { return nil, LogErr("Access denied", err) }

    // Validate input
    var input UpdateArtefactInput
    StructMap(graphql.GetResolverContext(ctx).Args["input"], input)

    var nodes []*model.NodeRef
    if input.Set != nil {
        if len(input.Set.Nodes) == 0 { return nil, LogErr("Access denied", fmt.Errorf("A node must be given.")) }
        nodes = append(nodes, input.Set.Nodes[0])
    }
    if input.Remove != nil {
        if len(input.Remove.Nodes) == 0 { return nil, LogErr("Access denied", fmt.Errorf("A node must be given.")) }
        nodes = append(nodes, input.Remove.Nodes[0])
    }

    // Authorization
    // Check that user satisfy strict condition (coordo roles on node linked)
    mode := model.NodeModeCoordinated
    for _, node := range nodes {
        ok, err := auth.HasCoordoRole(uctx, *node.Nameid, &mode)
        if err != nil { return nil, LogErr("Internal error", err) }
        if !ok {
            return nil, LogErr("Access denied", fmt.Errorf("Contact a coordinator to access this ressource."))
        }
    }
    return next(ctx)
}

