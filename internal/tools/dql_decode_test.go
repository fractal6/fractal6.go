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
		"uid":             "0x1",
		"Node.nameid":     "org#circle",
		"Node.name":       "Circle",
		"Node.type_":      "Circle",
		"Node.isRoot":     false,
		"Node.mode":       "Coordinated",
		"Node.rights":     float64(0),
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
			"Node.nameid":     "org#circle",
			"Node.mode":       "Coordinated",
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

func TestDecodeDql_MultipleRecords(t *testing.T) {
	input := []map[string]any{
		{"name": "alice", "username": "alice123"},
		{"name": "bob", "username": "bob456"},
	}
	results, err := DecodeDql[[]map[string]any](input)
	if err != nil {
		t.Fatalf("DecodeDql slice error: %v", err)
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

func TestDecodeDql_EmptySliceFromMaps(t *testing.T) {
	results, err := DecodeDql[[]map[string]any]([]map[string]any{})
	if err != nil {
		t.Fatalf("DecodeDql slice error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected empty results, got %d", len(results))
	}
}

func TestDecodeDql_TypedStructSlice(t *testing.T) {
	input := []map[string]any{
		{
			"uid":         "0x1",
			"Node.nameid": "org#circle",
			"Node.name":   "Circle",
		},
	}
	results, err := DecodeDql[[]model.Node](input)
	if err != nil {
		t.Fatalf("DecodeDql typed slice error: %v", err)
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

func TestDedupe(t *testing.T) {
	type item struct {
		Name string
		Val  int
	}
	items := []item{
		{"a", 1}, {"b", 2}, {"a", 3}, {"c", 4}, {"b", 5},
	}
	got := Dedupe(items, func(i item) string { return i.Name })
	if len(got) != 3 {
		t.Fatalf("expected 3 items, got %d", len(got))
	}
	// First occurrence wins
	if got[0].Val != 1 || got[1].Val != 2 || got[2].Val != 4 {
		t.Errorf("expected vals [1,2,4], got [%d,%d,%d]", got[0].Val, got[1].Val, got[2].Val)
	}
}

func TestDedupe_Empty(t *testing.T) {
	got := Dedupe([]string{}, func(s string) string { return s })
	if len(got) != 0 {
		t.Errorf("expected empty, got %d", len(got))
	}
}

// Tests below verify that DecodeDql correctly decodes the DQL response
// shapes produced by refactored functions in db/dql.go.

func TestDecodeDql_TensionSearchData(t *testing.T) {
	// Simulates the DQL response shape from getTensionSearchData template:
	//   Tension.labels { Label.name }
	//   Tension.comments(first:1) { message: Post.message }
	type tensionSearchData struct {
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
		Comments []struct {
			Message string `json:"message"`
		} `json:"comments"`
	}

	t.Run("with_data", func(t *testing.T) {
		raw := map[string]any{
			"Tension.labels": []any{
				map[string]any{"Label.name": "bug"},
				map[string]any{"Label.name": "urgent"},
			},
			"Tension.comments": []any{
				map[string]any{"message": "First comment"},
			},
		}
		data, err := DecodeDql[tensionSearchData](raw)
		if err != nil {
			t.Fatalf("DecodeDql error: %v", err)
		}
		if len(data.Labels) != 2 {
			t.Fatalf("expected 2 labels, got %d", len(data.Labels))
		}
		if data.Labels[0].Name != "bug" {
			t.Errorf("expected first label=bug, got %s", data.Labels[0].Name)
		}
		if data.Labels[1].Name != "urgent" {
			t.Errorf("expected second label=urgent, got %s", data.Labels[1].Name)
		}
		if len(data.Comments) != 1 || data.Comments[0].Message != "First comment" {
			t.Errorf("expected comment='First comment', got %+v", data.Comments)
		}
	})

	t.Run("empty_result", func(t *testing.T) {
		raw := map[string]any{}
		data, err := DecodeDql[tensionSearchData](raw)
		if err != nil {
			t.Fatalf("DecodeDql error: %v", err)
		}
		if len(data.Labels) != 0 {
			t.Errorf("expected 0 labels, got %d", len(data.Labels))
		}
		if len(data.Comments) != 0 {
			t.Errorf("expected 0 comments, got %d", len(data.Comments))
		}
	})

	t.Run("labels_only", func(t *testing.T) {
		raw := map[string]any{
			"Tension.labels": []any{
				map[string]any{"Label.name": "feature"},
			},
		}
		data, err := DecodeDql[tensionSearchData](raw)
		if err != nil {
			t.Fatalf("DecodeDql error: %v", err)
		}
		if len(data.Labels) != 1 || data.Labels[0].Name != "feature" {
			t.Errorf("expected [feature], got %+v", data.Labels)
		}
		if len(data.Comments) != 0 {
			t.Errorf("expected 0 comments, got %d", len(data.Comments))
		}
	})
}

func TestDecodeDql_TensionBlobRef(t *testing.T) {
	// Simulates the DQL response shape from getLastBlobId template:
	//   Tension.blobs(orderdesc: Post.createdAt, first: 1) { uid }
	type tensionBlobRef struct {
		Blobs []struct {
			ID string `json:"id"`
		} `json:"blobs"`
	}

	t.Run("with_blob", func(t *testing.T) {
		raw := map[string]any{
			"Tension.blobs": []any{
				map[string]any{"uid": "0xabc"},
			},
		}
		data, err := DecodeDql[tensionBlobRef](raw)
		if err != nil {
			t.Fatalf("DecodeDql error: %v", err)
		}
		if len(data.Blobs) != 1 || data.Blobs[0].ID != "0xabc" {
			t.Errorf("expected blob id=0xabc, got %+v", data.Blobs)
		}
	})

	t.Run("empty_blobs", func(t *testing.T) {
		raw := map[string]any{
			"Tension.blobs": []any{},
		}
		data, err := DecodeDql[tensionBlobRef](raw)
		if err != nil {
			t.Fatalf("DecodeDql error: %v", err)
		}
		if len(data.Blobs) != 0 {
			t.Errorf("expected 0 blobs, got %d", len(data.Blobs))
		}
	})
}

func TestDecodeDql_NodeChildRefs(t *testing.T) {
	// Simulates the DQL response shape from getCoordos template:
	//   Node.children @filter(...) { uid }
	type nodeChildRefs struct {
		Children []struct {
			ID string `json:"id"`
		} `json:"children"`
	}

	t.Run("has_children", func(t *testing.T) {
		raw := map[string]any{
			"Node.children": []any{
				map[string]any{"uid": "0x1"},
				map[string]any{"uid": "0x2"},
			},
		}
		data, err := DecodeDql[nodeChildRefs](raw)
		if err != nil {
			t.Fatalf("DecodeDql error: %v", err)
		}
		if len(data.Children) != 2 {
			t.Fatalf("expected 2 children, got %d", len(data.Children))
		}
		if data.Children[0].ID != "0x1" {
			t.Errorf("expected first child id=0x1, got %s", data.Children[0].ID)
		}
	})

	t.Run("no_children", func(t *testing.T) {
		raw := map[string]any{}
		data, err := DecodeDql[nodeChildRefs](raw)
		if err != nil {
			t.Fatalf("DecodeDql error: %v", err)
		}
		if len(data.Children) != 0 {
			t.Errorf("expected 0 children, got %d", len(data.Children))
		}
	})
}

func TestDecodeDql_NodeChildNameids(t *testing.T) {
	// Simulates the DQL response shape from getChildren template:
	//   Node.children @filter(...) { Node.nameid }
	type nodeChildNameids struct {
		Children []struct {
			Nameid string `json:"nameid"`
		} `json:"children"`
	}

	t.Run("with_children", func(t *testing.T) {
		raw := map[string]any{
			"Node.children": []any{
				map[string]any{"Node.nameid": "org#circle1"},
				map[string]any{"Node.nameid": "org#circle2"},
				map[string]any{"Node.nameid": "org##@user"},
			},
		}
		data, err := DecodeDql[nodeChildNameids](raw)
		if err != nil {
			t.Fatalf("DecodeDql error: %v", err)
		}
		if len(data.Children) != 3 {
			t.Fatalf("expected 3 children, got %d", len(data.Children))
		}
		expected := []string{"org#circle1", "org#circle2", "org##@user"}
		for i, want := range expected {
			if data.Children[i].Nameid != want {
				t.Errorf("child[%d]: expected %s, got %s", i, want, data.Children[i].Nameid)
			}
		}
	})

	t.Run("empty", func(t *testing.T) {
		raw := map[string]any{
			"Node.children": []any{},
		}
		data, err := DecodeDql[nodeChildNameids](raw)
		if err != nil {
			t.Fatalf("DecodeDql error: %v", err)
		}
		if len(data.Children) != 0 {
			t.Errorf("expected 0 children, got %d", len(data.Children))
		}
	})
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
