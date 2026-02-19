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

package handlers

import (
	"encoding/json"
	"net/http"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/web/auth"
)

//
// Query data
// @Todo: token and check private status
//

// nodeQuery is the common request body for node query endpoints.
type nodeQuery struct {
	Nameid      string `json:"nameid"`
	IncludeSelf bool   `json:"include_self"`
}

func SubNodes(w http.ResponseWriter, r *http.Request) {
	var form nodeQuery

	// Get the JSON body and decode it
	err := json.NewDecoder(r.Body).Decode(&form)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// Get sub children
	data, err := db.GetDB().GetSubNodes("nameid", form.Nameid, form.IncludeSelf)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Filter nodes by visibility
	uctx := auth.GetUserContextOrEmpty(r.Context())
	var nameids []string
	for _, n := range data {
		nameids = append(nameids, n.Nameid)
	}
	visible, err := auth.NodeVisibilityFilter(&uctx, nameids)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	filtered := []model.Node{}
	for _, n := range data {
		if visible[n.Nameid] {
			filtered = append(filtered, n)
		}
	}

	jsonData, err := json.Marshal(filtered)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write(jsonData)
}

func SubMembers(w http.ResponseWriter, r *http.Request) {
	var form nodeQuery

	// Get the JSON body and decode it
	err := json.NewDecoder(r.Body).Decode(&form)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// Get sub members
	data, err := db.GetDB().GetSubMembers("nameid", form.Nameid, "User.name User.username", form.IncludeSelf)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Filter members by parent circle visibility
	uctx := auth.GetUserContextOrEmpty(r.Context())
	parentSet := make(map[string]bool)
	for _, n := range data {
		if n.Parent != nil {
			parentSet[n.Parent.Nameid] = true
		}
	}
	var parentNameids []string
	for nid := range parentSet {
		parentNameids = append(parentNameids, nid)
	}
	visible, err := auth.NodeVisibilityFilter(&uctx, parentNameids)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	filtered := []model.Node{}
	for _, n := range data {
		if n.Parent != nil && visible[n.Parent.Nameid] {
			filtered = append(filtered, n)
		}
	}

	jsonData, err := json.Marshal(filtered)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write(jsonData)
}

func TopLabels(w http.ResponseWriter, r *http.Request) {
	var form nodeQuery

	// Get the JSON body and decode it
	err := json.NewDecoder(r.Body).Decode(&form)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// Get top labels
	data, err := db.GetDB().GetTopLabels("nameid", form.Nameid, form.IncludeSelf)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Filter labels by node visibility
	uctx := auth.GetUserContextOrEmpty(r.Context())
	filtered, err := filterByNodeVisibility(&uctx, data)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	jsonData, err := json.Marshal(filtered)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write(jsonData)
}

func SubLabels(w http.ResponseWriter, r *http.Request) {
	var form nodeQuery

	// Get the JSON body and decode it
	err := json.NewDecoder(r.Body).Decode(&form)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// Get sub labels
	data, err := db.GetDB().GetSubLabels("nameid", form.Nameid, form.IncludeSelf)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Filter labels by node visibility
	uctx := auth.GetUserContextOrEmpty(r.Context())
	filtered, err := filterByNodeVisibility(&uctx, data)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	jsonData, err := json.Marshal(filtered)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write(jsonData)
}

func TopRoles(w http.ResponseWriter, r *http.Request) {
	var form nodeQuery

	// Get the JSON body and decode it
	err := json.NewDecoder(r.Body).Decode(&form)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// Get top roles
	data, err := db.GetDB().GetTopRoles("nameid", form.Nameid, form.IncludeSelf)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Filter roles by node visibility
	uctx := auth.GetUserContextOrEmpty(r.Context())
	filtered, err := filterByNodeVisibility(&uctx, data)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	jsonData, err := json.Marshal(filtered)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write(jsonData)
}

func SubRoles(w http.ResponseWriter, r *http.Request) {
	var form nodeQuery

	// Get the JSON body and decode it
	err := json.NewDecoder(r.Body).Decode(&form)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// Get sub roles
	data, err := db.GetDB().GetSubRoles("nameid", form.Nameid, form.IncludeSelf)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Filter roles by node visibility
	uctx := auth.GetUserContextOrEmpty(r.Context())
	filtered, err := filterByNodeVisibility(&uctx, data)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	jsonData, err := json.Marshal(filtered)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write(jsonData)
}

// nodeHolder is satisfied by types that have a Nodes []*model.Node field.
type nodeHolder interface {
	model.Label | model.RoleExt | db.ProjectFull
}

// getNodes returns the Nodes field for items implementing nodeHolder.
func getNodes[T nodeHolder](item *T) []*model.Node {
	switch v := any(item).(type) {
	case *model.Label:
		return v.Nodes
	case *model.RoleExt:
		return v.Nodes
	case *db.ProjectFull:
		return v.Nodes
	}
	return nil
}

// setNodes sets the Nodes field for items implementing nodeHolder.
func setNodes[T nodeHolder](item *T, nodes []*model.Node) {
	switch v := any(item).(type) {
	case *model.Label:
		v.Nodes = nodes
	case *model.RoleExt:
		v.Nodes = nodes
	case *db.ProjectFull:
		v.Nodes = nodes
	}
}

