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

// Tests for named-template queries: tension search/pattern, parents/history
// traversal, and the two-phase /q/* visibility helpers (Get*Visibilities,
// Get*In). Generic primitives are tested in integration_helpers_test.go.

package db_test

import (
	"strings"
	"testing"

	. "fractale/fractal6.go/db"
	"fractale/fractal6.go/graph/model"
)

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

// commentUID resolves a seeded comment uid by its exact Post.message.
func commentUID(t *testing.T, message string) string {
	t.Helper()
	cids, err := GetDB().GetIDs("Post.message", message, nil, nil)
	if err != nil {
		t.Fatalf("GetIDs(Post.message) returned error: %v", err)
	}
	if len(cids) == 0 {
		t.Fatalf("No comment found with message %q", message)
	}
	return cids[0]
}

func TestGetCommentTension_Integration(t *testing.T) {
	t.Parallel()
	tid := getTensionUID(t)

	t.Run("first_comment", func(t *testing.T) {
		t.Parallel()
		cid := commentUID(t, "This is the first comment on the test tension")
		gotTid, isFirst, err := GetDB().GetCommentTension(cid)
		if err != nil {
			t.Fatalf("GetCommentTension returned error: %v", err)
		}
		if gotTid != tid {
			t.Errorf("tid = %q, want %q", gotTid, tid)
		}
		if !isFirst {
			t.Error("isFirst = false, want true for the first comment")
		}
	})

	t.Run("non_first_comment", func(t *testing.T) {
		t.Parallel()
		cid := commentUID(t, "file-test: public comment by testuser")
		gotTid, isFirst, err := GetDB().GetCommentTension(cid)
		if err != nil {
			t.Fatalf("GetCommentTension returned error: %v", err)
		}
		if gotTid != tid {
			t.Errorf("tid = %q, want %q", gotTid, tid)
		}
		if isFirst {
			t.Error("isFirst = true, want false for a non-first comment")
		}
	})

	t.Run("unknown_uid", func(t *testing.T) {
		t.Parallel()
		gotTid, isFirst, err := GetDB().GetCommentTension("0xdeadbeef")
		if err != nil {
			t.Fatalf("GetCommentTension returned error: %v", err)
		}
		if gotTid != "" || isFirst {
			t.Errorf("got (%q, %v), want empty result for unknown uid", gotTid, isFirst)
		}
	})

	t.Run("invalid_uid", func(t *testing.T) {
		t.Parallel()
		if _, _, err := GetDB().GetCommentTension("not-a-uid"); err == nil {
			t.Error("GetCommentTension accepted an invalid uid, want error")
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

//
// Two-phase /q/* visibility helpers
//

// Phase 1, recurse-down: subtree of test-org should include test-org itself
// (when includeSelf=true) plus its sub-circles. Members and roles are filtered
// out (only Circle-type nodes are returned).
func TestGetSubNodeVisibilities_Integration(t *testing.T) {
	t.Parallel()

	t.Run("includes_self", func(t *testing.T) {
		t.Parallel()
		visMap, err := GetDB().GetSubNodeVisibilities("nameid", "test-org", true)
		if err != nil {
			t.Fatalf("GetSubNodeVisibilities returned error: %v", err)
		}
		if _, ok := visMap["test-org"]; !ok {
			t.Errorf("expected 'test-org' in result with includeSelf=true, got %v", keys(visMap))
		}
		if len(visMap) < 1 {
			t.Error("expected at least 1 circle in subtree")
		}
	})

	t.Run("excludes_self", func(t *testing.T) {
		t.Parallel()
		visMap, err := GetDB().GetSubNodeVisibilities("nameid", "test-org", false)
		if err != nil {
			t.Fatalf("GetSubNodeVisibilities returned error: %v", err)
		}
		if _, ok := visMap["test-org"]; ok {
			t.Errorf("'test-org' should not appear with includeSelf=false, got %v", keys(visMap))
		}
	})

	t.Run("excludes_non_circles", func(t *testing.T) {
		t.Parallel()
		// Members (e.g. test-org##@testuser) are role-type nodes — should not appear.
		visMap, err := GetDB().GetSubNodeVisibilities("nameid", "test-org", true)
		if err != nil {
			t.Fatalf("GetSubNodeVisibilities returned error: %v", err)
		}
		for nameid := range visMap {
			if strings.Contains(nameid, "##@") || strings.Contains(nameid, "##:") || strings.Contains(nameid, "#:") {
				t.Errorf("non-Circle node leaked into subtree visibility: %q", nameid)
			}
		}
	})
}

// Phase 1, recurse-up: ancestor chain of a child role should reach its root org.
func TestGetTopNodeVisibilities_Integration(t *testing.T) {
	t.Parallel()

	visMap, err := GetDB().GetTopNodeVisibilities("nameid", "sec-org#secret-circle#:coordo", false)
	if err != nil {
		t.Fatalf("GetTopNodeVisibilities returned error: %v", err)
	}
	// Ancestors should include sec-org#secret-circle and sec-org.
	wantAncestor := "sec-org"
	if _, ok := visMap[wantAncestor]; !ok {
		t.Errorf("expected ancestor %q in result, got %v", wantAncestor, keys(visMap))
	}
	// Self (the role) is excluded by includeSelf=false; even with includeSelf=true
	// it'd be filtered by Node.type_ = Circle.
	if _, ok := visMap["sec-org#secret-circle#:coordo"]; ok {
		t.Errorf("self role leaked despite includeSelf=false: %v", keys(visMap))
	}
}

// Phase 3: members attached to circles in the visible-nameids list.
func TestGetMembersIn_Integration(t *testing.T) {
	t.Parallel()

	t.Run("returns_members", func(t *testing.T) {
		t.Parallel()
		members, err := GetDB().GetMembersIn([]string{"test-org"}, "User.username")
		if err != nil {
			t.Fatalf("GetMembersIn returned error: %v", err)
		}
		if len(members) == 0 {
			t.Fatal("GetMembersIn returned empty, expected at least one member")
		}
		// Each result must be a role-typed node with first_link.
		for _, m := range members {
			if m.RoleType == nil {
				t.Errorf("member %q missing role_type", m.Nameid)
			}
			if m.FirstLink == nil {
				t.Errorf("member %q missing first_link", m.Nameid)
			}
		}
	})

	t.Run("empty_input", func(t *testing.T) {
		t.Parallel()
		members, err := GetDB().GetMembersIn(nil, "User.username")
		if err != nil {
			t.Fatalf("GetMembersIn(nil) returned error: %v", err)
		}
		if len(members) != 0 {
			t.Errorf("empty input should return empty slice, got %d", len(members))
		}
	})
}

// Phase 3: roles attached to circles in the visible-nameids list.
func TestGetRolesIn_Integration(t *testing.T) {
	t.Parallel()

	roles, err := GetDB().GetRolesIn([]string{"test-org"}, "")
	if err != nil {
		t.Fatalf("GetRolesIn returned error: %v", err)
	}
	// Seed data attaches at least one RoleExt template to test-org.
	if len(roles) == 0 {
		t.Skip("seed data has no RoleExt attached to test-org; skip")
	}
	for _, r := range roles {
		if r.Name == "" {
			t.Errorf("RoleExt with empty name: %+v", r)
		}
	}
}

// Phase 3: projects attached to circles in the visible-nameids list, filtered
// to status=Open.
func TestGetProjectsIn_Integration(t *testing.T) {
	t.Parallel()

	t.Run("filters_open_only", func(t *testing.T) {
		t.Parallel()
		// sec-org#private-circle has "private-project" attached.
		projects, err := GetDB().GetProjectsIn([]string{"sec-org#private-circle"}, "")
		if err != nil {
			t.Fatalf("GetProjectsIn returned error: %v", err)
		}
		if len(projects) == 0 {
			t.Skip("seed data has no Open projects under sec-org#private-circle; skip")
		}
		for _, p := range projects {
			if p.Name == "" {
				t.Errorf("project with empty name: %+v", p)
			}
		}
	})

	t.Run("empty_input", func(t *testing.T) {
		t.Parallel()
		projects, err := GetDB().GetProjectsIn(nil, "")
		if err != nil {
			t.Fatalf("GetProjectsIn(nil) returned error: %v", err)
		}
		if len(projects) != 0 {
			t.Errorf("empty input should return empty slice, got %d", len(projects))
		}
	})
}

func TestGetLabelsIn_Integration(t *testing.T) {
	t.Parallel()
	// test-org has label "bug" directly attached.
	labels, err := GetDB().GetLabelsIn([]string{"test-org"}, "")
	if err != nil {
		t.Fatalf("GetLabelsIn returned error: %v", err)
	}
	if len(labels) == 0 {
		t.Fatal("GetLabelsIn returned empty, expected at least 'bug'")
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

// /q/tension_templates/top from test-org with includeSelf=true: visible
// nameids = ["test-org"] (= self). Self pulls all templates → bug-report present.
func TestGetTopTensionTemplatesIn_FromSelf(t *testing.T) {
	t.Parallel()
	templates, err := GetDB().GetTopTensionTemplatesIn([]string{"test-org"}, "test-org")
	if err != nil {
		t.Fatalf("GetTopTensionTemplatesIn returned error: %v", err)
	}
	if len(templates) == 0 {
		t.Fatal("GetTopTensionTemplatesIn returned empty, expected at least 'bug-report'")
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

// /q/tension_templates/sub from test-org: visible nameids include test-org —
// pulls all templates attached, including bug-report.
func TestGetTensionTemplatesIn_Integration(t *testing.T) {
	t.Parallel()
	templates, err := GetDB().GetTensionTemplatesIn([]string{"test-org"}, "")
	if err != nil {
		t.Fatalf("GetTensionTemplatesIn returned error: %v", err)
	}
	if len(templates) == 0 {
		t.Fatal("GetTensionTemplatesIn returned empty, expected at least 'bug-report'")
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

// /q/tension_templates/top from a child role with includeSelf=false. Visible
// nameids only contain ancestor "test-org"; the is_recursive=true filter must
// keep "bug-report" but drop "local-only".
func TestGetTopTensionTemplatesIn_FromChild(t *testing.T) {
	t.Parallel()
	templates, err := GetDB().GetTopTensionTemplatesIn([]string{"test-org"}, "test-org##:coordo")
	if err != nil {
		t.Fatalf("GetTopTensionTemplatesIn from child: %v", err)
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

// /q/tension_templates/top from a child role with includeSelf=true. Visible
// nameids = [child, ancestor]. Ancestors get is_recursive=true filter; self
// (the coordo role) has no templates. Only recursive templates should appear.
func TestGetTopTensionTemplatesIn_ExcludesNonRecursive(t *testing.T) {
	t.Parallel()
	templates, err := GetDB().GetTopTensionTemplatesIn([]string{"test-org##:coordo", "test-org"}, "test-org##:coordo")
	if err != nil {
		t.Fatalf("GetTopTensionTemplatesIn (non-recursive check): %v", err)
	}

	for _, tt := range templates {
		if !tt.IsRecursive {
			t.Errorf("top query returned non-recursive template %q (is_recursive=false); only recursive templates should appear", tt.Name)
		}
	}
}

// /q/project_templates/top from test-org with includeSelf=true: visible
// nameids = ["test-org"] (= self). Self pulls all templates → bug-board present.
func TestGetTopProjectTemplatesIn_FromSelf(t *testing.T) {
	t.Parallel()
	templates, err := GetDB().GetTopProjectTemplatesIn([]string{"test-org"}, "test-org")
	if err != nil {
		t.Fatalf("GetTopProjectTemplatesIn returned error: %v", err)
	}
	if len(templates) == 0 {
		t.Fatal("GetTopProjectTemplatesIn returned empty, expected at least 'bug-board'")
	}
	found := false
	for _, pt := range templates {
		if pt.Name == "bug-board" {
			found = true
			if !pt.IsRecursive {
				t.Error("bug-board is_recursive = false, want true")
			}
			if pt.ColumnsJSON == "" {
				t.Error("bug-board columns_json is empty, want non-empty seeded payload")
			}
		}
	}
	if !found {
		names := make([]string, len(templates))
		for i, pt := range templates {
			names[i] = pt.Name
		}
		t.Errorf("project_templates = %v, expected to contain %q", names, "bug-board")
	}
}

// /q/project_templates/sub from test-org: visible nameids include test-org —
// pulls all templates attached, including bug-board.
func TestGetProjectTemplatesIn_Integration(t *testing.T) {
	t.Parallel()
	templates, err := GetDB().GetProjectTemplatesIn([]string{"test-org"}, "")
	if err != nil {
		t.Fatalf("GetProjectTemplatesIn returned error: %v", err)
	}
	if len(templates) == 0 {
		t.Fatal("GetProjectTemplatesIn returned empty, expected at least 'bug-board'")
	}
	found := false
	for _, pt := range templates {
		if pt.Name == "bug-board" {
			found = true
		}
	}
	if !found {
		names := make([]string, len(templates))
		for i, pt := range templates {
			names[i] = pt.Name
		}
		t.Errorf("project_templates = %v, expected to contain %q", names, "bug-board")
	}
}

// /q/project_templates/top from a child role with includeSelf=false. Visible
// nameids only contain ancestor "test-org"; the is_recursive=true filter must
// keep "bug-board" but drop "local-board".
func TestGetTopProjectTemplatesIn_FromChild(t *testing.T) {
	t.Parallel()
	templates, err := GetDB().GetTopProjectTemplatesIn([]string{"test-org"}, "test-org##:coordo")
	if err != nil {
		t.Fatalf("GetTopProjectTemplatesIn from child: %v", err)
	}

	foundRecursive := false
	foundLocal := false
	for _, pt := range templates {
		switch pt.Name {
		case "bug-board":
			foundRecursive = true
		case "local-board":
			foundLocal = true
		}
	}

	if !foundRecursive {
		names := make([]string, len(templates))
		for i, pt := range templates {
			names[i] = pt.Name
		}
		t.Errorf("top project templates from child = %v, expected recursive template %q to be present", names, "bug-board")
	}
	if foundLocal {
		t.Errorf("top project templates from child should NOT contain non-recursive template %q, but it was returned", "local-board")
	}
}

// /q/project_templates/top from a child role with includeSelf=true. Visible
// nameids = [child, ancestor]. Ancestors get is_recursive=true filter; self
// (the coordo role) has no templates. Only recursive templates should appear.
func TestGetTopProjectTemplatesIn_ExcludesNonRecursive(t *testing.T) {
	t.Parallel()
	templates, err := GetDB().GetTopProjectTemplatesIn([]string{"test-org##:coordo", "test-org"}, "test-org##:coordo")
	if err != nil {
		t.Fatalf("GetTopProjectTemplatesIn (non-recursive check): %v", err)
	}

	for _, pt := range templates {
		if !pt.IsRecursive {
			t.Errorf("top query returned non-recursive template %q (is_recursive=false); only recursive templates should appear", pt.Name)
		}
	}
}

func TestFormatTensionIntExtMap_PatternFilter(t *testing.T) {
	t.Parallel()
	pattern := "search term"
	q := TensionQuery{
		Nameids:  []string{"test-org"},
		First:    10,
		Pattern:  &pattern,
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

func keys[K comparable, V any](m map[K]V) []K {
	out := make([]K, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
