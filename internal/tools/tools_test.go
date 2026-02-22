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
	"reflect"
	"testing"

	"fractale/fractal6.go/graph/model"
	. "fractale/fractal6.go/internal/tools"
)

func TestCleanDqlMap_BasicKeys(t *testing.T) {
	input := map[string]any{
		"User.name":     "alice",
		"User.username": "alice123",
	}
	got := CleanDqlMap(input)
	want := map[string]any{
		"name":     "alice",
		"username": "alice123",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("CleanDqlMap basic keys: got %v, want %v", got, want)
	}
}

func TestCleanDqlMap_UidToId(t *testing.T) {
	input := map[string]any{
		"uid":       "0x1",
		"Node.name": "root",
	}
	got := CleanDqlMap(input)
	if got["id"] != "0x1" {
		t.Errorf("expected uid->id mapping, got %v", got)
	}
	if got["name"] != "root" {
		t.Errorf("expected name key, got %v", got)
	}
}

func TestCleanDqlMap_NestedMap(t *testing.T) {
	input := map[string]any{
		"Node.nameid": "org#circle",
		"Node.parent": map[string]any{
			"Node.nameid": "org",
			"uid":         "0x2",
		},
	}
	got := CleanDqlMap(input)
	nested := got["parent"].(map[string]any)
	if nested["nameid"] != "org" {
		t.Errorf("should clean composite names in nested maps, got %v", nested)
	}
	if nested["id"] != "0x2" {
		t.Errorf("should rename uid->id in nested maps, got %v", nested)
	}
}

func TestCleanDqlMap_ArrayOfMaps(t *testing.T) {
	input := map[string]any{
		"Node.children": []any{
			map[string]any{
				"Node.nameid": "child1",
				"uid":         "0x3",
			},
		},
	}
	got := CleanDqlMap(input)
	children := got["children"].([]any)
	child := children[0].(map[string]any)
	if child["nameid"] != "child1" {
		t.Errorf("array items should be cleaned, got %v", child)
	}
	if child["id"] != "0x3" {
		t.Errorf("array items should rename uid->id, got %v", child)
	}
}

func TestCleanAliasedMap_TrailingDigits(t *testing.T) {
	input := map[string]any{
		"name1":  "alice",
		"name2":  "bob",
		"status": "Open",
	}
	got := CleanAliasedMap(input)
	// "name1" -> "name", "name2" -> "name" (last one wins), "status" unchanged
	if got["status"] != "Open" {
		t.Errorf("non-digit key should be unchanged, got %v", got)
	}
	if _, hasName := got["name"]; !hasName {
		t.Errorf("trailing digits should be stripped, got %v", got)
	}
}

func TestCleanAliasedMap_NestedMap(t *testing.T) {
	input := map[string]any{
		"data": map[string]any{
			"field1": "value",
		},
	}
	got := CleanAliasedMap(input)
	nested := got["data"].(map[string]any)
	if nested["field"] != "value" {
		t.Errorf("nested map trailing digits should be stripped, got %v", nested)
	}
}

func TestStructMap_Roundtrip(t *testing.T) {
	name := "Test"
	input := model.Label{
		Name:       "test-label",
		Rootnameid: "org",
		Color:      &name,
	}
	m := StructMap[map[string]any](input)
	if m["name"] != "test-label" {
		t.Errorf("expected name=test-label in map, got %v", m["name"])
	}
	if m["rootnameid"] != "org" {
		t.Errorf("expected rootnameid=org in map, got %v", m["rootnameid"])
	}
}
