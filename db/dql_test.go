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

package db_test

import (
	"bytes"
	"fmt"
	"testing"
	"text/template"

	. "fractale/fractal6.go/db"
	. "fractale/fractal6.go/internal/tools"
)

// dummyVars provides a value for every template variable used across
// DqlQueries and DqlMutations.
var dummyVars = map[string]string{
	"id":                  "0x1",
	"fieldName":           "Node.name",
	"fieldid":             "Node.nameid",
	"value":               "test",
	"filter":              "",
	"f1":                  "Node.nameid",
	"v1":                  "a",
	"f2":                  "Node.isRoot",
	"v2":                  "true",
	"nameid":              "test-org#",
	"nameids":             `"test-org#"`,
	"nameidsProtected":    `"test-org#"`,
	"rootnameid":          "test-org#",
	"rootnameidProtected": "test-org#",
	"nameid_old":          "test-org#old",
	"nameid_new":          "test-org#new",
	"regex":               "test-org.*",
	"payload":             "{ uid }",
	"user_payload":        "{ uid }",
	"userid":              "user1",
	"username":            "user1",
	"email":               "a@b.co",
	"token":               "tok",
	"from":                "test-org#a",
	"to":                  "test-org#b",
	"parent":              "test-org#",
	"child":               "test-org#child",
	"query":               "search",
	"tid":                 "0x2",
	"cid":                 "0x3",
	"colid":               "0x4",
	"old_colid":           "0x5",
	"cardid":              "0x6",
	"ghostid":             "0x7",
	"pos":                 "1",
	"old_pos":             "0",
	"first":               "10",
	"offset":              "0",
	"order":               "orderdesc",
	"orderBy":             "Post.createdAt",
	"now":                 "2026-01-01T00:00:00Z",
	"old_name":            "old",
	"new_name":            "new",
	"k":                   "username",
	"v":                   "user1",
	"tensionFilter":       `@filter(eq(Tension.status, "Open"))`,
	"labelsFilter":        "",
	"authorsFilter":       "",
	"excludeSelf":         "",
	"extra_pre_vars":      "",
	// Variables for converted ad-hoc mutations
	"predicate":  "Node.name",
	"predicate1": "Node.source",
	"predicate2": "Node.nameid",
	"objid":      "test-org#",
	"bid":        "0x8",
	"flag":       "2026-01-01T00:00:00Z",
	"action":     "NewRole",
	"roleType":   "Member",
	"activityid": "u#user1#2026-01-01",
	"ownerid":    "u#user1",
	"date":       "2026-01-01T00:00:00Z",
}

// assertTemplateRenders parses raw as a Go template and verifies it renders
// without error using dummyVars. label is used for error messages.
func assertTemplateRenders(t *testing.T, label, raw string) {
	t.Helper()
	tmpl := CleanString(raw, false)

	parsed, err := template.New(label).Parse(tmpl)
	if err != nil {
		t.Errorf("%s: failed to parse: %v", label, err)
		return
	}

	var buf bytes.Buffer
	if err := parsed.Execute(&buf, dummyVars); err != nil {
		t.Errorf("%s: failed to render: %v", label, err)
		return
	}
	if buf.Len() == 0 {
		t.Errorf("%s: rendered to empty string", label)
	}
}

// TestDqlQueriesRender verifies that every DQL query template parses and
// renders without error.
func TestDqlQueriesRender(t *testing.T) {
	for name, raw := range DqlQueries {
		assertTemplateRenders(t, name, raw)
	}
}

// TestDqlMutationsRender verifies that every DQL mutation template (query,
// set, delete, and condition parts) parses and renders without error.
func TestDqlMutationsRender(t *testing.T) {
	for name, qm := range DqlMutations {
		assertTemplateRenders(t, name+".Q", qm.Q)
		for i, x := range qm.M {
			if x.S != "" {
				assertTemplateRenders(t, fmt.Sprintf("%s.M[%d].S", name, i), x.S)
			}
			if x.D != "" {
				assertTemplateRenders(t, fmt.Sprintf("%s.M[%d].D", name, i), x.D)
			}
			if x.C != "" {
				assertTemplateRenders(t, fmt.Sprintf("%s.M[%d].C", name, i), x.C)
			}
		}
	}
}

func TestDecodeDqlResp_NilResponse(t *testing.T) {
	results, err := DecodeDqlResp(nil)
	if err != nil {
		t.Fatalf("DecodeDqlResp nil error: %v", err)
	}
	if results != nil {
		t.Errorf("expected nil results for nil response, got %v", results)
	}
}

// --- Unit tests for decode helpers ---

