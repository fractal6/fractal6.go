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

package db_test

import (
	"encoding/json"
	"strings"
	"testing"

	. "fractale/fractal6.go/db"
	"fractale/fractal6.go/graph/model"
)

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

func TestGetFieldByEq_Integration(t *testing.T) {
	t.Parallel()
	val, err := GetDB().GetFieldByEq("Node.nameid", "test-org", "Node.name")
	if err != nil {
		t.Fatalf("GetFieldByEq returned error: %v", err)
	}
	name, ok := val.(string)
	if !ok {
		t.Fatalf("GetFieldByEq returned type %T, want string", val)
	}
	if name != "Test Org" {
		t.Errorf("GetFieldByEq(Node.name) = %q, want %q", name, "Test Org")
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

func TestGetUserRoles_Integration(t *testing.T) {
	t.Parallel()
	roles, err := Meta[*model.Node]("getUserRoles", map[string]string{"userid": "testuser"})
	if err != nil {
		t.Fatalf("GetUserRoles returned error: %v", err)
	}
	t.Logf("GetUserRoles returned %d roles", len(roles))
	for i, r := range roles {
		rtStr := "<nil>"
		if r.RoleType != nil {
			rtStr = string(*r.RoleType)
		}
		t.Logf("  role[%d]: nameid=%q name=%q role_type=%s", i, r.Nameid, r.Name, rtStr)
	}
	if len(roles) < 1 {
		t.Errorf("expected at least 1 role, got %d", len(roles))
	}
}

func TestMeta_IntegrationGetNodeHistory(t *testing.T) {
	t.Parallel()
	results, err := GetDB().Meta("getNodeHistory", map[string]string{
		"nameid": "test-org",
		"query":  "",
	})
	if err != nil {
		t.Fatalf("Meta(getNodeHistory) returned error: %v", err)
	}
	if len(results) < 1 {
		t.Errorf("Meta(getNodeHistory) returned %d results, want >= 1", len(results))
	}
	t.Logf("Meta(getNodeHistory) returned %d events", len(results))
}

// getTensionUID is a test helper that returns the UID of the seed "Test tension".
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

func TestGetTensionSearchData_Integration(t *testing.T) {
	t.Parallel()
	tid := getTensionUID(t)

	t.Run("returns_labels_and_comment", func(t *testing.T) {
		t.Parallel()
		labels, firstComment, err := GetDB().GetTensionSearchData(tid)
		if err != nil {
			t.Fatalf("GetTensionSearchData returned error: %v", err)
		}

		// Seed data has label "bug" on this tension.
		if len(labels) != 1 || labels[0] != "bug" {
			t.Errorf("labels = %v, want [bug]", labels)
		}

		// Seed data has a comment "This is the first comment on the test tension".
		if firstComment != "This is the first comment on the test tension" {
			t.Errorf("firstComment = %q, want %q", firstComment, "This is the first comment on the test tension")
		}
	})

	t.Run("nonexistent_uid", func(t *testing.T) {
		t.Parallel()
		labels, comment, err := GetDB().GetTensionSearchData("0xdeadbeef")
		if err != nil {
			t.Fatalf("GetTensionSearchData returned error: %v", err)
		}
		if len(labels) != 0 {
			t.Errorf("labels = %v, want empty", labels)
		}
		if comment != "" {
			t.Errorf("firstComment = %q, want empty", comment)
		}
	})
}

func TestGetTensions_PatternMatchesMessage_Integration(t *testing.T) {
	t.Parallel()

	// The seed data sets Post.message on the test tension to:
	//   "---\nbug\n---\n\nThis is the first comment on the test tension"
	// This lets us test searching by title vs message content.

	status := model.TensionStatusOpen
	q := TensionQuery{
		Nameids:  []string{"test-org"},
		First:    10,
		Offset:   0,
		Status:   &status,
		Username: "testuser",
	}

	t.Run("match_title", func(t *testing.T) {
		t.Parallel()
		pattern := "tension"
		qc := q
		qc.Pattern = &pattern
		tensions, err := GetDB().GetTensions(qc, "int")
		if err != nil {
			t.Fatalf("GetTensions returned error: %v", err)
		}
		if len(tensions) == 0 {
			t.Error("GetTensions with title pattern returned 0 results, want >= 1")
		}
	})

	t.Run("match_message", func(t *testing.T) {
		t.Parallel()
		// "comment" appears only in Post.message, not in Tension.title
		pattern := "comment"
		qc := q
		qc.Pattern = &pattern
		tensions, err := GetDB().GetTensions(qc, "int")
		if err != nil {
			t.Fatalf("GetTensions returned error: %v", err)
		}
		if len(tensions) == 0 {
			t.Error("GetTensions with message pattern returned 0 results, want >= 1")
		}
	})

	t.Run("no_match", func(t *testing.T) {
		t.Parallel()
		pattern := "zzzznonexistent"
		qc := q
		qc.Pattern = &pattern
		tensions, err := GetDB().GetTensions(qc, "int")
		if err != nil {
			t.Fatalf("GetTensions returned error: %v", err)
		}
		if len(tensions) != 0 {
			t.Errorf("GetTensions with non-matching pattern returned %d results, want 0", len(tensions))
		}
	})
}

func TestFormatTensionIntExtMap_PatternFilter(t *testing.T) {
	t.Parallel()
	pattern := "search term"
	q := TensionQuery{
		Nameids: []string{"test-org"},
		First:   10,
		Pattern: &pattern,
		Username: "testuser",
	}
	maps, err := FormatTensionIntExtMap(q)
	if err != nil {
		t.Fatalf("FormatTensionIntExtMap returned error: %v", err)
	}
	tf := (*maps)["tensionFilter"]
	if !strings.Contains(tf, "anyoftext(Tension.title,") {
		t.Errorf("tensionFilter missing Tension.title check: %s", tf)
	}
	if !strings.Contains(tf, "anyoftext(Post.message,") {
		t.Errorf("tensionFilter missing Post.message check: %s", tf)
	}
	if !strings.Contains(tf, " OR ") {
		t.Errorf("tensionFilter missing OR between title and message: %s", tf)
	}
}
