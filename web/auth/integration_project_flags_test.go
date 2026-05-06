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

package auth_test

import (
	"context"
	"testing"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/web/auth"
	"fractale/fractal6.go/web/sessions"
)

// getProjectUID returns the Dgraph UID for the root-project in sec-org.
func getProjectUID(t *testing.T) string {
	t.Helper()
	data, err := db.GetDB().GetByEq("Project.nameid", "root-project", "uid")
	if err != nil {
		t.Fatalf("failed to look up root-project: %v", err)
	}
	uid, ok := data.(string)
	if !ok || uid == "" {
		t.Fatal("root-project UID not found")
	}
	return uid
}

// setProjectFlags sets both peerCanEditProject and guestCanEditProject on a project.
func setProjectFlags(t *testing.T, projectUID string, peer, guest bool) {
	t.Helper()
	peerStr := "false"
	if peer {
		peerStr = "true"
	}
	guestStr := "false"
	if guest {
		guestStr = "true"
	}
	if err := db.GetDB().SetFieldById(projectUID, "Project.peerCanEditProject", peerStr); err != nil {
		t.Fatalf("SetFieldById(peerCanEditProject) error: %v", err)
	}
	if err := db.GetDB().SetFieldById(projectUID, "Project.guestCanEditProject", guestStr); err != nil {
		t.Fatalf("SetFieldById(guestCanEditProject) error: %v", err)
	}
}

// clearProjectFlags removes both project permission flags (restores default nil/false).
func clearProjectFlags(t *testing.T, projectUID string) {
	t.Helper()
	// Setting to "false" restores the default deny behaviour.
	setProjectFlags(t, projectUID, false, false)
}

// addCollaborator adds a user as project collaborator via DQL.
func addCollaborator(t *testing.T, projectUID, username string) {
	t.Helper()
	qm := db.QueryMut{
		Q: `query {
			proj as var(func: uid(` + projectUID + `))
			user as var(func: eq(User.username, "` + username + `"))
		}`,
		M: []db.X{{S: `uid(proj) <Project.collaborators> uid(user) .`}},
	}
	if _, err := db.GetDB().Gamma(qm, map[string]string{}); err != nil {
		t.Fatalf("addCollaborator error: %v", err)
	}
}

// removeCollaborator removes a user from project collaborators via DQL.
func removeCollaborator(t *testing.T, projectUID, username string) {
	t.Helper()
	qm := db.QueryMut{
		Q: `query {
			proj as var(func: uid(` + projectUID + `))
			user as var(func: eq(User.username, "` + username + `"))
		}`,
		M: []db.X{{D: `uid(proj) <Project.collaborators> uid(user) .`}},
	}
	if _, err := db.GetDB().Gamma(qm, map[string]string{}); err != nil {
		t.Fatalf("removeCollaborator error: %v", err)
	}
}

// setRoleType changes the role_type of a member node in Dgraph and flushes
// the Redis role cache for the affected user to prevent stale data in subsequent tests.
func setRoleType(t *testing.T, nameid string, rt string) {
	t.Helper()
	if err := db.GetDB().SetFieldByEq("Node.nameid", nameid, "Node.role_type", rt); err != nil {
		t.Fatalf("SetFieldByEq(role_type) error for %s: %v", nameid, err)
	}
	// Extract username from nameid (e.g. "sec-org##@testuser" → "testuser")
	// and flush its Redis role cache to avoid poisoning subsequent test packages.
	if i := len(nameid) - 1; i > 0 {
		for j := i; j >= 0; j-- {
			if nameid[j] == '@' {
				username := nameid[j+1:]
				sessions.GetCache().Del(context.Background(), username+"roles")
				break
			}
		}
	}
}

