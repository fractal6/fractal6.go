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

package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/internal/testutil"
	. "fractale/fractal6.go/web/handlers"
)

// projectNames extracts project names from a ProjectFull slice.
func projectNames(projects []db.ProjectFull) map[string]bool {
	names := make(map[string]bool, len(projects))
	for _, p := range projects {
		names[p.Name] = true
	}
	return names
}

// decodeProjects unmarshals a JSON response into a ProjectFull slice.
// Returns nil (not error) for "null" responses (empty results).
func decodeProjects(t *testing.T, body []byte) []db.ProjectFull {
	t.Helper()
	if string(body) == "null" {
		return nil
	}
	var projects []db.ProjectFull
	if err := json.Unmarshal(body, &projects); err != nil {
		t.Fatalf("failed to decode projects: %v (body: %s)", err, string(body))
	}
	return projects
}

// --- sec-org visibility tests ---
// sec-org is Private. testuser is Member, testuser2 is Owner + secret-circle Coordinator.
// Sub-circles: private-circle (Private), secret-circle (Secret).
// Projects: "Root Project" (on sec-org), "Private Project" (on private-circle),
//           "Secret Project" (on secret-circle).

func TestSubProjects_MemberSeesPrivateNotSecret(t *testing.T) {
	// testuser is a Member of sec-org.
	// They can see Private circles (org membership) but NOT Secret circles (no role there).
	jwtCookie := loginAs(testutil.TestUser, testutil.TestPassword)

	rr := doRequest("POST", "/q/projects/sub",
		NodeQuery{Nameid: testutil.SecOrg, IncludeSelf: true}, jwtCookie)
	requireStatus(t, rr, http.StatusOK)

	projects := decodeProjects(t, rr.Body.Bytes())
	names := projectNames(projects)

	// Should see root project (linked to Private root, testuser is member)
	if !names[testutil.SecOrgRootProject] {
		t.Errorf("expected to see %q, got projects: %v", testutil.SecOrgRootProject, names)
	}
	// Should see private project (linked to Private sub-circle, testuser is org member)
	if !names[testutil.SecOrgPrivateProject] {
		t.Errorf("expected to see %q, got projects: %v", testutil.SecOrgPrivateProject, names)
	}
	// Should NOT see secret project (linked to Secret sub-circle, testuser has no role there)
	if names[testutil.SecOrgSecretProject] {
		t.Errorf("expected NOT to see %q, but it was present", testutil.SecOrgSecretProject)
	}
}

func TestSubProjects_OwnerWithSecretRoleSeesAll(t *testing.T) {
	// testuser2 is Owner of sec-org AND Coordinator of secret-circle.
	// They can see all projects including the secret one.
	jwtCookie := loginAs(testutil.TestUser2, testutil.TestPassword2)

	rr := doRequest("POST", "/q/projects/sub",
		NodeQuery{Nameid: testutil.SecOrg, IncludeSelf: true}, jwtCookie)
	requireStatus(t, rr, http.StatusOK)

	projects := decodeProjects(t, rr.Body.Bytes())
	names := projectNames(projects)

	if !names[testutil.SecOrgRootProject] {
		t.Errorf("expected to see %q, got projects: %v", testutil.SecOrgRootProject, names)
	}
	if !names[testutil.SecOrgPrivateProject] {
		t.Errorf("expected to see %q, got projects: %v", testutil.SecOrgPrivateProject, names)
	}
	if !names[testutil.SecOrgSecretProject] {
		t.Errorf("expected to see %q, got projects: %v", testutil.SecOrgSecretProject, names)
	}
}

func TestSubProjects_ExcludeSelf(t *testing.T) {
	// With IncludeSelf=false, the root circle's project should be excluded.
	jwtCookie := loginAs(testutil.TestUser, testutil.TestPassword)

	rr := doRequest("POST", "/q/projects/sub",
		NodeQuery{Nameid: testutil.SecOrg, IncludeSelf: false}, jwtCookie)
	requireStatus(t, rr, http.StatusOK)

	projects := decodeProjects(t, rr.Body.Bytes())
	names := projectNames(projects)

	// Root project is linked only to sec-org, should be excluded
	if names[testutil.SecOrgRootProject] {
		t.Errorf("expected NOT to see %q with IncludeSelf=false, but it was present", testutil.SecOrgRootProject)
	}
	// Private project is on a sub-circle, should still be visible
	if !names[testutil.SecOrgPrivateProject] {
		t.Errorf("expected to see %q, got projects: %v", testutil.SecOrgPrivateProject, names)
	}
	// Secret project still not visible (no role)
	if names[testutil.SecOrgSecretProject] {
		t.Errorf("expected NOT to see %q, but it was present", testutil.SecOrgSecretProject)
	}
}

