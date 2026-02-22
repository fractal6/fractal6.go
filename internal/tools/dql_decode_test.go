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

package tools_test

import (
	"fmt"
	"testing"

	"fractale/fractal6.go/graph/model"
	. "fractale/fractal6.go/internal/tools"
)

func TestDecodeDql_SingleRecord(t *testing.T) {
	raw := map[string]any{
		"uid":           "0x1",
		"Node.nameid":   "org#circle",
		"Node.name":     "Circle",
		"Node.type_":    "Circle",
		"Node.isRoot":   false,
		"Node.mode":     "Coordinated",
		"Node.rights":   float64(0),
		"Node.visibility": "Public",
	}
	node, err := DecodeDql[model.Node](raw)
	if err != nil {
		t.Fatalf("DecodeDql single record error: %v", err)
	}
	if node.ID != "0x1" {
		t.Errorf("expected id=0x1, got %s", node.ID)
	}
	if node.Nameid != "org#circle" {
		t.Errorf("expected nameid=org#circle, got %s", node.Nameid)
	}
	if node.Name != "Circle" {
		t.Errorf("expected name=Circle, got %s", node.Name)
	}
}

func TestDecodeDql_SliceOfRecords(t *testing.T) {
	raw := []map[string]any{
		{
			"uid":         "0x1",
			"Node.nameid": "org#circle1",
			"Node.name":   "Circle1",
		},
		{
			"uid":         "0x2",
			"Node.nameid": "org#circle2",
			"Node.name":   "Circle2",
		},
	}
	nodes, err := DecodeDql[[]model.Node](raw)
	if err != nil {
		t.Fatalf("DecodeDql slice error: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
	if nodes[0].Nameid != "org#circle1" {
		t.Errorf("expected first nameid=org#circle1, got %s", nodes[0].Nameid)
	}
	if nodes[1].ID != "0x2" {
		t.Errorf("expected second id=0x2, got %s", nodes[1].ID)
	}
}

func TestDecodeDql_PointerSlice(t *testing.T) {
	raw := []map[string]any{
		{
			"uid":            "0x1",
			"Node.nameid":    "org##@user",
			"Node.name":      "User Role",
			"Node.role_type": "Member",
		},
	}
	nodes, err := DecodeDql[[]*model.Node](raw)
	if err != nil {
		t.Fatalf("DecodeDql pointer slice error: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if nodes[0].Nameid != "org##@user" {
		t.Errorf("expected nameid=org##@user, got %s", nodes[0].Nameid)
	}
	if nodes[0].RoleType == nil || *nodes[0].RoleType != model.RoleTypeMember {
		t.Errorf("expected role_type=Member, got %v", nodes[0].RoleType)
	}
}

func TestDecodeDql_NestedStruct(t *testing.T) {
	raw := map[string]any{
		"uid":            "0x10",
		"Post.createdBy": map[string]any{"User.username": "alice"},
		"Tension.action": "NewRole",
		"Tension.emitter": map[string]any{
			"Node.nameid": "org",
		},
		"Tension.receiver": map[string]any{
			"Node.nameid":    "org#circle",
			"Node.mode":      "Coordinated",
			"Node.visibility": "Public",
		},
	}
	tension, err := DecodeDql[model.Tension](raw)
	if err != nil {
		t.Fatalf("DecodeDql nested struct error: %v", err)
	}
	if tension.ID != "0x10" {
		t.Errorf("expected id=0x10, got %s", tension.ID)
	}
}

func TestDecodeDql_EmptySlice(t *testing.T) {
	raw := []map[string]any{}
	nodes, err := DecodeDql[[]model.Node](raw)
	if err != nil {
		t.Fatalf("DecodeDql empty slice error: %v", err)
	}
	if len(nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(nodes))
	}
}

func TestDecodeDqlSlice_MultipleRecords(t *testing.T) {
	input := []map[string]any{
		{"name": "alice", "username": "alice123"},
		{"name": "bob", "username": "bob456"},
	}
	results, err := DecodeDqlSlice[map[string]any](input)
	if err != nil {
		t.Fatalf("DecodeDqlSlice error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0]["name"] != "alice" {
		t.Errorf("expected first name=alice, got %v", results[0]["name"])
	}
	if results[1]["username"] != "bob456" {
		t.Errorf("expected second username=bob456, got %v", results[1]["username"])
	}
}

func TestDecodeDqlSlice_EmptySlice(t *testing.T) {
	results, err := DecodeDqlSlice[map[string]any]([]map[string]any{})
	if err != nil {
		t.Fatalf("DecodeDqlSlice error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected empty results, got %d", len(results))
	}
}

func TestDecodeDqlSlice_TypedStruct(t *testing.T) {
	input := []map[string]any{
		{
			"uid":         "0x1",
			"Node.nameid": "org#circle",
			"Node.name":   "Circle",
		},
	}
	results, err := DecodeDqlSlice[model.Node](input)
	if err != nil {
		t.Fatalf("DecodeDqlSlice typed error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Nameid != "org#circle" {
		t.Errorf("expected nameid=org#circle, got %s", results[0].Nameid)
	}
}

func TestFirst_WithItems(t *testing.T) {
	items := []string{"a", "b", "c"}
	got, err := First(items, nil)
	if err != nil {
		t.Fatalf("First returned unexpected error: %v", err)
	}
	if got != "a" {
		t.Errorf("First: expected 'a', got '%s'", got)
	}
}

func TestFirst_EmptySlice(t *testing.T) {
	got, err := First([]int{}, nil)
	if err != nil {
		t.Fatalf("First returned unexpected error: %v", err)
	}
	if got != 0 {
		t.Errorf("First empty: expected zero value, got %d", got)
	}
}

func TestFirst_NilSlice(t *testing.T) {
	got, err := First[string](nil, nil)
	if err != nil {
		t.Fatalf("First returned unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("First nil: expected zero value, got '%s'", got)
	}
}

func TestFirst_PropagatesError(t *testing.T) {
	testErr := fmt.Errorf("test error")
	got, err := First([]string{"a"}, testErr)
	if err != testErr {
		t.Fatalf("First should propagate error, got: %v", err)
	}
	if got != "" {
		t.Errorf("First with error: expected zero value, got '%s'", got)
	}
}

func TestFirst_Struct(t *testing.T) {
	type item struct {
		Name string
		ID   int
	}
	items := []item{{Name: "first", ID: 1}, {Name: "second", ID: 2}}
	got, err := First(items, nil)
	if err != nil {
		t.Fatalf("First returned unexpected error: %v", err)
	}
	if got.Name != "first" || got.ID != 1 {
		t.Errorf("First struct: expected {first 1}, got %+v", got)
	}
}

func TestCleanDqlMap(t *testing.T) {
	input := map[string]any{
		"uid":           "0x5",
		"Tension.title": "Test tension",
		"Node.nameid":   "org#circle",
		"Tension.receiver": map[string]any{
			"uid":         "0x3",
			"Node.nameid": "org#target",
		},
	}
	result := CleanDqlMap(input)
	if result["id"] != "0x5" {
		t.Errorf("expected id=0x5, got %v", result["id"])
	}
	if result["title"] != "Test tension" {
		t.Errorf("expected title='Test tension', got %v", result["title"])
	}
	// Nested map should also be cleaned (deep=true)
	receiver, ok := result["receiver"].(map[string]any)
	if !ok {
		t.Fatalf("expected receiver to be map[string]any, got %T", result["receiver"])
	}
	if receiver["id"] != "0x3" {
		t.Errorf("expected nested id=0x3, got %v", receiver["id"])
	}
	if receiver["nameid"] != "org#target" {
		t.Errorf("expected nested nameid=org#target, got %v", receiver["nameid"])
	}
}
