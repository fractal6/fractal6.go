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
	"slices"
	"testing"

	"github.com/xuri/excelize/v2"

	"fractale/fractal6.go/graph/model"
)

func TestBuildTensionsXlsx(t *testing.T) {
	s := func(v string) *string { return &v }
	typ := model.TensionTypeOperational
	status := model.TensionStatusOpen
	n := 2

	tensions := []model.TensionRef{{
		ID:              s("0x1"),
		CreatedAt:       s("2026-01-01T00:00:00Z"),
		CreatedBy:       &model.UserRef{Username: s("alice")},
		Type:            &typ,
		Status:          &status,
		Title:           s("A tension"),
		Receiver:        &model.NodeRef{Name: s("Circle Name")},
		Labels:          []*model.LabelRef{{Name: s("bug")}, {Name: s("urgent")}},
		Assignees:       []*model.UserRef{{Username: s("bob")}},
		ProjectStatuses: []*model.ProjectColumnRef{{Project: &model.ProjectRef{Name: s("Roadmap")}}},
		Comments:        []*model.CommentRef{{Message: s("first message")}},
		NComments:       &n,
	}, {
		// Empty tension: all optional fields nil
		ID: s("0x2"),
	}}

	buf, err := buildTensionsXlsx(tensions, "Tensions")
	if err != nil {
		t.Fatalf("buildTensionsXlsx() error: %v", err)
	}

	f, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("OpenReader() error: %v", err)
	}
	defer f.Close()

	rows, err := f.GetRows("Tensions")
	if err != nil {
		t.Fatalf("GetRows() error: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows (header + 2), got %d", len(rows))
	}
	// No published blob: no node.* columns
	if len(rows[0]) != len(tensionExportCols) || rows[0][0] != "id" || rows[0][1] != "title" || rows[0][11] != "message" {
		t.Fatalf("unexpected header: %v", rows[0])
	}

	want := []string{"0x1", "A tension", "Circle Name", "Open", "Operational", "2026-01-01T00:00:00Z", "alice", "bug, urgent", "bob", "Roadmap", "2", "first message"}
	for i, v := range want {
		if rows[1][i] != v {
			t.Errorf("row 1 col %d: got %q, want %q", i, rows[1][i], v)
		}
	}

	// Nil fields must not leak "<nil>" in the sheet
	for _, c := range rows[2] {
		if c == "<nil>" {
			t.Fatalf("nil field rendered in row 2: %v", rows[2])
		}
	}
	if rows[2][0] != "0x2" {
		t.Errorf("row 2 id: got %q", rows[2][0])
	}

	// Header formatting: bold, frozen and filterable
	styleID, err := f.GetCellStyle("Tensions", "A1")
	if err != nil {
		t.Fatalf("GetCellStyle() error: %v", err)
	}
	style, err := f.GetStyle(styleID)
	if err != nil {
		t.Fatalf("GetStyle() error: %v", err)
	}
	if style.Font == nil || !style.Font.Bold {
		t.Error("header row is not bold")
	}
	if width, err := f.GetColWidth("Tensions", "B"); err != nil || width != tensionExportCols[1].width {
		t.Errorf("title column width: got %v (err %v), want %v", width, err, tensionExportCols[1].width)
	}

	// Long text: wrapped cells, fixed row height (user autofits on demand)
	msgStyleID, err := f.GetCellStyle("Tensions", "L2")
	if err != nil {
		t.Fatalf("GetCellStyle(L2) error: %v", err)
	}
	msgStyle, err := f.GetStyle(msgStyleID)
	if err != nil {
		t.Fatalf("GetStyle() error: %v", err)
	}
	if msgStyle.Alignment == nil || !msgStyle.Alignment.WrapText {
		t.Error("message cell is not wrapped")
	}
	if h, err := f.GetRowHeight("Tensions", 2); err != nil || h != exportRowHeight {
		t.Errorf("row 2 height: got %v (err %v), want %v", h, err, exportRowHeight)
	}
}

func TestBuildTensionsXlsxNode(t *testing.T) {
	s := func(v string) *string { return &v }
	nodeType := model.NodeTypeRole
	roleType := model.RoleTypePeer
	visibility := model.NodeVisibilityPublic

	tensions := []model.TensionRef{{ID: s("0x1")}, {
		ID: s("0x2"),
		Blobs: []*model.BlobRef{{Node: &model.NodeFragmentRef{
			Nameid:     s("org#circle#role"),
			Name:       s("A role"),
			Mandate:    &model.MandateRef{Purpose: s("the purpose"), Rules: s("the rules")},
			Skills:     []string{"go", "elm"},
			Visibility: &visibility,
			Type:       &nodeType,
			FirstLink:  s("alice"),
			RoleType:   &roleType,
		}}},
	}}

	buf, err := buildTensionsXlsx(tensions, "Tensions")
	if err != nil {
		t.Fatalf("buildTensionsXlsx() error: %v", err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("OpenReader() error: %v", err)
	}
	defer f.Close()
	rows, err := f.GetRows("Tensions")
	if err != nil {
		t.Fatalf("GetRows() error: %v", err)
	}
	if len(rows[0]) != len(tensionExportCols)+len(nodeExportCols) {
		t.Fatalf("expected node columns in header: %v", rows[0])
	}

	cell := func(row []string, name string) string {
		i := slices.Index(rows[0], name)
		if i < 0 || i >= len(row) {
			return ""
		}
		return row[i]
	}
	want := map[string]string{
		"node.nameid": "org#circle#role", "node.name": "A role", "node.mandate.purpose": "the purpose",
		"node.mandate.rules": "the rules", "node.skills": "go, elm", "node.visibility": "Public",
		"node.type": "Role", "node.first_link": "alice", "node.role_type": "Peer", "node.about": "",
	}
	for name, v := range want {
		if got := cell(rows[2], name); got != v {
			t.Errorf("%s: got %q, want %q", name, got, v)
		}
	}
	// Tension without blob: node cells left empty
	if len(rows[1]) > len(tensionExportCols) {
		t.Errorf("row without blob has node cells: %v", rows[1])
	}

	// Long mandate text is wrapped
	col, _ := excelize.ColumnNumberToName(slices.Index(rows[0], "node.mandate.purpose") + 1)
	styleID, err := f.GetCellStyle("Tensions", col+"3")
	if err != nil {
		t.Fatalf("GetCellStyle() error: %v", err)
	}
	style, err := f.GetStyle(styleID)
	if err != nil {
		t.Fatalf("GetStyle() error: %v", err)
	}
	if style.Alignment == nil || !style.Alignment.WrapText {
		t.Error("purpose cell is not wrapped")
	}
}