func TestSubProjects_UnauthenticatedSeesNothingOnPrivateOrg(t *testing.T) {
	// No JWT — sec-org is Private, so unauthenticated users see no projects.
	rr := doRequest("POST", "/q/projects/sub",
		NodeQuery{Nameid: testutil.SecOrg, IncludeSelf: true})
	requireStatus(t, rr, http.StatusOK)

	projects := decodeProjects(t, rr.Body.Bytes())
	if len(projects) > 0 {
		t.Errorf("expected no projects for unauthenticated user on private org, got %d: %v",
			len(projects), projectNames(projects))
	}
}

// --- test-org basic tests ---
// test-org is Public with no projects seeded — just verify the endpoint works.

func TestSubProjects_PublicOrgVisibleToAnyone(t *testing.T) {
	// test-org is Public; its public-project must be visible regardless of auth.
	rr := doRequest("POST", "/q/projects/sub",
		NodeQuery{Nameid: "test-org", IncludeSelf: true})
	requireStatus(t, rr, http.StatusOK)

	projects := decodeProjects(t, rr.Body.Bytes())
	names := make(map[string]bool, len(projects))
	for _, p := range projects {
		names[p.Name] = true
	}
	if !names["Public Project"] {
		t.Errorf("expected to see %q in test-org projects, got: %v", "Public Project", names)
	}
}

func TestSubProjects_InvalidBody(t *testing.T) {
	// Send invalid JSON body (number instead of struct)
	rr := doRequest("POST", "/q/projects/sub", 12345)
	if rr.Code == http.StatusOK {
		t.Fatal("expected non-200 status for invalid body")
	}
}

