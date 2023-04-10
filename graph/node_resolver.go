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
    "reflect"
	"fmt"
	"github.com/99designs/gqlgen/graphql"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/web/auth"
	"fractale/fractal6.go/graph/codec"
	"fractale/fractal6.go/graph/model"
	. "fractale/fractal6.go/tools"
)

////////////////////////////////////////////////
// Node Resolver
////////////////////////////////////////////////

// ras

////////////////////////////////////////////////
// Artefact Resolver (Label, RoleExt...)
////////////////////////////////////////////////

type AddArtefactInput struct {
    Name        *string             `json:"name"`
    Color       *string             `json:"color"`
	Rootnameid  string              `json:"rootnameid,omitempty"`
	Nodes       []*model.NodeRef    `json:"nodes,omitempty"`
}

type FilterArtefactInput struct {
	ID         []string                                `json:"id,omitempty"`
	Rootnameid *model.StringHashFilter                 `json:"rootnameid,omitempty"`
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
    ctx, uctx, err := auth.GetUserContext(ctx)
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
        rootnameid, _ := codec.Nid2rootid(*node.Nameid)
        if rootnameid != input.Rootnameid { return nil, LogErr("Access denied", fmt.Errorf("rootnameid and nameid does not match.")) }
        ok, err = auth.HasCoordoAuth(uctx, *node.Nameid, &mode)
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
    ctx, uctx, err := auth.GetUserContext(ctx)
    if err != nil { return nil, LogErr("Access denied", err) }

    // Validate input
    var input UpdateArtefactInput
    StructMap(graphql.GetResolverContext(ctx).Args["input"], &input)

    var typeName string
    var rootnameid string
    var nodes []*model.NodeRef
    if input.Set != nil {
        if len(input.Set.Nodes) == 0 { return nil, LogErr("Access denied", fmt.Errorf("A node must be given.")) }
        nodes = append(nodes, input.Set.Nodes[0])
        rootnameid, _ = codec.Nid2rootid(*nodes[0].Nameid)

        // (@FUTURE contract) Lock update if artefact belongs to multiple nodes
        n_nodes := 0
        _, typeName, _, err = queryTypeFromGraphqlContext(ctx)
        if err != nil { return nil, LogErr("UpdateNodeArtefact", err) }
        if len(input.Filter.ID) > 0 {
            n_nodes = db.GetDB().Count(input.Filter.ID[0], typeName +".nodes")
        } else if input.Filter.Name.Eq != nil && input.Filter.Rootnameid.Eq != nil {
            if rootnameid != *input.Filter.Rootnameid.Eq { return nil, LogErr("Access denied", fmt.Errorf("rootnameid and nameid do not match.")) }
            n_nodes = db.GetDB().Count2(typeName+".name", *input.Filter.Name.Eq, typeName+".rootnameid", *input.Filter.Rootnameid.Eq, typeName+".nodes")
        } else {
            return nil, LogErr("Access denied", fmt.Errorf("invalid filter to query node artefact."))
        }

        if n_nodes > 1 && *nodes[0].Nameid != rootnameid  {
            // Instanciate an empty empty object of the same type than input.Set
            t := reflect.TypeOf(input.Set).Elem()
            a := reflect.New(t).Elem().Interface()
            b := *input.Set
            b.Nodes = nil
            // Ignore if the update it is just appending the data to new node (not actually modifing it)
            if !reflect.DeepEqual(a, b) {
                return nil, LogErr("Access denied", fmt.Errorf("This object belongs to more than one node, edition is locked. Edition is only possible at the root circle level."))
            }
        }
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
        ok, err = auth.HasCoordoAuth(uctx, *node.Nameid, &mode)
        if err != nil { return nil, LogErr("Internal error", err) }
        if !ok {
            return nil, LogErr("Access denied", fmt.Errorf("Contact a coordinator to access this ressource."))
        }
    }
    if !ok {
        return nil, LogErr("Access denied", fmt.Errorf("Contact a coordinator to access this ressource."))
    }

    // Update the Label event in tension history as data id hardocoded on new/old value.
    // @debug/perf: run this asynchronously
    if typeName == "Label" && input.Set != nil && len(input.Filter.ID) > 0 {
        old := struct { Name, Color string }{}
        new := struct { Name, Color string }{}
        // Old value -- Color is embeded in the event new/old value
        old_, err := db.GetDB().GetFieldById(input.Filter.ID[0], "Label.name Label.color")
        if err != nil { return nil, LogErr("Internal error", err) }
        StructMap(old_, &old)

        // New value
        new_name :=  input.Set.Name
        new_color := input.Set.Color
        if new_name == nil {
            new.Name = old.Name
        } else {
           new.Name = *new_name
        }
        if new_color == nil {
            new.Color = old.Color
        } else {
           new.Color = *new_color
        }

        // Rewrite
        _, err = db.GetDB().Meta("rewriteLabelEvents", map[string]string{
            "rootnameid": rootnameid,
            "old_name": old.Name + "§" + old.Color,
            "new_name": new.Name + "§" + new.Color,
        })
        if err != nil { return nil, LogErr("Internal error", err) }
    }

    return next(ctx)
}

