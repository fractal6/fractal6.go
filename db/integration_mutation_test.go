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
	"testing"
	"time"

	. "fractale/fractal6.go/db"
	"fractale/fractal6.go/graph/model"
)

func TestSetFieldByEq_Integration(t *testing.T) {
	// Set Node.about on our test org
	newAbout := "Updated about text"
	err := GetDB().SetFieldByEq("Node.nameid", "test-org", "Node.about", newAbout)
	if err != nil {
		t.Fatalf("SetFieldByEq returned error: %v", err)
	}

	// Read it back
	val, err := GetDB().GetFieldByEq("Node.nameid", "test-org", "Node.about")
	if err != nil {
		t.Fatalf("GetFieldByEq returned error: %v", err)
	}
	about, ok := val.(string)
	if !ok {
		t.Fatalf("GetFieldByEq returned type %T, want string", val)
	}
	if about != newAbout {
		t.Errorf("Node.about = %q, want %q", about, newAbout)
	}

	// Restore original value
	_ = GetDB().SetFieldByEq("Node.nameid", "test-org", "Node.about", "A test organisation")
}

func TestMeta_MarkAllAsRead_Integration(t *testing.T) {
	// markAllAsRead is a mutation that marks UserEvents as read.
	// With no unread events, this is a no-op mutation — should succeed without error.
	_, err := GetDB().Meta("markAllAsRead", map[string]string{
		"username": "testuser",
	})
	if err != nil {
		t.Fatalf("Meta(markAllAsRead) returned error: %v", err)
	}
}

func TestUpgradeMember_Integration(t *testing.T) {
	nameid := "test-org##@testuser"

	// Change role_type to Guest
	err := GetDB().UpgradeMember(nameid, model.RoleTypeGuest)
	if err != nil {
		t.Fatalf("UpgradeMember to Guest returned error: %v", err)
	}

	// Verify the change
	val, err := GetDB().GetFieldByEq("Node.nameid", nameid, "Node.role_type")
	if err != nil {
		t.Fatalf("GetFieldByEq returned error: %v", err)
	}
	roleType, ok := val.(string)
	if !ok {
		t.Fatalf("GetFieldByEq returned type %T, want string", val)
	}
	if roleType != "Guest" {
		t.Errorf("Node.role_type = %q, want %q", roleType, "Guest")
	}

	// Restore to Owner (original seed value)
	err = GetDB().UpgradeMember(nameid, model.RoleTypeOwner)
	if err != nil {
		t.Fatalf("UpgradeMember to Owner (restore) returned error: %v", err)
	}
}

func TestGamma_Integration(t *testing.T) {
	// Use Gamma with an inline QueryMut to set and verify a field.
	// Set Node.about to a new value via custom upsert.
	newAbout := "gamma-test-value"
	qm := QueryMut{
		Q: `query {
			node as var(func: eq(Node.nameid, "test-org"))
		}`,
		M: []X{{
			S: `uid(node) <Node.about> "` + newAbout + `" .`,
		}},
	}

	results, err := GetDB().Gamma(qm, map[string]string{})
	if err != nil {
		t.Fatalf("Gamma returned error: %v", err)
	}
	t.Logf("Gamma returned %d results", len(results))

	// Verify the change
	val, err := GetDB().GetFieldByEq("Node.nameid", "test-org", "Node.about")
	if err != nil {
		t.Fatalf("GetFieldByEq returned error: %v", err)
	}
	about, ok := val.(string)
	if !ok {
		t.Fatalf("GetFieldByEq returned type %T, want string", val)
	}
	if about != newAbout {
		t.Errorf("Node.about = %q, want %q", about, newAbout)
	}

	// Restore original value
	_ = GetDB().SetFieldByEq("Node.nameid", "test-org", "Node.about", "A test organisation")
}

