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

func TestGetFieldById_Integration(t *testing.T) {
	t.Parallel()
	tid := getTensionUID(t)

	t.Run("single_field", func(t *testing.T) {
		t.Parallel()
		val, err := GetDB().GetFieldById(tid, "Tension.title")
		if err != nil {
			t.Fatalf("GetFieldById returned error: %v", err)
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
		val, err := GetDB().GetFieldById(tid, "Tension.title Tension.status")
		if err != nil {
			t.Fatalf("GetFieldById returned error: %v", err)
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
		val, err := GetDB().GetFieldById("0xdeadbeef", "Tension.title")
		if err != nil {
			t.Fatalf("GetFieldById returned error: %v", err)
		}
		if val != nil {
			t.Errorf("expected nil for nonexistent uid, got %v", val)
		}
	})
}

func TestGetFieldByEq_MultiField_Integration(t *testing.T) {
	t.Parallel()
	val, err := GetDB().GetFieldByEq("Node.nameid", "test-org", "Node.name Node.about")
	if err != nil {
		t.Fatalf("GetFieldByEq returned error: %v", err)
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

func TestGetSubFieldById_Integration(t *testing.T) {
	t.Parallel()
	tid := getTensionUID(t)

	t.Run("scalar_result", func(t *testing.T) {
		t.Parallel()
		val, err := GetDB().GetSubFieldById(tid, "Post.createdBy", "User.username")
		if err != nil {
			t.Fatalf("GetSubFieldById returned error: %v", err)
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
		val, err := GetDB().GetSubFieldById("0xdeadbeef", "Post.createdBy", "User.username")
		if err != nil {
			t.Fatalf("GetSubFieldById returned error: %v", err)
		}
		if val != nil {
			t.Errorf("expected nil for nonexistent uid, got %v", val)
		}
	})
}

func TestGetSubFieldByEq_Integration(t *testing.T) {
	t.Parallel()

	t.Run("scalar_result", func(t *testing.T) {
		t.Parallel()
		// test-org##@testuser has first_link -> testuser
		val, err := GetDB().GetSubFieldByEq("Node.nameid", "test-org##@testuser", "Node.first_link", "User.username")
		if err != nil {
			t.Fatalf("GetSubFieldByEq returned error: %v", err)
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
		val, err := GetDB().GetSubFieldByEq("Node.nameid", "test-org", "Node.children", "Node.nameid")
		if err != nil {
			t.Fatalf("GetSubFieldByEq returned error: %v", err)
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
		val, err := GetDB().GetSubFieldByEq("Node.nameid", "nonexistent-org", "Node.first_link", "User.username")
		if err != nil {
			t.Fatalf("GetSubFieldByEq returned error: %v", err)
		}
		if val != nil {
			t.Errorf("expected nil for nonexistent node, got %v", val)
		}
	})
}

func TestGetFieldByEqWithFilter_Integration(t *testing.T) {
	t.Parallel()

	t.Run("found", func(t *testing.T) {
		t.Parallel()
		// Query the "private-project" project by nameid + parentnameid filter
		val, err := GetDB().GetFieldByEq(
			"Project.nameid", "private-project",
			"uid Project.rootnameid",
			"Project.parentnameid", "sec-org#private-circle",
		)
		if err != nil {
			t.Fatalf("GetFieldByEq with filter returned error: %v", err)
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
		val, err := GetDB().GetFieldByEq(
			"Project.nameid", "nonexistent-project",
			"uid Project.rootnameid",
			"Project.parentnameid", "sec-org",
		)
		if err != nil {
			t.Fatalf("GetFieldByEq with filter returned error: %v", err)
		}
		if val != nil {
			t.Errorf("expected nil for nonexistent project, got %v", val)
		}
	})
}

func TestGetSubSubFieldByEq_Integration(t *testing.T) {
	t.Parallel()

	t.Run("found", func(t *testing.T) {
		t.Parallel()
		// test-org has source (Blob) -> tension -> uid
		val, err := GetDB().GetSubSubFieldByEq("Node.nameid", "test-org", "Node.source", "Blob.tension", "uid")
		if err != nil {
			t.Fatalf("GetSubSubFieldByEq returned error: %v", err)
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
		val, err := GetDB().GetSubSubFieldByEq("Node.nameid", "nonexistent-org", "Node.source", "Blob.tension", "uid")
		if err != nil {
			t.Fatalf("GetSubSubFieldByEq returned error: %v", err)
		}
		if val != nil {
			t.Errorf("expected nil for nonexistent node, got %v", val)
		}
	})
}

func TestGetParents_Integration(t *testing.T) {
	t.Parallel()

	t.Run("child_role", func(t *testing.T) {
		t.Parallel()
		// test-org##@testuser's parent is test-org
		parents, err := GetDB().GetParents("test-org##@testuser")
		if err != nil {
			t.Fatalf("GetParents returned error: %v", err)
		}
		if len(parents) == 0 {
			t.Fatal("GetParents returned empty, want at least test-org")
		}
		found := false
		for _, p := range parents {
			if p == "test-org" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("parents = %v, expected to contain %q", parents, "test-org")
		}
	})

	t.Run("root_node", func(t *testing.T) {
		t.Parallel()
		// test-org is a root node, should have no parents
		parents, err := GetDB().GetParents("test-org")
		if err != nil {
			t.Fatalf("GetParents returned error: %v", err)
		}
		if len(parents) != 0 {
			t.Errorf("root node parents = %v, want empty", parents)
		}
	})

	t.Run("nested_child", func(t *testing.T) {
		t.Parallel()
		// sec-org#secret-circle#:coordo -> sec-org#secret-circle -> sec-org
		parents, err := GetDB().GetParents("sec-org#secret-circle#:coordo")
		if err != nil {
			t.Fatalf("GetParents returned error: %v", err)
		}
		if len(parents) < 2 {
			t.Errorf("expected >= 2 parents in chain, got %v", parents)
		}
	})
}

func TestGetTopLabels_Integration(t *testing.T) {
	t.Parallel()
	// test-org has label "bug" directly attached.
	// Note: fieldid is "nameid" (not "Node.nameid") because the DQL template builds "Node.{{.fieldid}}".
	labels, err := GetDB().GetTopLabels("nameid", "test-org", true)
	if err != nil {
		t.Fatalf("GetTopLabels returned error: %v", err)
	}
	if len(labels) == 0 {
		t.Fatal("GetTopLabels returned empty, expected at least 'bug'")
	}
	found := false
	for _, l := range labels {
		if l.Name == "bug" {
			found = true
			if l.Color == nil || *l.Color != "#d73a4a" {
				t.Errorf("bug label color = %v, want %q", l.Color, "#d73a4a")
			}
		}
	}
	if !found {
		names := make([]string, len(labels))
		for i, l := range labels {
			names[i] = l.Name
		}
		t.Errorf("labels = %v, expected to contain %q", names, "bug")
	}
}

func TestGetTopTensionTemplates_Integration(t *testing.T) {
	t.Parallel()
	// test-org has tension template "bug-report" directly attached.
	templates, err := GetDB().GetTopTensionTemplates("nameid", "test-org", true)
	if err != nil {
		t.Fatalf("GetTopTensionTemplates returned error: %v", err)
	}
	if len(templates) == 0 {
		t.Fatal("GetTopTensionTemplates returned empty, expected at least 'bug-report'")
	}
	found := false
	for _, tt := range templates {
		if tt.Name == "bug-report" {
			found = true
			if tt.Title != "Bug: " {
				t.Errorf("bug-report title = %q, want %q", tt.Title, "Bug: ")
			}
			if !tt.IsRecursive {
				t.Error("bug-report is_recursive = false, want true")
			}
		}
	}
	if !found {
		names := make([]string, len(templates))
		for i, tt := range templates {
			names[i] = tt.Name
		}
		t.Errorf("tension_templates = %v, expected to contain %q", names, "bug-report")
	}
}

func TestGetSubTensionTemplates_Integration(t *testing.T) {
	t.Parallel()
	// test-org has tension template "bug-report" attached — sub query from root should find it.
	templates, err := GetDB().GetSubTensionTemplates("nameid", "test-org", true)
	if err != nil {
		t.Fatalf("GetSubTensionTemplates returned error: %v", err)
	}
	if len(templates) == 0 {
		t.Fatal("GetSubTensionTemplates returned empty, expected at least 'bug-report'")
	}
	found := false
	for _, tt := range templates {
		if tt.Name == "bug-report" {
			found = true
		}
	}
	if !found {
		names := make([]string, len(templates))
		for i, tt := range templates {
			names[i] = tt.Name
		}
		t.Errorf("tension_templates = %v, expected to contain %q", names, "bug-report")
	}
}

// TestGetTopTensionTemplates_FromChild queries from a child role node (test-org##:coordo)
// and verifies the recursive parent traversal finds "bug-report" (is_recursive=true)
// but excludes "local-only" (is_recursive=false).
func TestGetTopTensionTemplates_FromChild(t *testing.T) {
	t.Parallel()
	templates, err := GetDB().GetTopTensionTemplates("nameid", "test-org##:coordo", false)
	if err != nil {
		t.Fatalf("GetTopTensionTemplates from child: %v", err)
	}

	foundRecursive := false
	foundLocal := false
	for _, tt := range templates {
		switch tt.Name {
		case "bug-report":
			foundRecursive = true
		case "local-only":
			foundLocal = true
		}
	}

	if !foundRecursive {
		names := make([]string, len(templates))
		for i, tt := range templates {
			names[i] = tt.Name
		}
		t.Errorf("top templates from child = %v, expected recursive template %q to be present", names, "bug-report")
	}
	if foundLocal {
		t.Errorf("top templates from child should NOT contain non-recursive template %q, but it was returned", "local-only")
	}
}

// TestGetTopTensionTemplates_ExcludesNonRecursive queries from test-org itself with includeSelf=false
// and verifies that even from a direct parent perspective, the is_recursive filter applies.
func TestGetTopTensionTemplates_ExcludesNonRecursive(t *testing.T) {
	t.Parallel()
	// Query from the coordo role, excludeSelf=true (i.e. include the coordo node itself,
	// but it has no templates — only parent test-org does).
	templates, err := GetDB().GetTopTensionTemplates("nameid", "test-org##:coordo", true)
	if err != nil {
		t.Fatalf("GetTopTensionTemplates (non-recursive check): %v", err)
	}

	for _, tt := range templates {
		if !tt.IsRecursive {
			t.Errorf("top query returned non-recursive template %q (is_recursive=false); only recursive templates should appear", tt.Name)
		}
	}
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
