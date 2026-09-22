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
	"bytes"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph/codec"
	"fractale/fractal6.go/graph/model"
	. "fractale/fractal6.go/internal/tools"
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

//
// Export Tensions
//

const (
	maxExportSize     = 250 << 20 // 250MB
	maxExportTensions = 10000
)

var tensionExportHeader = []any{"id", "title", "receiver", "status", "type", "createdAt", "createdBy", "labels", "assignees", "projects", "n_comments", "message"}

// Column widths, in the tensionExportHeader order.
var tensionExportWidths = []float64{14, 60, 26, 10, 12, 22, 16, 24, 20, 20, 12, 80}

// Long text columns get word wrap, title and message.
var tensionExportWrapCols = []string{"B", "L"}

const exportRowHeight = 15

// TensionsExport streams the queried tensions as a xlsx file.
// Same query/auth semantics as TensionsHandler.
func TensionsExport(w http.ResponseWriter, r *http.Request) {
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
	if q.First <= 0 || q.First > maxExportTensions {
		q.First = maxExportTensions
	}

	tensions, err := db.GetDB().GetTensions(q, "export")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	term := tensionTerm(q)
	buf, err := buildTensionsXlsx(tensions, term+"s")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	// ponytail: size checked after building in memory, go for a streaming writer if it ever OOMs
	if buf.Len() > maxExportSize {
		http.Error(w, "export too large: narrow down the query", 413)
		return
	}

	filename := fmt.Sprintf("%ss_%s.xlsx", strings.ToLower(term), time.Now().Format("20060102"))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	buf.WriteTo(w)
}

// tensionTerm returns the org lexicon term for "Tension" (Node.lexicon json field).
// Purely cosmetic: any lookup/decoding failure falls back to "Tension".
func tensionTerm(q db.TensionQuery) string {
	const fallback = "Tension"
	nameids := q.Nameids
	if len(nameids) == 0 {
		nameids = q.NameidsProtected
	}
	if len(nameids) == 0 {
		return fallback
	}
	rootnameid, err := codec.Nid2rootid(nameids[0])
	if err != nil {
		return fallback
	}
	v, _ := db.GetDB().GetByEq("Node.nameid", rootnameid, "Node.lexicon")
	raw, _ := v.(string)
	var lexicon map[string]string
	if json.Unmarshal([]byte(raw), &lexicon) != nil || lexicon["Tension"] == "" {
		return fallback
	}
	return lexicon["Tension"]
}

// buildTensionsXlsx renders the tensions in a single sheet workbook.
func buildTensionsXlsx(tensions []model.TensionRef, sheet string) (*bytes.Buffer, error) {
	f := excelize.NewFile()
	defer f.Close()
	// Lexicon terms may violate xlsx sheet name rules (31 chars, no :\/?*[]): keep the default name then
	if err := f.SetSheetName(f.GetSheetName(0), sheet); err != nil {
		sheet = f.GetSheetName(0)
	}
	if err := f.SetSheetRow(sheet, "A1", &tensionExportHeader); err != nil {
		return nil, err
	}
	if err := styleTensionsSheet(f, sheet); err != nil {
		return nil, err
	}

	for i, t := range tensions {
		var labels, assignees, projects []string
		for _, l := range t.Labels {
			labels = append(labels, Deref(l.Name))
		}
		for _, a := range t.Assignees {
			assignees = append(assignees, Deref(a.Username))
		}
		for _, c := range t.ProjectStatuses {
			if c.Project != nil {
				projects = append(projects, Deref(c.Project.Name))
			}
		}
		var author, receiver, message string
		if t.CreatedBy != nil {
			author = Deref(t.CreatedBy.Username)
		}
		if t.Receiver != nil {
			receiver = Deref(t.Receiver.Name)
		}
		// The tension body is its first comment
		if len(t.Comments) > 0 && t.Comments[0] != nil {
			message = Deref(t.Comments[0].Message)
		}

		row := []any{
			Deref(t.ID), Deref(t.Title), receiver,
			string(Deref(t.Status)), string(Deref(t.Type)),
			Deref(t.CreatedAt), author,
			strings.Join(labels, ", "), strings.Join(assignees, ", "), strings.Join(projects, ", "),
			Deref(t.NComments), message,
		}
		if err := f.SetSheetRow(sheet, fmt.Sprintf("A%d", i+2), &row); err != nil {
			return nil, err
		}
		// Fixed height so wrapped cells stay one line until the user autofits the row
		if err := f.SetRowHeight(sheet, i+2, exportRowHeight); err != nil {
			return nil, err
		}
	}

	return f.WriteToBuffer()
}

// styleTensionsSheet sets the column widths and turns the first row into a
// bold, frozen and filterable header.
func styleTensionsSheet(f *excelize.File, sheet string) error {
	lastCol, err := excelize.ColumnNumberToName(len(tensionExportHeader))
	if err != nil {
		return err
	}
	for i, width := range tensionExportWidths {
		col, err := excelize.ColumnNumberToName(i + 1)
		if err != nil {
			return err
		}
		if err := f.SetColWidth(sheet, col, col, width); err != nil {
			return err
		}
	}
	wrapStyle, err := f.NewStyle(&excelize.Style{Alignment: &excelize.Alignment{WrapText: true, Vertical: "top"}})
	if err != nil {
		return err
	}
	for _, col := range tensionExportWrapCols {
		if err := f.SetColStyle(sheet, col, wrapStyle); err != nil {
			return err
		}
	}
	// After SetColStyle: column styles overwrite existing cells, header bold must win
	headerStyle, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return err
	}
	if err := f.SetCellStyle(sheet, "A1", lastCol+"1", headerStyle); err != nil {
		return err
	}
	if err := f.SetPanes(sheet, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"}); err != nil {
		return err
	}
	return f.AutoFilter(sheet, "A1:"+lastCol+"1", nil)
}