// TestCheckProjectAuth_PeerFlagGrantsMember verifies that a Member user is
// granted access when peerCanEditProject is true.
func TestCheckProjectAuth_PeerFlagGrantsMember(t *testing.T) {
	uid := getProjectUID(t)
	defer clearProjectFlags(t, uid)

	// testuser is Member in sec-org (not a coordinator, not a collaborator).
	uctx := &model.UserCtx{Username: "testuser", NoCache: true}

	// Without flag: should be denied (falls through to coordinator check).
	ok, err := auth.CheckProjectAuth(uctx, uid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected denial without peerCanEditProject flag")
	}

	// With peerCanEditProject=true: should be granted.
	setProjectFlags(t, uid, true, false)
	ok, err = auth.CheckProjectAuth(uctx, uid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected access granted with peerCanEditProject=true for Member")
	}
}

// TestCheckProjectAuth_PeerFlagDeniesGuest verifies that a Guest user is
// denied when only peerCanEditProject is true (guestCanEditProject is false).
func TestCheckProjectAuth_PeerFlagDeniesGuest(t *testing.T) {
	uid := getProjectUID(t)
	defer clearProjectFlags(t, uid)

	// Temporarily change testuser to Guest role.
	setRoleType(t, "sec-org##@testuser", "Guest")
	defer setRoleType(t, "sec-org##@testuser", "Member")

	uctx := &model.UserCtx{Username: "testuser", NoCache: true}
	setProjectFlags(t, uid, true, false)

	ok, err := auth.CheckProjectAuth(uctx, uid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected denial for Guest when only peerCanEditProject=true")
	}
}

// TestCheckProjectAuth_BothFlagsGrantGuest verifies that a Guest user is
// granted access when both peerCanEditProject and guestCanEditProject are true.
func TestCheckProjectAuth_BothFlagsGrantGuest(t *testing.T) {
	uid := getProjectUID(t)
	defer clearProjectFlags(t, uid)

	// Temporarily change testuser to Guest role.
	setRoleType(t, "sec-org##@testuser", "Guest")
	defer setRoleType(t, "sec-org##@testuser", "Member")

	uctx := &model.UserCtx{Username: "testuser", NoCache: true}
	setProjectFlags(t, uid, true, true)

	ok, err := auth.CheckProjectAuth(uctx, uid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected access granted for Guest when both flags are true")
	}
}

// TestCheckProjectAuth_CoordinatorAlwaysPasses verifies that a coordinator
// passes regardless of flags.
func TestCheckProjectAuth_CoordinatorAlwaysPasses(t *testing.T) {
	uid := getProjectUID(t)
	defer clearProjectFlags(t, uid)

	// testuser2 is Owner of sec-org (coordinator-level auth on linked node).
	uctx := &model.UserCtx{Username: "testuser2", NoCache: true}

	// Flags off — coordinator should still pass.
	ok, err := auth.CheckProjectAuth(uctx, uid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected coordinator to always have access regardless of flags")
	}
}

// TestCheckProjectAuth_CollaboratorBypassesFlags verifies that a collaborator
// passes regardless of flags being off.
func TestCheckProjectAuth_CollaboratorBypassesFlags(t *testing.T) {
	uid := getProjectUID(t)
	defer clearProjectFlags(t, uid)

	// Add testuser as collaborator on the project.
	addCollaborator(t, uid, "testuser")
	defer removeCollaborator(t, uid, "testuser")

	uctx := &model.UserCtx{Username: "testuser", NoCache: true}

	// Flags off — collaborator should still pass.
	ok, err := auth.CheckProjectAuth(uctx, uid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected collaborator to always have access regardless of flags")
	}
}

// TestCheckProjectAuth_NonMemberDenied verifies that a user with no role in the
// org is denied even when both flags are true.
func TestCheckProjectAuth_NonMemberDenied(t *testing.T) {
	uid := getProjectUID(t)
	defer clearProjectFlags(t, uid)

	// Temporarily remove testuser's membership in sec-org by setting role to Retired.
	setRoleType(t, "sec-org##@testuser", "Retired")
	defer setRoleType(t, "sec-org##@testuser", "Member")

	uctx := &model.UserCtx{Username: "testuser", NoCache: true}
	setProjectFlags(t, uid, true, true)

	ok, err := auth.CheckProjectAuth(uctx, uid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected denial for Retired user even with both flags true")
	}
}
