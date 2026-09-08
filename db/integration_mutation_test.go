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
	val, err := GetDB().GetByEq("Node.nameid", "test-org", "Node.about")
	if err != nil {
		t.Fatalf("GetByEq returned error: %v", err)
	}
	about, ok := val.(string)
	if !ok {
		t.Fatalf("GetByEq returned type %T, want string", val)
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
	val, err := GetDB().GetByEq("Node.nameid", nameid, "Node.role_type")
	if err != nil {
		t.Fatalf("GetByEq returned error: %v", err)
	}
	roleType, ok := val.(string)
	if !ok {
		t.Fatalf("GetByEq returned type %T, want string", val)
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
	val, err := GetDB().GetByEq("Node.nameid", "test-org", "Node.about")
	if err != nil {
		t.Fatalf("GetByEq returned error: %v", err)
	}
	about, ok := val.(string)
	if !ok {
		t.Fatalf("GetByEq returned type %T, want string", val)
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

	// Get the private-project's UID via GetByEqFiltered
	pData, err := GetDB().GetByEqFiltered(
		"Project.nameid", "private-project",
		"Project.parentnameid", "sec-org#private-circle",
		"uid Project.parentnameid Project.rootnameid",
	)
	if err != nil {
		t.Fatalf("GetByEqFiltered returned error: %v", err)
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
	val, err := GetDB().GetByEqFiltered(
		"Project.nameid", "private-project",
		"Project.parentnameid", "sec-org",
		"Project.parentnameid",
	)
	if err != nil {
		t.Fatalf("GetByEqFiltered returned error: %v", err)
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

func TestGovernedNodeMutations_Integration(t *testing.T) {
	const key = "integration-governed-node"
	cleanup := QueryMut{
		Q: `query {
			n as var(func: regexp(Node.nameid, /^` + key + `/))
			t as var(func: regexp(Tension.receiverid, /^` + key + `/)) { b as Tension.blobs }
		}`,
		M: []X{{D: `uid(b) * * .
			uid(t) * * .
			uid(n) * * .`}},
	}
	_, _ = GetDB().Gamma(cleanup, nil)
	t.Cleanup(func() { _, _ = GetDB().Gamma(cleanup, nil) })

	// One tension owning the blob and the node it governs.
	create := QueryMut{
		Q: `query { all(func: eq(Node.nameid, "` + key + `")) { uid } }`,
		M: []X{{S: `_:n <dgraph.type> "Node" .
		_:n <Node.nameid> "` + key + `" .
		_:n <Node.type_> "Role" .
		_:n <Node.isArchived> "false" .
		_:t <dgraph.type> "Tension" .
		_:t <Tension.receiverid> "` + key + `" .
		_:t <Tension.blobs> _:b .
		_:b <dgraph.type> "Blob" .
		_:b <Blob.tension> _:t .
		_:b <Blob.pushedFlag> "2026-01-01T00:00:00Z" .`}},
	}
	if _, err := GetDB().Gamma(create, nil); err != nil {
		t.Fatalf("creating governed mutation fixtures: %v", err)
	}

	uidByEq := func(predicate, value string) string {
		t.Helper()
		v, err := GetDB().GetByEq(predicate, value, "uid")
		if err != nil {
			t.Fatalf("resolving %s=%s: %v", predicate, value, err)
		}
		uid, ok := v.(string)
		if !ok {
			t.Fatalf("%s=%s UID has type %T", predicate, value, v)
		}
		return uid
	}
	blobUID := func(receiverid string) string {
		t.Helper()
		v, err := GetDB().GetByEq("Tension.receiverid", receiverid, "Tension.blobs", "uid")
		if err != nil {
			t.Fatalf("resolving blob for %s: %v", receiverid, err)
		}
		values, ok := v.([]any)
		if !ok || len(values) != 1 {
			t.Fatalf("blob for %s has value %T(%v)", receiverid, v, v)
		}
		uid, ok := values[0].(string)
		if !ok {
			t.Fatalf("blob UID for %s has type %T", receiverid, values[0])
		}
		return uid
	}
	nid := uidByEq("Node.nameid", key)
	tid := uidByEq("Tension.receiverid", key)
	bid := blobUID(key)

	if err := GetDB().LinkGovernedNode(tid, nid, bid); err != nil {
		t.Fatalf("LinkGovernedNode: %v", err)
	}
	if source, _ := GetDB().GetByUid(nid, "Node.source", "uid"); source != bid {
		t.Fatalf("link set source %v, want %s", source, bid)
	}
	if governed, _ := GetDB().GetByUid(tid, "Tension.governed_node", "uid"); governed != nid {
		t.Fatalf("link set governed node %v, want %s", governed, nid)
	}

	if err := GetDB().SetFieldById(nid, "Node.isArchived", "true"); err != nil {
		t.Fatalf("archive governed node: %v", err)
	}
	if state, _ := GetDB().GetByUid(nid, "Node.isArchived"); state != true {
		t.Fatalf("archive left Node.isArchived at %v", state)
	}

	if err := GetDB().SetFieldById(nid, "Node.isArchived", "false"); err != nil {
		t.Fatalf("unarchive governed node: %v", err)
	}
	if state, _ := GetDB().GetByUid(nid, "Node.isArchived"); state != false {
		t.Fatalf("unarchive left Node.isArchived at %v", state)
	}
}

func TestUpsertActivity_Integration(t *testing.T) {
	today := time.Now().UTC().Format("2006-01-02")
	todayISO := today + "T00:00:00Z"
	// Synthetic owner: "testuser" activity rows are shared with the graph and
	// web/handlers test binaries (which run concurrently and increment them via
	// trackActivity), so exact-count asserts and the delete-cleanup below are
	// only safe on a key nothing else writes.
	activityid := "u#upsert-test#" + today

	// Clean up any leftover from a previous run
	cleanup := QueryMut{
		Q: `query { v as var(func: eq(Activity.activityid, "` + activityid + `")) }`,
		M: []X{{D: `uid(v) * * .`}},
	}
	_, _ = GetDB().Gamma(cleanup, map[string]string{})

	// Helper to query the count for today's activity entry
	getCount := func() int {
		results, err := GetDB().Meta("getUserActivity", map[string]string{
			"username": "upsert-test",
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
			"ownerid":    "u#upsert-test",
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

// TestAddTensionComment_Integration guards the inbound-email comment insert:
// it must return the new uid (the anchor for attachments) and produce a node
// the app reads back as a Post/Comment.
func TestAddTensionComment_Integration(t *testing.T) {
	t.Parallel()

	// Throwaway tension: parallel tests assert on the shared one's comments.
	const title = "atc-tension"
	seed := QueryMut{
		Q: `query { r as var(func: eq(Node.nameid, "test-org")) }`,
		M: []X{{
			S: `_:t <dgraph.type> "Tension" .
                _:t <Tension.title> "` + title + `" .
                _:t <Tension.receiver> uid(r) .`,
		}},
	}
	if _, err := GetDB().Gamma(seed, map[string]string{}); err != nil {
		t.Fatalf("seed tension: %v", err)
	}
	t.Cleanup(func() {
		del := QueryMut{
			Q: `query {
                t as var(func: eq(Tension.title, "` + title + `")) { c as Tension.comments }
            }`,
			M: []X{{D: `uid(c) * * .
                       uid(t) * * .`}},
		}
		if _, err := GetDB().Gamma(del, map[string]string{}); err != nil {
			t.Errorf("cleanup: %v", err)
		}
	})
	tids, err := GetDB().GetIDs("Tension.title", title, nil, nil)
	if err != nil || len(tids) == 0 {
		t.Fatalf("GetIDs(%s) = %v, %v", title, tids, err)
	}
	tid := tids[0]

	cid, err := GetDB().AddTensionComment(tid, "testuser", `hello "quoted" reply`, "2026-05-01T00:00:00Z")
	if err != nil || cid == "" {
		t.Fatalf("AddTensionComment = %q, %v", cid, err)
	}

	// Belongs to the tension, carries author + message (the upload path's view).
	c, err := GetDB().GetCommentForUpload(tid, cid)
	if err != nil {
		t.Fatalf("GetCommentForUpload: %v", err)
	}
	if !c.Found || c.AuthorUsername != "testuser" || c.Message != `hello "quoted" reply` {
		t.Fatalf("comment = %+v", c)
	}

	// The GraphQL layer writes both interface and concrete type; queries on
	// Post would skip the node otherwise.
	types := QueryMut{Q: `query { all(func: uid(` + cid + `)) @filter(type(Post) AND type(Comment)) { uid } }`}
	res, err := GetDB().Gamma(types, map[string]string{})
	if err != nil || len(res) != 1 {
		t.Fatalf("type(Post) AND type(Comment) = %+v, %v", res, err)
	}
}

// TestAddContractComment_Integration: the contract-reply insert must return
// the new uid and hang the comment under Contract.comments with its author.
func TestAddContractComment_Integration(t *testing.T) {
	t.Parallel()

	const contractid = "acc-contract"
	seed := QueryMut{M: []X{{
		S: `_:c <dgraph.type> "Contract" .
            _:c <Contract.contractid> "` + contractid + `" .`,
	}}}
	res, err := GetDB().UpsertDql(seed, map[string]string{})
	if err != nil {
		t.Fatalf("seed contract: %v", err)
	}
	contractUid := res.Uids["c"]
	t.Cleanup(func() {
		del := QueryMut{
			Q: `query { c as var(func: uid(` + contractUid + `)) { cm as Contract.comments } }`,
			M: []X{{D: `uid(cm) * * .
                       uid(c) * * .`}},
		}
		if _, err := GetDB().Gamma(del, map[string]string{}); err != nil {
			t.Errorf("cleanup: %v", err)
		}
	})

	cid, err := GetDB().AddContractComment(contractUid, "testuser", "contract reply", "2026-05-01T00:00:00Z")
	if err != nil || cid == "" {
		t.Fatalf("AddContractComment = %q, %v", cid, err)
	}

	q := QueryMut{Q: `query {
        all(func: uid(` + contractUid + `)) @normalize {
            Contract.comments @filter(uid(` + cid + `)) {
                message: Post.message
                Post.createdBy { username: User.username }
            }
        }
    }`}
	rows, err := GetDB().Gamma(q, map[string]string{})
	if err != nil || len(rows) != 1 || rows[0]["message"] != "contract reply" || rows[0]["username"] != "testuser" {
		t.Fatalf("comment under contract = %+v, %v", rows, err)
	}
}