// filterByNodeVisibility filters items (Labels, RoleExt, or Projects) by checking
// visibility of their attached nodes. Items with no visible nodes are dropped.
func filterByNodeVisibility[T nodeHolder](uctx *model.UserCtx, data []T) ([]T, error) {
	nodeSet := make(map[string]bool)
	for i := range data {
		for _, n := range getNodes(&data[i]) {
			if n != nil {
				nodeSet[n.Nameid] = true
			}
		}
	}
	var allNameids []string
	for nid := range nodeSet {
		allNameids = append(allNameids, nid)
	}
	visible, err := auth.NodeVisibilityFilter(uctx, allNameids)
	if err != nil {
		return nil, err
	}
	filtered := make([]T, 0)
	for i := range data {
		var visibleNodes []*model.Node
		for _, n := range getNodes(&data[i]) {
			if n != nil && visible[n.Nameid] {
				visibleNodes = append(visibleNodes, n)
			}
		}
		if len(visibleNodes) > 0 {
			setNodes(&data[i], visibleNodes)
			filtered = append(filtered, data[i])
		}
	}
	return filtered, nil
}

func SubProjects(w http.ResponseWriter, r *http.Request) {
	var form nodeQuery

	// Get the JSON body and decode it
	err := json.NewDecoder(r.Body).Decode(&form)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// Get sub projects
	data, err := db.GetDB().GetSubProjects("nameid", form.Nameid, form.IncludeSelf)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Filter projects by node visibility
	uctx := auth.GetUserContextOrEmpty(r.Context())
	filtered, err := filterByNodeVisibility(&uctx, data)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	jsonData, err := json.Marshal(filtered)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write(jsonData)
}

//
// Query Tensions
//

func TensionsLight(w http.ResponseWriter, r *http.Request) {
	var q db.TensionQuery

	// Get the JSON body and decode it
	err := json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// Filter the nameids according to the @auth directives
	uctx := auth.GetUserContextOrEmpty(r.Context())
	err = auth.QueryAuthFilter(uctx, &q)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Get Int Tensions
	data, err := db.GetDB().GetTensions(q, "light")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Filter authorized tension
	//final := []model.Tension{}
	//ids := []string{}
	//for _, t := range data {
	//    ids = append(ids, t.ID)
	//}
	//newIds, err := db.GetDB().Query(uctx, "tension", "id", ids, "id")
	//if err != nil {
	//    http.Error(w, err.Error(), 500)
	//    return
	//}
	//if len(ids) != len(newIds) {
	//    // What to do ?
	//    // It is prompt to breaks the "LoadMore" functionality
	//}

	// Return the user context
	jsonData, err := json.Marshal(data)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write(jsonData)
}

func TensionsInt(w http.ResponseWriter, r *http.Request) {
	var q db.TensionQuery

	// Get the JSON body and decode it
	err := json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// Filter the nameids according to the @auth directives
	uctx := auth.GetUserContextOrEmpty(r.Context())
	err = auth.QueryAuthFilter(uctx, &q)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Get Int Tensions
	data, err := db.GetDB().GetTensions(q, "int")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Filter authorized tension
	//final := []model.Tension{}
	//ids := []string{}
	//for _, t := range data {
	//    ids = append(ids, t.ID)
	//}
	//newIds, err := db.GetDB().Query(uctx, "tension", "id", ids, "id")
	//if err != nil {
	//    http.Error(w, err.Error(), 500)
	//    return
	//}
	//if len(ids) != len(newIds) {
	//    // What to do ?
	//    // It is prompt to breaks the "LoadMore" functionality
	//}

	// Return the user context
	jsonData, err := json.Marshal(data)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write(jsonData)
}

func TensionsExt(w http.ResponseWriter, r *http.Request) {
	var q db.TensionQuery

	// Get the JSON body and decode it
	err := json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// Filter the nameids according to the @auth directives
	uctx := auth.GetUserContextOrEmpty(r.Context())
	err = auth.QueryAuthFilter(uctx, &q)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Get Ext Tensions
	data, err := db.GetDB().GetTensions(q, "ext")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Return the user context
	jsonData, err := json.Marshal(data)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write(jsonData)
}

func TensionsAll(w http.ResponseWriter, r *http.Request) {
	var q db.TensionQuery

	// Get the JSON body and decode it
	err := json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// Filter the nameids according to the @auth directives
	uctx := auth.GetUserContextOrEmpty(r.Context())
	err = auth.QueryAuthFilter(uctx, &q)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Get all tensions
	data, err := db.GetDB().GetTensions(q, "all")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Return the user context
	jsonData, err := json.Marshal(data)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write(jsonData)
}

func TensionsCount(w http.ResponseWriter, r *http.Request) {
	var q db.TensionQuery

	// Get the JSON body and decode it
	err := json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// Filter the nameids according to the @auth directives
	uctx := auth.GetUserContextOrEmpty(r.Context())
	err = auth.QueryAuthFilter(uctx, &q)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Get tension counts
	data, err := db.GetDB().GetTensionsCount(q)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Return the user context
	jsonData, err := json.Marshal(data)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write(jsonData)
}
