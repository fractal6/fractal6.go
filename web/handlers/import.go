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
	"strings"

	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/internal/orgimport"
	"fractale/fractal6.go/web/auth"
)

// ImportOrga handles spreadsheet upload and creates an organisation from it.
func ImportOrga(w http.ResponseWriter, r *http.Request) {
	// Authenticate user
	uctx, err := auth.GetUserContextLight(r.Context())
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Parse multipart form (10MB max)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "failed to parse form: "+err.Error(), 400)
		return
	}

	// Extract form fields
	name := strings.TrimSpace(r.FormValue("name"))
	nameid := strings.TrimSpace(r.FormValue("nameid"))
	format := strings.TrimSpace(r.FormValue("format"))
	about := r.FormValue("about")

	var visibility model.NodeVisibility
	visStr := r.FormValue("visibility")
	if visStr != "" && model.NodeVisibility(visStr).IsValid() {
		visibility = model.NodeVisibility(visStr)
	} else {
		visibility = model.NodeVisibilityPublic
	}

	// Validate required fields
	if err := auth.ValidateName(name); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if err := auth.ValidateNameid(nameid, nameid); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if strings.Contains(nameid, "#") {
		http.Error(w, "Illegal character '#' in nameid", 400)
		return
	}

	// Check plan permissions
	form := model.OrgaForm{
		Name:       name,
		Nameid:     nameid,
		About:      &about,
		Visibility: &visibility,
	}
	ok, err := auth.CanNewOrga(*uctx, form)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if !ok {
		http.Error(w, "permission denied: cannot create organisation", 403)
		return
	}

	// Extract uploaded file
	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing or invalid file: "+err.Error(), 400)
		return
	}
	defer file.Close()

	// Read spreadsheet
	sheets, err := orgimport.ReadSpreadsheet(file, handler.Filename)
	if err != nil {
		http.Error(w, "failed to read spreadsheet: "+err.Error(), 400)
		return
	}

	// Detect source format and parse into tree
	if format == "" {
		format = orgimport.DetectSourceFormat(sheets)
	}
	tree, err := orgimport.ParseByFormat(format, sheets)
	if err != nil {
		http.Error(w, "failed to parse spreadsheet: "+err.Error(), 400)
		return
	}

	// Override root node name/purpose with form values
	tree.Name = name
	if about != "" {
		tree.Purpose = about
	}

	// Build the org
	if err := orgimport.BuildOrgFromTree(uctx, form, tree, visibility); err != nil {
		http.Error(w, "failed to create organisation: "+err.Error(), 500)
		return
	}

	writeJSON(w, model.Node{Nameid: nameid})
}
