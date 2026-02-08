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

package db

import (
	"testing"

	"fractale/fractal6.go/graph/model"
)

func TestSetFieldByEq_Integration(t *testing.T) {
	// Set Node.about on our test org
	newAbout := "Updated about text"
	err := DB.SetFieldByEq("Node.nameid", "test-org#", "Node.about", newAbout)
	if err != nil {
		t.Fatalf("SetFieldByEq returned error: %v", err)
	}

	// Read it back
	val, err := DB.GetFieldByEq("Node.nameid", "test-org#", "Node.about")
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
	_ = DB.SetFieldByEq("Node.nameid", "test-org#", "Node.about", "A test organisation")
}

func TestMeta_MarkAllAsRead_Integration(t *testing.T) {
	// markAllAsRead is a mutation that marks UserEvents as read.
	// With no unread events, this is a no-op mutation — should succeed without error.
	_, err := DB.Meta("markAllAsRead", map[string]string{
		"username": "testuser",
	})
	if err != nil {
		t.Fatalf("Meta(markAllAsRead) returned error: %v", err)
	}
}

func TestUpgradeMember_Integration(t *testing.T) {
	nameid := "test-org#@testuser"

	// Change role_type to Guest
	err := DB.UpgradeMember(nameid, model.RoleTypeGuest)
	if err != nil {
		t.Fatalf("UpgradeMember to Guest returned error: %v", err)
	}

	// Verify the change
	val, err := DB.GetFieldByEq("Node.nameid", nameid, "Node.role_type")
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

	// Restore to Member
	err = DB.UpgradeMember(nameid, model.RoleTypeMember)
	if err != nil {
		t.Fatalf("UpgradeMember to Member (restore) returned error: %v", err)
	}
}

func TestGamma_Integration(t *testing.T) {
	// Use Gamma with an inline QueryMut to set and verify a field.
	// Set Node.about to a new value via custom upsert.
	newAbout := "gamma-test-value"
	qm := QueryMut{
		Q: `query {
			node as var(func: eq(Node.nameid, "test-org#"))
		}`,
		M: []X{{
			S: `uid(node) <Node.about> "` + newAbout + `" .`,
		}},
	}

	results, err := DB.Gamma(qm, map[string]string{})
	if err != nil {
		t.Fatalf("Gamma returned error: %v", err)
	}
	t.Logf("Gamma returned %d results", len(results))

	// Verify the change
	val, err := DB.GetFieldByEq("Node.nameid", "test-org#", "Node.about")
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
	_ = DB.SetFieldByEq("Node.nameid", "test-org#", "Node.about", "A test organisation")
}
