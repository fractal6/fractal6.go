/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2024 Fractale Co
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
	//"fmt"
	"encoding/json"
	"net/http"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/web/auth"
)

//
// Query data
// @Todo: token and check private status
//

func SubNodes(w http.ResponseWriter, r *http.Request) {
	var q string

	// Get the JSON body and decode it
	err := json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// Get sub children
	data, err := db.GetDB().GetSubNodes("nameid", q)
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

func SubMembers(w http.ResponseWriter, r *http.Request) {
	var q string

	// Get the JSON body and decode it
	err := json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// Get sub members
	data, err := db.GetDB().GetSubMembers("nameid", q, "User.name User.username")
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

func TopLabels(w http.ResponseWriter, r *http.Request) {
	form := struct {
		Nameid      string
		IncludeSelf bool
	}{}

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

	// Return the user context
	jsonData, err := json.Marshal(data)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write(jsonData)
}

func SubLabels(w http.ResponseWriter, r *http.Request) {
	var q string

	// Get the JSON body and decode it
	err := json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// Get sub labels
	data, err := db.GetDB().GetSubLabels("nameid", q)
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

func TopRoles(w http.ResponseWriter, r *http.Request) {
	form := struct {
		Nameid      string
		IncludeSelf bool
	}{}

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

	// Return the user context
	jsonData, err := json.Marshal(data)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write(jsonData)
}

func SubRoles(w http.ResponseWriter, r *http.Request) {
	var q string

	// Get the JSON body and decode it
	err := json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// Get sub roles
	data, err := db.GetDB().GetSubRoles("nameid", q)
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