func TestDecodeAt_Depth1(t *testing.T) {
	t.Parallel()

	t.Run("single_field", func(t *testing.T) {
		t.Parallel()
		results := []map[string]any{{"name": "Test Org", "about": "desc"}}
		val, err := DecodeAt(results, "Node.name")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != "Test Org" {
			t.Errorf("got %v, want %q", val, "Test Org")
		}
	})

	t.Run("uid_field", func(t *testing.T) {
		t.Parallel()
		results := []map[string]any{{"id": "0x1"}}
		val, err := DecodeAt(results, "uid")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != "0x1" {
			t.Errorf("got %v, want %q", val, "0x1")
		}
	})

	t.Run("multi_field_returns_map", func(t *testing.T) {
		t.Parallel()
		results := []map[string]any{{"name": "Test", "about": "desc"}}
		val, err := DecodeAt(results, "Node.name Node.about")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		m, ok := val.(map[string]any)
		if !ok {
			t.Fatalf("expected map[string]any, got %T", val)
		}
		if m["name"] != "Test" || m["about"] != "desc" {
			t.Errorf("got %v, want name=Test, about=desc", m)
		}
	})

	t.Run("empty_results", func(t *testing.T) {
		t.Parallel()
		val, err := DecodeAt(nil, "Node.name")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != nil {
			t.Errorf("expected nil, got %v", val)
		}
	})

	t.Run("multiple_results_error", func(t *testing.T) {
		t.Parallel()
		results := []map[string]any{{"name": "a"}, {"name": "b"}}
		_, err := DecodeAt(results, "Node.name")
		if err == nil {
			t.Error("expected error for multiple results, got nil")
		}
	})
}

func TestDecodeAt_Depth2(t *testing.T) {
	t.Parallel()

	t.Run("scalar_single_field", func(t *testing.T) {
		t.Parallel()
		results := []map[string]any{{
			"createdBy": map[string]any{"username": "testuser"},
		}}
		val, err := DecodeAt(results, "Post.createdBy", "User.username")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != "testuser" {
			t.Errorf("got %v, want %q", val, "testuser")
		}
	})

	t.Run("scalar_multi_field", func(t *testing.T) {
		t.Parallel()
		results := []map[string]any{{
			"createdBy": map[string]any{"username": "testuser", "name": "Test User"},
		}}
		val, err := DecodeAt(results, "Post.createdBy", "User.username User.name")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		m, ok := val.(map[string]any)
		if !ok {
			t.Fatalf("expected map, got %T", val)
		}
		if m["username"] != "testuser" {
			t.Errorf("username = %v, want %q", m["username"], "testuser")
		}
	})

	t.Run("list_single_field", func(t *testing.T) {
		t.Parallel()
		results := []map[string]any{{
			"assignees": []any{
				map[string]any{"username": "user1"},
				map[string]any{"username": "user2"},
			},
		}}
		val, err := DecodeAt(results, "Tension.assignees", "User.username")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		list, ok := val.([]any)
		if !ok {
			t.Fatalf("expected []any, got %T", val)
		}
		if len(list) != 2 || list[0] != "user1" || list[1] != "user2" {
			t.Errorf("got %v, want [user1 user2]", list)
		}
	})

	t.Run("nil_sub_field", func(t *testing.T) {
		t.Parallel()
		results := []map[string]any{{"parent": nil}}
		val, err := DecodeAt(results, "Node.parent", "Node.nameid")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != nil {
			t.Errorf("expected nil, got %v", val)
		}
	})
}

func TestDecodeAt_Depth3(t *testing.T) {
	t.Parallel()

	t.Run("single_field", func(t *testing.T) {
		t.Parallel()
		results := []map[string]any{{
			"source": map[string]any{
				"tension": map[string]any{"id": "0x42"},
			},
		}}
		val, err := DecodeAt(results, "Node.source", "Blob.tension", "uid")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != "0x42" {
			t.Errorf("got %v, want %q", val, "0x42")
		}
	})

	t.Run("multi_field_returns_map", func(t *testing.T) {
		t.Parallel()
		results := []map[string]any{{
			"source": map[string]any{
				"tension": map[string]any{"id": "0x42", "title": "Hello"},
			},
		}}
		val, err := DecodeAt(results, "Node.source", "Blob.tension", "uid Tension.title")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		m, ok := val.(map[string]any)
		if !ok {
			t.Fatalf("expected map, got %T", val)
		}
		if m["id"] != "0x42" {
			t.Errorf("id = %v, want %q", m["id"], "0x42")
		}
	})

	t.Run("nil_intermediate", func(t *testing.T) {
		t.Parallel()
		results := []map[string]any{{
			"source": map[string]any{"tension": nil},
		}}
		val, err := DecodeAt(results, "Node.source", "Blob.tension", "uid")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != nil {
			t.Errorf("expected nil, got %v", val)
		}
	})
}
