//go:build integration

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

// Tests for generic DQL primitives: Count, Exists, IsChild, GetByUid,
// GetByEq, GetByEqFiltered. Tests for named-template queries live in
// integration_query_test.go.

package db_test

import (
	"encoding/json"
	"testing"

	. "fractale/fractal6.go/db"
)

// getTensionUID is shared by tests that need a tension uid from the seed data.
func getTensionUID(t *testing.T) string {
	t.Helper()
	tids, err := GetDB().GetIDs("Tension.title", "Test tension", nil, nil)
	if err != nil {
		t.Fatalf("GetIDs(Tension.title) returned error: %v", err)
	}
	if len(tids) == 0 {
		t.Fatal("No tension found with title 'Test tension'")
	}
	return tids[0]
}

func TestCountHas_Integration(t *testing.T) {
	t.Parallel()
	count := GetDB().CountHas("Node.nameid")
	if count < 1 {
		t.Errorf("CountHas(Node.nameid) = %d, want >= 1", count)
	}
	t.Logf("CountHas(Node.nameid) = %d", count)
}

func TestExists_Integration(t *testing.T) {
	t.Parallel()

	t.Run("found", func(t *testing.T) {
		t.Parallel()
		found, err := GetDB().Exists("Node.nameid", "test-org", nil)
		if err != nil {
			t.Fatalf("Exists returned error: %v", err)
		}
		if !found {
			t.Error("Exists(Node.nameid, test-org) = false, want true")
		}
	})

	t.Run("not_found", func(t *testing.T) {
		t.Parallel()
		found, err := GetDB().Exists("Node.nameid", "nonexistent-org#", nil)
		if err != nil {
			t.Fatalf("Exists returned error: %v", err)
		}
		if found {
			t.Error("Exists(Node.nameid, nonexistent-org#) = true, want false")
		}
	})
}

func TestGetByEq_Integration(t *testing.T) {
	t.Parallel()
	val, err := GetDB().GetByEq("Node.nameid", "test-org", "Node.name")
	if err != nil {
		t.Fatalf("GetByEq returned error: %v", err)
	}
	name, ok := val.(string)
	if !ok {
		t.Fatalf("GetByEq returned type %T, want string", val)
	}
	if name != "Test Org" {
		t.Errorf("GetByEq(Node.name) = %q, want %q", name, "Test Org")
	}
}

func TestIsChild_Integration(t *testing.T) {
	t.Parallel()

	t.Run("is_child", func(t *testing.T) {
		t.Parallel()
		isChild, err := GetDB().IsChild("test-org", "test-org##@testuser")
		if err != nil {
			t.Fatalf("IsChild returned error: %v", err)
		}
		if !isChild {
			t.Error("IsChild(test-org, test-org##@testuser) = false, want true")
		}
	})

	t.Run("not_child", func(t *testing.T) {
		t.Parallel()
		isChild, err := GetDB().IsChild("test-org", "nonexistent#")
		if err != nil {
			t.Fatalf("IsChild returned error: %v", err)
		}
		if isChild {
			t.Error("IsChild(test-org, nonexistent#) = true, want false")
		}
	})
}

func TestGetChildren_Integration(t *testing.T) {
	t.Parallel()
	children, err := GetDB().GetChildren("test-org")
	if err != nil {
		t.Fatalf("GetChildren returned error: %v", err)
	}
	if len(children) < 2 {
		t.Errorf("GetChildren(test-org) returned %d children, want >= 2", len(children))
	}
	t.Logf("GetChildren(test-org) = %v", children)
}

func TestHasCoordos_Integration(t *testing.T) {
	t.Parallel()
	has := GetDB().HasCoordos("test-org")
	if !has {
		t.Error("HasCoordos(test-org) = false, want true")
	}
}

func TestQueryDql_IntegrationRaw(t *testing.T) {
	t.Parallel()
	res, err := GetDB().QueryDql("exists", map[string]string{
		"fieldName": "Node.nameid",
		"value":     "test-org",
		"filter":    "",
	})
	if err != nil {
		t.Fatalf("QueryDql returned error: %v", err)
	}

	var resp DqlResp
	if err := json.Unmarshal(res.Json, &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if len(resp.All) == 0 {
		t.Error("QueryDql(exists) returned empty result")
	}
}

func TestGetByUid_Integration(t *testing.T) {
	t.Parallel()
	tid := getTensionUID(t)

	t.Run("single_field", func(t *testing.T) {
		t.Parallel()
		val, err := GetDB().GetByUid(tid, "Tension.title")
		if err != nil {
			t.Fatalf("GetByUid returned error: %v", err)
		}
		title, ok := val.(string)
		if !ok {
			t.Fatalf("expected string, got %T", val)
		}
		if title != "Test tension" {
			t.Errorf("Tension.title = %q, want %q", title, "Test tension")
		}
	})

	t.Run("multi_field", func(t *testing.T) {
		t.Parallel()
		val, err := GetDB().GetByUid(tid, "Tension.title Tension.status")
		if err != nil {
			t.Fatalf("GetByUid returned error: %v", err)
		}
		m, ok := val.(map[string]any)
		if !ok {
			t.Fatalf("expected map[string]any, got %T", val)
		}
		if m["title"] != "Test tension" {
			t.Errorf("title = %v, want %q", m["title"], "Test tension")
		}
		if m["status"] != "Open" {
			t.Errorf("status = %v, want %q", m["status"], "Open")
		}
	})

	t.Run("not_found", func(t *testing.T) {
		t.Parallel()
		val, err := GetDB().GetByUid("0xdeadbeef", "Tension.title")
		if err != nil {
			t.Fatalf("GetByUid returned error: %v", err)
		}
		if val != nil {
			t.Errorf("expected nil for nonexistent uid, got %v", val)
		}
	})
}

func TestGetByEq_MultiField_Integration(t *testing.T) {
	t.Parallel()
	val, err := GetDB().GetByEq("Node.nameid", "test-org", "Node.name Node.about")
	if err != nil {
		t.Fatalf("GetByEq returned error: %v", err)
	}
	m, ok := val.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", val)
	}
	if m["name"] != "Test Org" {
		t.Errorf("name = %v, want %q", m["name"], "Test Org")
	}
	if m["about"] != "A test organisation" {
		t.Errorf("about = %v, want %q", m["about"], "A test organisation")
	}
}