func TestTensionListsExposeDerivedGovernanceState(t *testing.T) {
	const key = "integration-rest-governance-state"
	cleanup := db.QueryMut{
		Q: `query {
			root as var(func: eq(Node.nameid, "test-org"))
			emitter as var(func: eq(Node.nameid, "` + key + `-emitter"))
			t as var(func: eq(Tension.receiverid, "` + key + `")) {
				b as Tension.blobs { f as Blob.node }
				n as Tension.governed_node
			}
		}`,
		M: []db.X{{D: `uid(root) <Node.tensions_in> uid(t) .
			uid(emitter) <Node.tensions_out> uid(t) .
			uid(f) * * .
			uid(b) * * .
			uid(t) * * .
			uid(n) * * .
			uid(emitter) * * .`}},
	}
	_, _ = db.GetDB().Gamma(cleanup, nil)
	t.Cleanup(func() { _, _ = db.GetDB().Gamma(cleanup, nil) })

	create := db.QueryMut{
		Q: `query {
			root as var(func: eq(Node.nameid, "test-org"))
			author as var(func: eq(User.username, "testuser"))
		}`,
		M: []db.X{{S: `_:emitter <dgraph.type> "Node" .
			_:emitter <Node.nameid> "` + key + `-emitter" .

			_:draft <dgraph.type> "Tension" .
			_:draft <Tension.title> "REST draft role" .
			_:draft <Tension.status> "Open" .
			_:draft <Tension.type_> "Operational" .
			_:draft <Tension.receiver> uid(root) .
			_:draft <Tension.receiverid> "` + key + `" .
			_:draft <Tension.emitter> _:emitter .
			_:draft <Post.createdBy> uid(author) .
			_:draft <Post.createdAt> "2099-01-03T00:00:00Z" .
			_:draft <Tension.blobs> _:draftBlob .
			_:draftBlob <dgraph.type> "Blob" .
			_:draftBlob <Post.createdAt> "2099-01-03T00:00:00Z" .
			_:draftBlob <Blob.node> _:draftFragment .
			_:draftFragment <dgraph.type> "NodeFragment" .
			_:draftFragment <NodeFragment.type_> "Role" .

			_:active <dgraph.type> "Tension" .
			_:active <Tension.title> "REST active circle" .
			_:active <Tension.status> "Open" .
			_:active <Tension.type_> "Operational" .
			_:active <Tension.receiver> uid(root) .
			_:active <Tension.receiverid> "` + key + `" .
			_:active <Tension.emitter> _:emitter .
			_:active <Post.createdBy> uid(author) .
			_:active <Post.createdAt> "2099-01-02T00:00:00Z" .
			_:active <Tension.governed_node> _:activeNode .
			_:activeNode <dgraph.type> "Node" .
			_:activeNode <Node.nameid> "` + key + `-active" .
			_:activeNode <Node.type_> "Circle" .
			_:activeNode <Node.isArchived> "false" .

			_:archived <dgraph.type> "Tension" .
			_:archived <Tension.title> "REST archived role" .
			_:archived <Tension.status> "Open" .
			_:archived <Tension.type_> "Operational" .
			_:archived <Tension.receiver> uid(root) .
			_:archived <Tension.receiverid> "` + key + `" .
			_:archived <Tension.emitter> _:emitter .
			_:archived <Post.createdBy> uid(author) .
			_:archived <Post.createdAt> "2099-01-01T00:00:00Z" .
			_:archived <Tension.governed_node> _:archivedNode .
			_:archivedNode <dgraph.type> "Node" .
			_:archivedNode <Node.nameid> "` + key + `-archived" .
			_:archivedNode <Node.type_> "Role" .
			_:archivedNode <Node.isArchived> "true" .

			uid(root) <Node.tensions_in> _:draft .
			uid(root) <Node.tensions_in> _:active .
			uid(root) <Node.tensions_in> _:archived .
			_:emitter <Node.tensions_out> _:draft .
			_:emitter <Node.tensions_out> _:active .
			_:emitter <Node.tensions_out> _:archived .`}},
	}
	if _, err := db.GetDB().Gamma(create, nil); err != nil {
		t.Fatalf("creating REST governance fixtures: %v", err)
	}

	query := db.TensionQuery{Nameids: []string{"test-org"}, First: 10}
	for _, mode := range []string{"int", "ext", "all"} {
		rr := doRequest("POST", "/q/tensions/"+mode, query)
		requireStatus(t, rr, http.StatusOK)
		if bytes.Contains(rr.Body.Bytes(), []byte(`"action"`)) {
			t.Errorf("/%s response still exposes action: %s", mode, rr.Body.String())
		}

		var tensions []model.TensionRef
		if err := json.Unmarshal(rr.Body.Bytes(), &tensions); err != nil {
			t.Fatalf("decoding /%s response: %v", mode, err)
		}
		byTitle := make(map[string]model.TensionRef, len(tensions))
		for _, tension := range tensions {
			if tension.Title != nil {
				byTitle[*tension.Title] = tension
			}
		}

		draft := byTitle["REST draft role"]
		if draft.GovernedNode != nil || len(draft.Blobs) != 1 || draft.Blobs[0].Node == nil ||
			draft.Blobs[0].Node.Type == nil || *draft.Blobs[0].Node.Type != model.NodeTypeRole {
			t.Errorf("/%s draft derivation inputs missing: %+v", mode, draft)
		}
		active := byTitle["REST active circle"].GovernedNode
		if active == nil || active.ID == nil || active.Nameid == nil || active.Type == nil || active.IsArchived == nil ||
			*active.Type != model.NodeTypeCircle || *active.IsArchived {
			t.Errorf("/%s active derivation inputs missing: %+v", mode, active)
		}
		archived := byTitle["REST archived role"].GovernedNode
		if archived == nil || archived.ID == nil || archived.Nameid == nil || archived.Type == nil || archived.IsArchived == nil ||
			*archived.Type != model.NodeTypeRole || !*archived.IsArchived {
			t.Errorf("/%s archived derivation inputs missing: %+v", mode, archived)
		}
	}
}
