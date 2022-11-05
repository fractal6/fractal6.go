/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2022 Fractale Co
 *
 * This file is part of Fractale.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with Fractale.  If not, see <http://www.gnu.org/licenses/>.
 */

package graph

import (
	"context"
	"fmt"
	"github.com/99designs/gqlgen/graphql"

	//"fractale/fractal6.go/db"
	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph/auth"
	"fractale/fractal6.go/graph/codec"
	"fractale/fractal6.go/graph/model"
	. "fractale/fractal6.go/tools"
	webauth "fractale/fractal6.go/web/auth"
)

////////////////////////////////////////////////
// Node Resolver
////////////////////////////////////////////////

// ras

////////////////////////////////////////////////
// Artefact Resolver (Label, RoleExt...)
////////////////////////////////////////////////

type AddArtefactInput struct {
	Rootnameid  string              `json:"rootnameid,omitempty"`
	Nodes       []*model.NodeRef    `json:"nodes,omitempty"`
}

type FilterArtefactInput struct {
	ID         []string                                `json:"id,omitempty"`
	Rootnameid *model.StringHashFilter                       `json:"rootnameid,omitempty"`
	Name       *model.StringHashFilterStringTermFilter `json:"name,omitempty"`
}

type UpdateArtefactInput struct {
	Filter *FilterArtefactInput `json:"filter,omitempty"`
	Set    *AddArtefactInput    `json:"set,omitempty"`
	Remove *AddArtefactInput    `json:"remove,omitempty"`
}


// Add "Artefeact" - Must be Coordo
func addNodeArtefactHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    var ok bool =  false
    // Get User context
    ctx, uctx, err := webauth.GetUserContext(ctx)
    if err != nil { return nil, LogErr("Access denied", err) }

    // Validate input
    var inputs []*AddArtefactInput
    inputs_, _ := InterfaceSlice(graphql.GetResolverContext(ctx).Args["input"])
    for _, s:= range inputs_ {
        temp := AddArtefactInput{}
        StructMap(s, &temp)
        inputs = append(inputs, &temp)
    }

    // Authorization
    // - Check that user satisfy strict condition (coordo roles on node linked)
    // - Check that rootnameid comply with Nodes
    mode := model.NodeModeCoordinated
    for _, input := range inputs {
        if len(input.Nodes) == 0 { return nil, LogErr("Access denied", fmt.Errorf("A node must be given.")) }
        node := input.Nodes[0]
        rid, _ := codec.Nid2rootid(*node.Nameid)
        if rid != input.Rootnameid { return nil, LogErr("Access denied", fmt.Errorf("rootnameid and nameid do not match.")) }
        ok, err = auth.HasCoordoRole(uctx, *node.Nameid, &mode)
        if err != nil { return nil, LogErr("Internal error", err) }
        if !ok {
            return nil, LogErr("Access denied", fmt.Errorf("Contact a coordinator to access this ressource."))
        }
    }
    if ok { return next(ctx) }
    return nil, LogErr("Access denied", fmt.Errorf("Contact a coordinator to access this ressource."))
}

// Update "Artefact" - Must be coordo
func updateNodeArtefactHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
    var ok bool =  false
    // Get User context
    ctx, uctx, err := webauth.GetUserContext(ctx)
    if err != nil { return nil, LogErr("Access denied", err) }

    // Validate input
    var input UpdateArtefactInput
    StructMap(graphql.GetResolverContext(ctx).Args["input"], &input)

    var nodes []*model.NodeRef
    if input.Set != nil {
        // (@FUTURE contract) Lock update if artefact belongs to multiple nodes
        n_nodes := 0
        _, typeName, _, err := queryTypeFromGraphqlContext(ctx)
        if err != nil { return nil, LogErr("UpdateNodeArtefact", err) }
        if len(input.Filter.ID) > 0 {
            n_nodes = db.GetDB().Count(input.Filter.ID[0], typeName +".nodes")
        } else if input.Filter.Name.Eq != nil && input.Filter.Rootnameid.Eq != nil {
            n_nodes = db.GetDB().Count2(typeName+".name", *input.Filter.Name.Eq, typeName+".rootnameid", *input.Filter.Rootnameid.Eq, typeName+".nodes")
        } else {
            return nil, LogErr("Access denied", fmt.Errorf("invalid filter to query node artefact."))
        }
        if n_nodes > 1 {
            return nil, LogErr("Access denied", fmt.Errorf("This object belongs to more than one node, edition is locked. Edition is possible only if one node defines this object."))
        }

        if len(input.Set.Nodes) == 0 { return nil, LogErr("Access denied", fmt.Errorf("A node must be given.")) }
        nodes = append(nodes, input.Set.Nodes[0])
    }
    if input.Remove != nil {
        // @DEBUG: only allow nodes to be removed...
        if len(input.Remove.Nodes) == 0 { return nil, LogErr("Access denied", fmt.Errorf("A node must be given.")) }
        nodes = append(nodes, input.Remove.Nodes[0])
    }

    // Authorization
    // Check that user satisfy strict condition (coordo roles on node linked)
    mode := model.NodeModeCoordinated
    for _, node := range nodes {
        ok, err = auth.HasCoordoRole(uctx, *node.Nameid, &mode)
        if err != nil { return nil, LogErr("Internal error", err) }
        if !ok {
            return nil, LogErr("Access denied", fmt.Errorf("Contact a coordinator to access this ressource."))
        }
    }
    if ok {
        return next(ctx)
    }
    return nil, LogErr("Access denied", fmt.Errorf("Contact a coordinator to access this ressource."))
}

