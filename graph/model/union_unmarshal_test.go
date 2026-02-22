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

package model_test

import (
	"encoding/json"
	"testing"

	. "fractale/fractal6.go/graph/model"
)

func TestDecodeEventKind_Event(t *testing.T) {
	raw := json.RawMessage(`{"__typename":"Event","id":"0x1","event_type":"Created"}`)
	kind, err := DecodeEventKind(raw)
	if err != nil {
		t.Fatalf("DecodeEventKind error: %v", err)
	}
	ev, ok := kind.(*Event)
	if !ok {
		t.Fatalf("expected *Event, got %T", kind)
	}
	if ev.ID != "0x1" {
		t.Errorf("expected id=0x1, got %s", ev.ID)
	}
}

func TestDecodeEventKind_Contract(t *testing.T) {
	raw := json.RawMessage(`{"__typename":"Contract","id":"0x2","status":"Open"}`)
	kind, err := DecodeEventKind(raw)
	if err != nil {
		t.Fatalf("DecodeEventKind error: %v", err)
	}
	c, ok := kind.(*Contract)
	if !ok {
		t.Fatalf("expected *Contract, got %T", kind)
	}
	if c.ID != "0x2" {
		t.Errorf("expected id=0x2, got %s", c.ID)
	}
}

func TestDecodeEventKind_Notif(t *testing.T) {
	raw := json.RawMessage(`{"__typename":"Notif","id":"0x3","link":"https://example.com"}`)
	kind, err := DecodeEventKind(raw)
	if err != nil {
		t.Fatalf("DecodeEventKind error: %v", err)
	}
	n, ok := kind.(*Notif)
	if !ok {
		t.Fatalf("expected *Notif, got %T", kind)
	}
	if n.Link == nil || *n.Link != "https://example.com" {
		t.Errorf("expected link=https://example.com, got %v", n.Link)
	}
}

func TestDecodeEventKind_UnknownType(t *testing.T) {
	raw := json.RawMessage(`{"__typename":"Unknown","id":"0x4"}`)
	_, err := DecodeEventKind(raw)
	if err == nil {
		t.Error("expected error for unknown __typename")
	}
}

func TestDecodeEventKind_MissingTypename(t *testing.T) {
	raw := json.RawMessage(`{"id":"0x5"}`)
	_, err := DecodeEventKind(raw)
	if err == nil {
		t.Error("expected error for missing __typename")
	}
}

func TestDecodeCardKind_Tension(t *testing.T) {
	raw := json.RawMessage(`{"__typename":"Tension","id":"0x10","title":"Test tension"}`)
	kind, err := DecodeCardKind(raw)
	if err != nil {
		t.Fatalf("DecodeCardKind error: %v", err)
	}
	ten, ok := kind.(*Tension)
	if !ok {
		t.Fatalf("expected *Tension, got %T", kind)
	}
	if ten.Title != "Test tension" {
		t.Errorf("expected title='Test tension', got %s", ten.Title)
	}
}

func TestDecodeCardKind_ProjectDraft(t *testing.T) {
	raw := json.RawMessage(`{"__typename":"ProjectDraft","id":"0x11","title":"Draft"}`)
	kind, err := DecodeCardKind(raw)
	if err != nil {
		t.Fatalf("DecodeCardKind error: %v", err)
	}
	pd, ok := kind.(*ProjectDraft)
	if !ok {
		t.Fatalf("expected *ProjectDraft, got %T", kind)
	}
	if pd.Title != "Draft" {
		t.Errorf("expected title='Draft', got %s", pd.Title)
	}
}

func TestUserEvent_UnmarshalJSON(t *testing.T) {
	data := `{
		"id": "0x20",
		"createdAt": "2024-01-01T00:00:00Z",
		"isRead": false,
		"event": [
			{"__typename":"Event","id":"0x21","event_type":"Created"},
			{"__typename":"Contract","id":"0x22","status":"Open"}
		]
	}`
	var ue UserEvent
	if err := json.Unmarshal([]byte(data), &ue); err != nil {
		t.Fatalf("UnmarshalJSON error: %v", err)
	}
	if ue.ID != "0x20" {
		t.Errorf("expected id=0x20, got %s", ue.ID)
	}
	if len(ue.Event) != 2 {
		t.Fatalf("expected 2 events, got %d", len(ue.Event))
	}
	if _, ok := ue.Event[0].(*Event); !ok {
		t.Errorf("expected first event to be *Event, got %T", ue.Event[0])
	}
	if _, ok := ue.Event[1].(*Contract); !ok {
		t.Errorf("expected second event to be *Contract, got %T", ue.Event[1])
	}
}

func TestProjectCard_UnmarshalJSON(t *testing.T) {
	data := `{
		"id": "0x30",
		"pos": 1,
		"card": {"__typename":"Tension","id":"0x31","title":"My tension"}
	}`
	var pc ProjectCard
	if err := json.Unmarshal([]byte(data), &pc); err != nil {
		t.Fatalf("UnmarshalJSON error: %v", err)
	}
	if pc.ID != "0x30" {
		t.Errorf("expected id=0x30, got %s", pc.ID)
	}
	ten, ok := pc.Card.(*Tension)
	if !ok {
		t.Fatalf("expected card to be *Tension, got %T", pc.Card)
	}
	if ten.Title != "My tension" {
		t.Errorf("expected title='My tension', got %s", ten.Title)
	}
}
