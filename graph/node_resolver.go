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

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph/codec"
	"fractale/fractal6.go/graph/model"
	. "fractale/fractal6.go/tools"
	"fractale/fractal6.go/web/auth"
)

////////////////////////////////////////////////
// Node Resolver
////////////////////////////////////////////////

// ras

////////////////////////////////////////////////
// Artefact Resolver (Label, RoleExt...)
////////////////////////////////////////////////

type AddArtefactInput struct {
	Name       *string          `json:"name"`
	Color      *string          `json:"color"`
	Rootnameid string           `json:"rootnameid,omitempty"`
	Nodes      []*model.NodeRef `json:"nodes,omitempty"`
}

type FilterArtefactInput struct {
	ID         []string                `json:"id,omitempty"`
	Rootnameid *model.StringHashFilter `json:"rootnameid,omitempty"`
	// For Project Only
	Parentnameid *model.StringHashFilter `json:"parentnameid,omitempty"`
	Nameid       *model.StringHashFilter `json:"nameid,omitempty"`
	// --
	Name *model.StringHashFilterStringTermFilter `json:"name,omitempty"`
}

type UpdateArtefactInput struct {
	Filter *FilterArtefactInput `json:"filter,omitempty"`
	Set    *AddArtefactInput    `json:"set,omitempty"`
	Remove *AddArtefactInput    `json:"remove,omitempty"`
}

// Add "Artefeact"
func addNodeArtefactHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
	// Validate input
	var inputs []*AddArtefactInput
	inputs_, _ := InterfaceSlice(graphql.GetResolverContext(ctx).Args["input"])
	for _, s := range inputs_ {
		temp := AddArtefactInput{}
		StructMap(s, &temp)
		inputs = append(inputs, &temp)
	}

	// Authorization
	// - Check that rootnameid comply with Nodes
	// - nodes is required
	for _, input := range inputs {
		if len(input.Nodes) == 0 {
			return nil, LogErr("Access denied", fmt.Errorf("A node must be given."))
		}
		node := input.Nodes[0]
		rootnameid, _ := codec.Nid2rootid(*node.Nameid)
		if rootnameid != input.Rootnameid {
			return nil, LogErr("Access denied", fmt.Errorf("rootnameid and nameid does not match."))
		}
	}
	return next(ctx)
}

// Update "Artefact" - Must be coordo
func updateNodeArtefactHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
	var ok bool = false
	// Protected Object has more restrivive conditions to be updated.
	// @TODO: Clarify how the ressources access policy for artefacts object (that can belongs to multiple nodes)
	// Ex: { Leaders: (mandate acess, tension access), Coordinators: (artefact access, tension access) }?
	protecteds := []string{"Label", "RoleExt"}
	isProtected := false
	_, typeName, _, err := queryTypeFromGraphqlContext(ctx)
	for _, obj := range protecteds {
		if typeName == obj {
			isProtected = true
			break
		}
	}

	// Get User context
	ctx, uctx, err := auth.GetUserContext(ctx)
	if err != nil {
		return nil, LogErr("Access denied", err)
	}

	// Validate input
	var input UpdateArtefactInput
	StructMap(graphql.GetResolverContext(ctx).Args["input"], &input)

	nodes := []string{}
	nodesGiven := []*model.NodeRef{}
	if input.Set != nil {
		nodesGiven = append(nodesGiven, input.Set.Nodes...)
		if isProtected {
			var x interface{}
			var err error
			if len(input.Filter.ID) > 0 { // Updates with UID
				x, err = db.GetDB().GetSubFieldById(input.Filter.ID[0], typeName+".nodes", "Node.nameid")
			} else { // Update from hash names
				if typeName == "Project" && input.Filter.Parentnameid.Eq != nil && input.Filter.Nameid.Eq != nil {
					// Project like artefacts
					x, err = db.GetDB().GetSubFieldByEq2(typeName+".nameid", *input.Filter.Nameid.Eq, typeName+".parentnameid", *input.Filter.Parentnameid.Eq, typeName+".nodes", "Node.nameid")
				} else if input.Filter.Name.Eq != nil && input.Filter.Rootnameid.Eq != nil {
					// Other Artefacts update from hash names
					x, err = db.GetDB().GetSubFieldByEq2(typeName+".name", *input.Filter.Name.Eq, typeName+".rootnameid", *input.Filter.Rootnameid.Eq, typeName+".nodes", "Node.nameid")
				} else {
					return nil, LogErr("Access denied", fmt.Errorf("invalid filter to update node artefact."))
				}
			}

			if err != nil {
				return nil, err
			} else if x != nil {
				for _, n := range x.([]interface{}) {
					nodes = append(nodes, n.(string))
				}
			}

			if len(nodes) > 1 {
				// Only allow if user has auth in the node with the shortest path.
				rootnameid, _ := codec.Nid2rootid(nodes[0])
				best_node := ""
				best_weight := -1.0
				for _, n := range nodes {
					w, err := db.GetDB().GetShortestPath(rootnameid, n)
					if err != nil {
						return nil, err
					}
					if w < best_weight || best_weight < 0 {
						best_weight = w
						best_node = n
					}
				}
				// Check auth on the higher circle
				ok, err = auth.HasCoordoAuth(uctx, best_node, nil)
				if err != nil {
					return nil, LogErr("Internal error", err)
				} else if !ok {
					return nil, LogErr("Access denied", fmt.Errorf("you need the be a coordinator of the highest circle that use this ressource to update it."))
				}

			}
		}
	}

	if input.Remove != nil {
		// @DEBUG: only allow nodes to be removed...
		if len(input.Remove.Nodes) == 0 {
			return nil, LogErr("Access denied", fmt.Errorf("A node must be given."))
		}
		nodesGiven = append(nodesGiven, input.Remove.Nodes...)
	}

	// Authorization with regards to nodes attributes..
	// Check that user satisfy strict condition (coordo roles on node linked)
	mode := model.NodeModeCoordinated
	for _, node := range nodesGiven {
		// @TODO: check that rootnameid matches Label.rootnameid
		ok, err = auth.HasCoordoAuth(uctx, *node.Nameid, &mode)
		if err != nil {
			return nil, LogErr("Internal error", err)
		}
		if !ok {
			return nil, LogErr("Access denied", fmt.Errorf("Contact a coordinator to access this ressource."))
		}
	}

	// Update the Label event in tension history as data is hardocoded on new/old value.
	// @debug/perf: run this asynchronously
	if typeName == "Label" && len(input.Filter.ID) > 0 && input.Set != nil && (input.Set.Name != nil || input.Set.Color != nil) {
		old := struct{ Name, Color, Rootnameid string }{}
		new := struct{ Name, Color string }{}
		// Old value -- Color is embeded in the event new/old value
		old_, err := db.GetDB().GetFieldById(input.Filter.ID[0], "Label.name Label.color Label.rootnameid")
		if err != nil {
			return nil, LogErr("Internal error", err)
		}
		StructMap(old_, &old)

		// New value
		new_name := input.Set.Name
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
			"rootnameid": old.Rootnameid,
			"old_name":   old.Name + "§" + old.Color,
			"new_name":   new.Name + "§" + new.Color,
		})
		if err != nil {
			return nil, LogErr("Internal error", err)
		}
	}

	return next(ctx)
}
