/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2026 Fractale Co
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
	"slices"
	"strings"

	"github.com/99designs/gqlgen/graphql"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph/codec"
	"fractale/fractal6.go/graph/model"
	. "fractale/fractal6.go/internal/tools"
	"fractale/fractal6.go/web/auth"
)

// sepRune is the descriptor separator used in tension history events
// (Label "{name}§{color}", Project "{id}§{name}§", ProjectColumn
// "{id}§{name}§{color}"). User-controlled name/color fields must reject it,
// otherwise the frontend descriptor parser breaks.
const sepRune = '§'

func checkNoSep(fields ...*string) error {
	for _, f := range fields {
		if f != nil && strings.ContainsRune(*f, sepRune) {
			return fmt.Errorf("name/color must not contain '%c'", sepRune)
		}
	}
	return nil
}

////////////////////////////////////////////////
// Node Resolver
////////////////////////////////////////////////

// ras

////////////////////////////////////////////////
// Artefact Resolver (Label, RoleExt...)
////////////////////////////////////////////////

type AddArtefactInput struct {
	Name          *string          `json:"name"`
	Color         *string          `json:"color"`
	Rootnameid    string           `json:"rootnameid,omitempty"`
	Nodes         []*model.NodeRef `json:"nodes,omitempty"`
	Collaborators []*model.UserRef `json:"collaborators,omitempty"`
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

// Add "Artefact"
func addNodeArtefactHook(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
	// Authorization
	// - Check that rootnameid comply with Nodes
	// - nodes is required
	// - Check that user satisfy strict condition (coordo roles on node linked)

	// Get User context
	ctx, uctx, err := auth.GetUserContext(ctx)
	if err != nil {
		return nil, LogErr("Access denied", err)
	}

	// Validate input
	var inputs []AddArtefactInput
	ExtractInputs(ctx, &inputs)
	for _, input := range inputs {
		if err := checkNoSep(input.Name, input.Color); err != nil {
			return nil, LogErr("Invalid input", err)
		}
		if len(input.Nodes) == 0 {
			return nil, LogErr("Access denied", fmt.Errorf("A node must be given."))
		}
		node := input.Nodes[0]
		rootnameid, _ := codec.Nid2rootid(*node.Nameid)
		if rootnameid != input.Rootnameid {
			return nil, LogErr("Access denied", fmt.Errorf("rootnameid and nameid does not match."))
		}
		// Authorization with regards to the given nodes.
		if err = auth.Authorize(auth.CheckNodesAuth(uctx, DerefSlice(input.Nodes), true)); err != nil {
			return nil, err
		}
	}

	// Forward Query
	return next(ctx)
}

// Update "Artefact" - Must be coordo
func updateNodeArtefactHook(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
	// Pre-processing:
	// - Auth
	// - get values prior mutattions

	// Protected Object has more restrivive conditions to be updated.
	// @TODO: Clarify how the resources access policy for artefacts object (that can belongs to multiple nodes)
	// Ex: { Leaders: (mandate acess, tension access), Coordinators: (artefact access, tension access) }?
	protecteds := []string{"Label", "RoleExt"}
	isProtected := false
	_, typeName, _, err := queryTypeFromGraphqlContext(ctx)
	if err != nil {
		return nil, err
	}
	if slices.Contains(protecteds, typeName) {
		isProtected = true
	}

	// Get User context
	ctx, uctx, err := auth.GetUserContext(ctx)
	if err != nil {
		return nil, LogErr("Access denied", err)
	}

	// Validate input
	var input UpdateArtefactInput
	ExtractInput(ctx, &input)
	if input.Set != nil {
		if err := checkNoSep(input.Set.Name, input.Set.Color); err != nil {
			return nil, LogErr("Invalid input", err)
		}
	}

	// Get nodes in order to perform @auth rules against it
	nodes := []model.NodeRef{}
	nodesGiven := []model.NodeRef{}
	var x any
	if len(input.Filter.ID) > 0 { // Updates with UID
		x, err = db.GetDB().GetByUid(input.Filter.ID[0], typeName+".nodes", "Node.nameid")
	} else { // Update from hash names
		if typeName == "Project" && input.Filter.Parentnameid.Eq != nil && input.Filter.Nameid.Eq != nil {
			// Project like artefacts
			x, err = db.GetDB().GetByEqFiltered(typeName+".nameid", *input.Filter.Nameid.Eq, typeName+".parentnameid", *input.Filter.Parentnameid.Eq, typeName+".nodes", "Node.nameid")
		} else if input.Filter.Name.Eq != nil && input.Filter.Rootnameid.Eq != nil {
			// Other Artefacts update from hash names
			x, err = db.GetDB().GetByEqFiltered(typeName+".name", *input.Filter.Name.Eq, typeName+".rootnameid", *input.Filter.Rootnameid.Eq, typeName+".nodes", "Node.nameid")
		} else {
			return nil, LogErr("Access denied", fmt.Errorf("invalid filter to update node artefact."))
		}
	}
	if err != nil {
		return nil, err
	}

	// If x is nil (artefact not yet linked), nodes stays empty — allowed
	for _, nameid := range InterfaceToSlice[string](x) {
		nodes = append(nodes, model.NodeRef{Nameid: &nameid})
	}

	// Get given nodes
	if input.Set != nil {
		nodesGiven = append(nodesGiven, DerefSlice(input.Set.Nodes)...)
	}
	if input.Remove != nil {
		hasNodes := len(input.Remove.Nodes) > 0
		hasCollaborators := len(input.Remove.Collaborators) > 0

		if hasNodes {
			nodesGiven = append(nodesGiven, DerefSlice(input.Remove.Nodes)...)
		} else if !hasCollaborators {
			// Remove must specify at least nodes or collaborators
			return nil, LogErr("Access denied", fmt.Errorf("A node must be given."))
		}
	}

	// Authorization with regards to nodes attributes.
	if err = auth.Authorize(auth.CheckNodesAuth(uctx, nodes, false)); err != nil {
		return nil, err
	}

	// Authorization with regards to the given nodes.
	if err = auth.Authorize(auth.CheckNodesAuth(uctx, nodesGiven, true)); err != nil {
		return nil, err
	}

	// If an artefact is protected and is linked to multiple nodes,
	// Only allow updates if user has auth in the node with the shortest path to root.
	if input.Set != nil && isProtected && len(nodes) > 1 {
		mode := model.NodeModeCoordinated
		rootnameid, _ := codec.Nid2rootid(*nodes[0].Nameid)
		best_node := ""
		best_weight := -1.0
		for _, n := range nodes {
			nameid := *n.Nameid
			w, err := db.GetDB().GetShortestPath(rootnameid, nameid)
			if err != nil {
				return nil, err
			}
			if w < best_weight || best_weight < 0 {
				best_weight = w
				best_node = nameid
			}
		}
		// Check auth on the higher circle
		ok, err := auth.HasCoordoAuth(uctx, best_node, &mode)
		if err != nil {
			return nil, LogErr("Internal error", err)
		} else if !ok {
			return nil, LogErr("Access denied", fmt.Errorf("you need the be a coordinator of the highest circle that use this resource to update it."))
		}

	}

	// Get project data for potential reparenting
	var projectId, projectParentnameid string
	if typeName == "Project" && input.Remove != nil && len(input.Remove.Nodes) > 0 {
		var pData any
		if len(input.Filter.ID) > 0 {
			pData, err = db.GetDB().GetByUid(input.Filter.ID[0], "uid Project.parentnameid Project.rootnameid")
		} else if input.Filter.Parentnameid != nil && input.Filter.Parentnameid.Eq != nil &&
			input.Filter.Nameid != nil && input.Filter.Nameid.Eq != nil {
			pData, err = db.GetDB().GetByEqFiltered(
				"Project.nameid", *input.Filter.Nameid.Eq,
				"Project.parentnameid", *input.Filter.Parentnameid.Eq,
				"uid Project.parentnameid Project.rootnameid",
			)
		}
		if err != nil {
			return nil, LogErr("Internal error", err)
		}
		if pData != nil {
			p := StructMap[struct{ Id, Parentnameid, Rootnameid string }](pData)
			projectId = p.Id
			projectParentnameid = p.Parentnameid
		}
	}

	// Prevent removing all nodes from a Project, and prepare reparenting data
	var removedSet map[string]bool
	var parentRemoved bool
	if typeName == "Project" && input.Remove != nil && len(input.Remove.Nodes) > 0 {
		added := 0
		if input.Set != nil {
			added = len(input.Set.Nodes)
		}
		if len(nodes)+added-len(input.Remove.Nodes) <= 0 {
			return nil, LogErr("Access denied", fmt.Errorf("Cannot remove the last node from a project."))
		}

		removedSet = make(map[string]bool)
		for _, node := range input.Remove.Nodes {
			if node.Nameid != nil {
				removedSet[*node.Nameid] = true
				if *node.Nameid == projectParentnameid {
					parentRemoved = true
				}
			}
		}
	}

	// Get value prior mutation
	isRelabeling := typeName == "Label" && len(input.Filter.ID) > 0 && input.Set != nil && (input.Set.Name != nil || input.Set.Color != nil)
	old := struct{ Name, Color, Rootnameid string }{}
	if isRelabeling {
		// Old value -- Color is embeded in the event new/old value
		old_, err := db.GetDB().GetByUid(input.Filter.ID[0], "Label.name Label.color Label.rootnameid")
		if err != nil {
			return nil, LogErr("Internal error", err)
		}
		old = StructMap[struct{ Name, Color, Rootnameid string }](old_)
	}

	// Forward Query
	data, err := next(ctx)
	if err != nil {
		return data, err
	}

	// Post-processing:
	// - Re-parent project if its parent node was removed from nodes
	// - Rename unlink labels

	if parentRemoved && projectId != "" {
		for _, node := range nodes {
			if node.Nameid != nil && !removedSet[*node.Nameid] {
				newParentnameid := *node.Nameid
				newRootnameid, _ := codec.Nid2rootid(newParentnameid)
				if err = db.GetDB().SetFieldById(projectId, "Project.parentnameid", newParentnameid); err != nil {
					return data, LogErr("Internal error", err)
				}
				if err = db.GetDB().SetFieldById(projectId, "Project.rootnameid", newRootnameid); err != nil {
					return data, LogErr("Internal error", err)
				}
				break
			}
		}
	}

	// Update the Label event in tension history as data is hardcoded on new/old value.
	// @debug/perf: run this asynchronously and after next()
	if isRelabeling {
		new := struct{ Name, Color string }{}
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
			return data, LogErr("Internal error", err)
		}
	}

	return data, err
}