func TestProjectReparent_Integration(t *testing.T) {
	// Simulate the reparenting logic: when a node matching parentnameid
	// is removed from Project.nodes, parentnameid should be updated to a remaining node.

	// Get the private-project's UID via GetFieldByEq with filter
	pData, err := GetDB().GetFieldByEq(
		"Project.nameid", "private-project",
		"uid Project.parentnameid Project.rootnameid",
		"Project.parentnameid", "sec-org#private-circle",
	)
	if err != nil {
		t.Fatalf("GetFieldByEq returned error: %v", err)
	}
	if pData == nil {
		t.Fatal("private-project not found")
	}
	m := pData.(map[string]any)
	projectUID := m["id"].(string)

	// Add the root node (sec-org) as a second node reference on the project
	addNode := QueryMut{
		Q: `query {
			proj as var(func: uid(` + projectUID + `))
			root as var(func: eq(Node.nameid, "sec-org"))
		}`,
		M: []X{{
			S: `uid(proj) <Project.nodes> uid(root) .
				uid(root) <Node.projects> uid(proj) .`,
		}},
	}
	_, err = GetDB().Gamma(addNode, map[string]string{})
	if err != nil {
		t.Fatalf("failed to add second node: %v", err)
	}

	// Now simulate reparenting: set parentnameid to the root node
	err = GetDB().SetFieldById(projectUID, "Project.parentnameid", "sec-org")
	if err != nil {
		t.Fatalf("SetFieldById(parentnameid) returned error: %v", err)
	}

	// Verify the change
	val, err := GetDB().GetFieldByEq(
		"Project.nameid", "private-project",
		"Project.parentnameid",
		"Project.parentnameid", "sec-org",
	)
	if err != nil {
		t.Fatalf("GetFieldByEq returned error: %v", err)
	}
	parentnameid, ok := val.(string)
	if !ok {
		t.Fatalf("expected string, got %T", val)
	}
	if parentnameid != "sec-org" {
		t.Errorf("parentnameid = %q, want %q", parentnameid, "sec-org")
	}

	// Restore original values
	_ = GetDB().SetFieldById(projectUID, "Project.parentnameid", "sec-org#private-circle")
	// Remove the extra node reference
	rmNode := QueryMut{
		Q: `query {
			proj as var(func: uid(` + projectUID + `))
			root as var(func: eq(Node.nameid, "sec-org"))
		}`,
		M: []X{{
			D: `uid(proj) <Project.nodes> uid(root) .
				uid(root) <Node.projects> uid(proj) .`,
		}},
	}
	_, _ = GetDB().Gamma(rmNode, map[string]string{})
}

func TestUpsertActivity_Integration(t *testing.T) {
	today := time.Now().UTC().Format("2006-01-02")
	todayISO := today + "T00:00:00Z"
	activityid := "u#testuser#" + today

	// Clean up any leftover from a previous run
	cleanup := QueryMut{
		Q: `query { v as var(func: eq(Activity.activityid, "` + activityid + `")) }`,
		M: []X{{D: `uid(v) * * .`}},
	}
	_, _ = GetDB().Gamma(cleanup, map[string]string{})

	// Helper to query the count for today's activity entry
	getCount := func() int {
		results, err := GetDB().Meta("getUserActivity", map[string]string{
			"username": "testuser",
		})
		if err != nil {
			t.Fatalf("getUserActivity returned error: %v", err)
		}
		for _, r := range results {
			if aid, ok := r["activityid"].(string); ok && aid == activityid {
				switch c := r["count"].(type) {
				case float64:
					return int(c)
				case int:
					return c
				default:
					t.Fatalf("count type = %T, want numeric", r["count"])
				}
			}
		}
		return 0
	}

	// Upsert 3 times: first creates (count=1), subsequent increment.
	for i := 1; i <= 3; i++ {
		_, err := GetDB().Meta("upsertActivity", map[string]string{
			"activityid": activityid,
			"ownerid":    "u#testuser",
			"date":       todayISO,
		})
		if err != nil {
			t.Fatalf("upsertActivity (iteration %d) returned error: %v", i, err)
		}

		if count := getCount(); count != i {
			t.Errorf("after upsert %d: count = %d, want %d", i, count, i)
		}
	}

	// Clean up
	_, err := GetDB().Gamma(cleanup, map[string]string{})
	if err != nil {
		t.Logf("cleanup warning: %v", err)
	}
}
