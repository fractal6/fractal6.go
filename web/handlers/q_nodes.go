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
	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/web/auth"
)

//
// Query node data
//
// All /q/* routes follow a two-phase pattern designed to be paginate-safe:
//   1. fetch {nameid -> visibility} for the requested subtree (or ancestor
//      chain) via DQL, bypassing @auth (cheap)
//   2. classify visible nameids in Go via auth.ClassifyVisibleNameids
//   3. fetch the artefacts (members, labels, roles, etc.) restricted to the
//      visible nameids
//
// Phase 3 only sees authorized circles, so any future pagination/limit on
// phase 3 will not under-fill due to post-fetch filtering.
//

// nodeQuery is the common request body for node query endpoints.
type nodeQuery struct {
	Nameid      string `json:"nameid"`
	IncludeSelf bool   `json:"include_self"`
}

// nodeHolder is satisfied by types returned by /q/{labels,roles,tension_templates,project_templates,projects}.
type nodeHolder interface {
	model.Label | model.RoleExt | model.TensionTemplate | model.ProjectTemplate | db.ProjectFull
}

// VisFetcher fetches per-circle visibility for a recursion shape (sub-tree or ancestor chain).
type VisFetcher func(fieldid, objid string, includeSelf bool) (map[string]model.NodeVisibility, error)

// ArtefactFetcher fetches artefacts attached to the given visible nameids.
// objid is passed through for fetchers that distinguish self vs ancestors
// (e.g. GetTopTensionTemplatesIn); other fetchers ignore it.
type ArtefactFetcher[T nodeHolder] func(visibleNameids []string, objid string) ([]T, error)

// classifyVisible runs phase 1 + phase 2 for a request: fetch visibility
// for the (visFn) recursion, classify in Go.
func classifyVisible(r *http.Request, form nodeQuery, visFn VisFetcher) ([]string, error) {
	visMap, err := visFn("nameid", form.Nameid, form.IncludeSelf)
	if err != nil {
		return nil, err
	}
	uctx := auth.GetUserContextOrEmpty(r.Context())
	return auth.ClassifyVisibleNameids(&uctx, visMap)
}

// NodeHolderHandler returns a generic /q/* handler that fetches visibility,
// classifies, then fetches artefacts restricted to visible nameids.
func NodeHolderHandler[T nodeHolder](visFn VisFetcher, fetchFn ArtefactFetcher[T]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var form nodeQuery
		if !decodeBody(w, r, &form) {
			return
		}
		visible, err := classifyVisible(r, form, visFn)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if len(visible) == 0 {
			writeJSON(w, []T{})
			return
		}
		data, err := fetchFn(visible, form.Nameid)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		writeJSON(w, data)
	}
}

// SubNodes returns visible circles in the subtree of form.Nameid.
// The visibility-fetch result already contains everything the response needs
// (nameid + visibility), so no second DQL call is issued.
func SubNodes(w http.ResponseWriter, r *http.Request) {
	var form nodeQuery
	if !decodeBody(w, r, &form) {
		return
	}
	visMap, err := db.GetDB().GetSubNodeVisibilities("nameid", form.Nameid, form.IncludeSelf)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	uctx := auth.GetUserContextOrEmpty(r.Context())
	visible, err := auth.ClassifyVisibleNameids(&uctx, visMap)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	out := make([]model.Node, 0, len(visible))
	for _, nameid := range visible {
		out = append(out, model.Node{Nameid: nameid, Visibility: visMap[nameid]})
	}
	writeJSON(w, out)
}

// SubMembers returns members attached to circles in the subtree of form.Nameid
// that the user is authorized to see.
func SubMembers(w http.ResponseWriter, r *http.Request) {
	var form nodeQuery
	if !decodeBody(w, r, &form) {
		return
	}
	visible, err := classifyVisible(r, form, db.GetDB().GetSubNodeVisibilities)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if len(visible) == 0 {
		writeJSON(w, []model.Node{})
		return
	}
	data, err := db.GetDB().GetMembersIn(visible, "User.name User.username")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, data)
}
