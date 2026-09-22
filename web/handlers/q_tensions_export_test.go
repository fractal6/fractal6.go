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
	if rows[0][0] != "id" || rows[0][1] != "title" || rows[0][11] != "message" {
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
	if width, err := f.GetColWidth("Tensions", "B"); err != nil || width != tensionExportWidths[1] {
		t.Errorf("title column width: got %v (err %v), want %v", width, err, tensionExportWidths[1])
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