func TestGetByUid_Sub_Integration(t *testing.T) {
	t.Parallel()
	tid := getTensionUID(t)

	t.Run("scalar_result", func(t *testing.T) {
		t.Parallel()
		val, err := GetDB().GetByUid(tid, "Post.createdBy", "User.username")
		if err != nil {
			t.Fatalf("GetByUid returned error: %v", err)
		}
		username, ok := val.(string)
		if !ok {
			t.Fatalf("expected string, got %T", val)
		}
		if username != "testuser" {
			t.Errorf("createdBy username = %q, want %q", username, "testuser")
		}
	})

	t.Run("not_found", func(t *testing.T) {
		t.Parallel()
		val, err := GetDB().GetByUid("0xdeadbeef", "Post.createdBy", "User.username")
		if err != nil {
			t.Fatalf("GetByUid returned error: %v", err)
		}
		if val != nil {
			t.Errorf("expected nil for nonexistent uid, got %v", val)
		}
	})
}

func TestGetByEq_Sub_Integration(t *testing.T) {
	t.Parallel()

	t.Run("scalar_result", func(t *testing.T) {
		t.Parallel()
		// test-org##@testuser has first_link -> testuser
		val, err := GetDB().GetByEq("Node.nameid", "test-org##@testuser", "Node.first_link", "User.username")
		if err != nil {
			t.Fatalf("GetByEq returned error: %v", err)
		}
		username, ok := val.(string)
		if !ok {
			t.Fatalf("expected string, got %T", val)
		}
		if username != "testuser" {
			t.Errorf("first_link username = %q, want %q", username, "testuser")
		}
	})

	t.Run("list_result", func(t *testing.T) {
		t.Parallel()
		// test-org has children (roles + coordo), so children returns a list
		val, err := GetDB().GetByEq("Node.nameid", "test-org", "Node.children", "Node.nameid")
		if err != nil {
			t.Fatalf("GetByEq returned error: %v", err)
		}
		list, ok := val.([]any)
		if !ok {
			t.Fatalf("expected []any, got %T", val)
		}
		if len(list) < 2 {
			t.Errorf("expected >= 2 children, got %d", len(list))
		}
	})

	t.Run("not_found", func(t *testing.T) {
		t.Parallel()
		val, err := GetDB().GetByEq("Node.nameid", "nonexistent-org", "Node.first_link", "User.username")
		if err != nil {
			t.Fatalf("GetByEq returned error: %v", err)
		}
		if val != nil {
			t.Errorf("expected nil for nonexistent node, got %v", val)
		}
	})
}

func TestGetByEqFiltered_Integration(t *testing.T) {
	t.Parallel()

	t.Run("found", func(t *testing.T) {
		t.Parallel()
		// Query the "private-project" project by nameid + parentnameid filter
		val, err := GetDB().GetByEqFiltered(
			"Project.nameid", "private-project",
			"Project.parentnameid", "sec-org#private-circle",
			"uid Project.rootnameid",
		)
		if err != nil {
			t.Fatalf("GetByEqFiltered returned error: %v", err)
		}
		m, ok := val.(map[string]any)
		if !ok {
			t.Fatalf("expected map[string]any, got %T", val)
		}
		uid, _ := m["id"].(string)
		if uid == "" {
			t.Error("expected non-empty uid")
		}
		rootnameid, _ := m["rootnameid"].(string)
		if rootnameid != "sec-org" {
			t.Errorf("rootnameid = %q, want %q", rootnameid, "sec-org")
		}
	})

	t.Run("not_found", func(t *testing.T) {
		t.Parallel()
		val, err := GetDB().GetByEqFiltered(
			"Project.nameid", "nonexistent-project",
			"Project.parentnameid", "sec-org",
			"uid Project.rootnameid",
		)
		if err != nil {
			t.Fatalf("GetByEqFiltered returned error: %v", err)
		}
		if val != nil {
			t.Errorf("expected nil for nonexistent project, got %v", val)
		}
	})
}

func TestGetByEq_SubSub_Integration(t *testing.T) {
	t.Parallel()

	t.Run("found", func(t *testing.T) {
		t.Parallel()
		// test-org has source (Blob) -> tension -> uid
		val, err := GetDB().GetByEq("Node.nameid", "test-org", "Node.source", "Blob.tension", "uid")
		if err != nil {
			t.Fatalf("GetByEq returned error: %v", err)
		}
		uid, ok := val.(string)
		if !ok {
			t.Fatalf("expected string, got %T (%v)", val, val)
		}
		if uid == "" {
			t.Error("expected non-empty uid for blob.tension")
		}
	})

	t.Run("not_found", func(t *testing.T) {
		t.Parallel()
		val, err := GetDB().GetByEq("Node.nameid", "nonexistent-org", "Node.source", "Blob.tension", "uid")
		if err != nil {
			t.Fatalf("GetByEq returned error: %v", err)
		}
		if val != nil {
			t.Errorf("expected nil for nonexistent node, got %v", val)
		}
	})
}
