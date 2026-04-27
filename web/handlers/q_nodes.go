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
	"net/http"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph/codec"
	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/web/auth"
)

//
// Query node data
// @Todo: token and check private status
//

// nodeQuery is the common request body for node query endpoints.
type nodeQuery struct {
	Nameid      string `json:"nameid"`
	IncludeSelf bool   `json:"include_self"`
}

// nodeHolder is satisfied by types that have a Nodes []*model.Node field.
type nodeHolder interface {
	model.Label | model.RoleExt | model.TensionTemplate | db.ProjectFull
}

// getNodes returns the Nodes field for items implementing nodeHolder.
func getNodes[T nodeHolder](item *T) []*model.Node {
	switch v := any(item).(type) {
	case *model.Label:
		return v.Nodes
	case *model.RoleExt:
		return v.Nodes
	case *model.TensionTemplate:
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
	case *model.TensionTemplate:
		v.Nodes = nodes
	case *db.ProjectFull:
		v.Nodes = nodes
	}
}

// filterByNodeVisibility filters items (Labels, RoleExt, or Projects) by checking
// visibility of their attached nodes. Items with no visible nodes are dropped.
//
// If every attached node already has Visibility populated (e.g. fetched in the
// DQL select), the membership/role check runs in-memory and skips the
// GraphQL @auth roundtrip — that auth pass is redundant since we re-check
// manually below.
func filterByNodeVisibility[T nodeHolder](uctx *model.UserCtx, data []T) ([]T, error) {
	visible, err := buildVisibilityMap(uctx, data)
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

// buildVisibilityMap returns a {nameid: bool} map for all nodes attached to
// data. Uses pre-fetched Node.Visibility when available; otherwise falls back
// to auth.NodeVisibilityFilter (which queries Dgraph through @auth rules).
func buildVisibilityMap[T nodeHolder](uctx *model.UserCtx, data []T) (map[string]bool, error) {
	type pair struct {
		nameid string
		vis    model.NodeVisibility
	}
	var flat []pair
	for i := range data {
		for _, n := range getNodes(&data[i]) {
			if n == nil {
				continue
			}
			flat = append(flat, pair{n.Nameid, n.Visibility})
		}
	}
	return visibilityMapFromNodes(uctx, flat, func(p pair) (string, model.NodeVisibility) {
		return p.nameid, p.vis
	})
}

// isNodeVisible mirrors the rules in auth.NodeVisibilityFilter but works from
// an already-known visibility value, avoiding a Dgraph roundtrip.
func isNodeVisible(uctx *model.UserCtx, nameid string, visibility model.NodeVisibility) (bool, error) {
	nid, err := codec.Nid2pid(nameid)
	if err != nil {
		return false, err
	}
	switch visibility {
	case model.NodeVisibilityPrivate:
		return auth.UserIsMember(uctx, nid) >= 0, nil
	case model.NodeVisibilitySecret:
		return auth.UserHasRole(uctx, nid) >= 0, nil
	default:
		return true, nil
	}
}

// NodeHolderHandler returns a handler that decodes a nodeQuery, fetches items
// using the provided function, filters by node visibility, and writes JSON.
func NodeHolderHandler[T nodeHolder](fetch func(fieldid, objid string, includeSelf bool) ([]T, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var form nodeQuery
		if !decodeBody(w, r, &form) {
			return
		}
		data, err := fetch("nameid", form.Nameid, form.IncludeSelf)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		uctx := auth.GetUserContextOrEmpty(r.Context())
		filtered, err := filterByNodeVisibility(&uctx, data)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSON(w, filtered)
	}
}

func SubNodes(w http.ResponseWriter, r *http.Request) {
	var form nodeQuery
	if !decodeBody(w, r, &form) {
		return
	}

	data, err := db.GetDB().GetSubNodes("nameid", form.Nameid, form.IncludeSelf)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	uctx := auth.GetUserContextOrEmpty(r.Context())
	visible, err := visibilityMapFromNodes(&uctx, data, func(n model.Node) (string, model.NodeVisibility) {
		return n.Nameid, n.Visibility
	})
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

	writeJSON(w, filtered)
}

func SubMembers(w http.ResponseWriter, r *http.Request) {
	var form nodeQuery
	if !decodeBody(w, r, &form) {
		return
	}

	data, err := db.GetDB().GetSubMembers("nameid", form.Nameid, "User.name User.username", form.IncludeSelf)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	uctx := auth.GetUserContextOrEmpty(r.Context())
	visible, err := visibilityMapFromNodes(&uctx, data, func(n model.Node) (string, model.NodeVisibility) {
		if n.Parent == nil {
			return "", ""
		}
		return n.Parent.Nameid, n.Parent.Visibility
	})
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

	writeJSON(w, filtered)
}

// visibilityMapFromNodes builds a {nameid: bool} visibility map for nodes
// extracted via keyFn. If every key has a non-empty Visibility, the check runs
// in-memory; otherwise it falls back to auth.NodeVisibilityFilter.
func visibilityMapFromNodes[T any](uctx *model.UserCtx, items []T, keyFn func(T) (string, model.NodeVisibility)) (map[string]bool, error) {
	known := make(map[string]model.NodeVisibility)
	allKnown := true
	for _, it := range items {
		nameid, vis := keyFn(it)
		if nameid == "" {
			continue
		}
		if _, ok := known[nameid]; ok {
			continue
		}
		known[nameid] = vis
		if vis == "" {
			allKnown = false
		}
	}

	if !allKnown {
		nameids := make([]string, 0, len(known))
		for nid := range known {
			nameids = append(nameids, nid)
		}
		return auth.NodeVisibilityFilter(uctx, nameids)
	}

	visible := make(map[string]bool, len(known))
	for nameid, vis := range known {
		v, err := isNodeVisible(uctx, nameid, vis)
		if err != nil {
			return nil, err
		}
		if v {
			visible[nameid] = true
		}
	}
	return visible, nil
}
