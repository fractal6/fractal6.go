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
	"fractale/fractal6.go/web/auth"
)

//
// Query Tensions
//

// TensionsHandler returns a handler for tension queries with the given mode.
func TensionsHandler(mode string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var q db.TensionQuery
		if !decodeBody(w, r, &q) {
			return
		}

		uctx := auth.GetUserContextOrEmpty(r.Context())
		if err := auth.QueryAuthFilter(uctx, &q); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		data, err := db.GetDB().GetTensions(q, mode)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		writeJSON(w, data)
	}
}

func TensionsCount(w http.ResponseWriter, r *http.Request) {
	var q db.TensionQuery
	if !decodeBody(w, r, &q) {
		return
	}

	// Filter the nameids according to the @auth directives
	uctx := auth.GetUserContextOrEmpty(r.Context())
	if err := auth.QueryAuthFilter(uctx, &q); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Get tension counts
	data, err := db.GetDB().GetTensionsCount(q)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	writeJSON(w, data)
}