// Delete "Artefact" - Must be coordo
func deleteNodeArtefactHook(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
	_, typeName, _, err := queryTypeFromGraphqlContext(ctx)
	if err != nil {
		return nil, err
	}

	// Get User context
	ctx, uctx, err := auth.GetUserContext(ctx)
	if err != nil {
		return nil, LogErr("Access denied", err)
	}

	// Validate filter
	var filter FilterArtefactInput
	ExtractFilter(ctx, &filter)

	// Get nodes linked to the artefact
	var x any
	if len(filter.ID) > 0 {
		x, err = db.GetDB().GetByUid(filter.ID[0], typeName+".nodes", "Node.nameid")
	} else if filter.Name != nil && filter.Name.Eq != nil && filter.Rootnameid != nil && filter.Rootnameid.Eq != nil {
		x, err = db.GetDB().GetByEqFiltered(typeName+".name", *filter.Name.Eq, typeName+".rootnameid", *filter.Rootnameid.Eq, typeName+".nodes", "Node.nameid")
	} else {
		return nil, LogErr("Access denied", fmt.Errorf("invalid filter to delete node artefact."))
	}
	if err != nil {
		return nil, err
	}

	nodes := []model.NodeRef{}
	for _, nameid := range InterfaceToSlice[string](x) {
		nodes = append(nodes, model.NodeRef{Nameid: &nameid})
	}

	// Authorization with regards to linked nodes
	if err = auth.Authorize(auth.CheckNodesAuth(uctx, nodes, false)); err != nil {
		return nil, err
	}

	return next(ctx)
}
