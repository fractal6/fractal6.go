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

// Cascade-delete tests for DeleteTensionDeep, DeleteContractDeep, DeleteUser.
// Each test builds a disposable fixture via UpsertDql (capturing every uid
// from the response Uids map), runs the delete, then probes the surviving
// graph through Gamma to confirm cascade + reverse-edge cleanup.

package db_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	. "fractale/fractal6.go/db"
)

// uidExists reports whether a uid still has any predicate (post-delete the
// node has no predicates, so a `has(dgraph.type)` filter returns empty).
func uidExists(t *testing.T, uid string) bool {
	t.Helper()
	q := QueryMut{Q: fmt.Sprintf(`{ all(func: uid(%s)) @filter(has(dgraph.type)) { uid } }`, uid)}
	res, err := GetDB().Gamma(q, map[string]string{})
	if err != nil {
		t.Fatalf("uidExists(%s): %v", uid, err)
	}
	return len(res) > 0
}

// edgeContains reports whether `parentUID` has an edge `<predicate>` to `targetUID`.
func edgeContains(t *testing.T, parentUID, predicate, targetUID string) bool {
	t.Helper()
	q := QueryMut{Q: fmt.Sprintf(`{
        all(func: uid(%s)) {
            <%s> @filter(uid(%s)) { uid }
        }
    }`, parentUID, predicate, targetUID)}
	res, err := GetDB().Gamma(q, map[string]string{})
	if err != nil {
		t.Fatalf("edgeContains(%s, %s, %s): %v", parentUID, predicate, targetUID, err)
	}
	if len(res) == 0 {
		return false
	}
	// CleanDqlMap strips the "Type." prefix; the survivor key is the predicate's
	// last segment (e.g. "Tension.contracts" → "contracts").
	short := predicate
	if i := strings.LastIndex(predicate, "."); i >= 0 {
		short = predicate[i+1:]
	}
	v, ok := res[0][short]
	if !ok {
		return false
	}
	switch x := v.(type) {
	case []any:
		return len(x) > 0
	case map[string]any:
		return true
	}
	return false
}

// resolveUID returns the uid of the node matching predicate=value.
func resolveUID(t *testing.T, predicate, value string) string {
	t.Helper()
	val, err := GetDB().GetByEq(predicate, value, "uid")
	if err != nil {
		t.Fatalf("resolveUID(%s=%s): %v", predicate, value, err)
	}
	if val == nil {
		t.Fatalf("resolveUID(%s=%s): not found", predicate, value)
	}
	return val.(string)
}

func TestDeleteTensionDeep_Integration(t *testing.T) {
	orgUID := resolveUID(t, "Node.nameid", "test-org")
	roleUID := resolveUID(t, "Node.nameid", "test-org##@testuser")
	userUID := resolveUID(t, "User.username", "testuser")

	// Unique markers so concurrent runs don't collide.
	tag := fmt.Sprintf("dt-%d", time.Now().UnixNano())
	now := time.Now().UTC().Format(time.RFC3339)

	// Build the fixture in a single upsert. Reverse edges from emitter/receiver
	// (Node.tensions_out / Node.tensions_in) are explicitly seeded so we can
	// verify the cascade delete drops them.
	setup := QueryMut{
		Q: fmt.Sprintf(`query {
            org as var(func: uid(%s))
            role as var(func: uid(%s))
            mt as var(func: uid(%s))
        }`, orgUID, roleUID, userUID),
		M: []X{{
			S: fmt.Sprintf(`
                _:t <dgraph.type> "Tension" .
                _:t <Tension.title> "%s-tension" .
                _:t <Tension.type_> "Operational" .
                _:t <Tension.status> "Open" .
                _:t <Tension.emitter> uid(org) .
                _:t <Tension.emitterid> "test-org" .
                _:t <Tension.receiver> uid(org) .
                _:t <Tension.receiverid> "test-org" .
                _:t <Post.createdAt> "%s" .
                _:t <Post.createdBy> uid(mt) .
                uid(org) <Node.tensions_out> _:t .
                uid(org) <Node.tensions_in> _:t .

                _:c <dgraph.type> "Comment" .
                _:c <Post.message> "%s-comment" .
                _:c <Post.createdAt> "%s" .
                _:c <Post.createdBy> uid(mt) .
                _:t <Tension.comments> _:c .

                _:r <dgraph.type> "Reaction" .
                _:r <Reaction.reactionid> "%s#tcomment#1" .
                _:r <Reaction.type_> "1" .
                _:r <Reaction.user> uid(mt) .
                _:r <Reaction.comment> _:c .
                _:c <Comment.reactions> _:r .

                _:f <dgraph.type> "File" .
                _:f <File.storageKey> "%s/tcomment-file" .
                _:f <File.filename> "a.png" .
                _:f <File.contentType> "image/png" .
                _:f <File.size> "1" .
                _:f <File.createdAt> "%s" .
                _:f <File.createdBy> uid(mt) .
                _:f <File.comment> _:c .
                _:f <File.tension> _:t .
                _:c <Comment.files> _:f .

                _:b <dgraph.type> "Blob" .
                _:b <Post.createdAt> "%s" .
                _:b <Post.createdBy> uid(mt) .
                _:b <Blob.tension> _:t .
                _:t <Tension.blobs> _:b .

                _:bn <dgraph.type> "NodeFragment" .
                _:bn <NodeFragment.name> "%s-frag" .
                _:b <Blob.node> _:bn .

                _:m <dgraph.type> "Mandate" .
                _:m <Mandate.purpose> "%s-mandate" .
                _:bn <NodeFragment.mandate> _:m .

                _:ev <dgraph.type> "Event" .
                _:ev <Event.event_type> "Created" .
                _:ev <Post.createdAt> "%s" .
                _:ev <Post.createdBy> uid(mt) .
                _:ev <Event.tension> _:t .
                _:t <Tension.history> _:ev .

                _:mev <dgraph.type> "Event" .
                _:mev <Event.event_type> "Mentioned" .
                _:mev <Post.createdAt> "%s" .
                _:mev <Post.createdBy> uid(mt) .
                _:mev <Event.mentioned> _:t .
                _:t <Tension.mentions> _:mev .

                _:ct <dgraph.type> "Contract" .
                _:ct <Contract.contractid> "%s#contract" .
                _:ct <Contract.status> "Open" .
                _:ct <Contract.contract_type> "AnyCoordoDual" .
                _:ct <Post.createdAt> "%s" .
                _:ct <Post.createdBy> uid(mt) .
                _:ct <Contract.tension> _:t .
                _:t <Tension.contracts> _:ct .

                _:ce <dgraph.type> "EventFragment" .
                _:ce <EventFragment.event_type> "Created" .
                _:ct <Contract.event> _:ce .

                _:vt <dgraph.type> "Vote" .
                _:vt <Vote.voteid> "%s#vote" .
                _:vt <Vote.contract> _:ct .
                _:vt <Vote.node> uid(role) .
                _:vt <Post.createdAt> "%s" .
                _:vt <Post.createdBy> uid(mt) .
                _:ct <Contract.participants> _:vt .

                _:cc <dgraph.type> "Comment" .
                _:cc <Post.message> "%s-ccomment" .
                _:cc <Post.createdAt> "%s" .
                _:cc <Post.createdBy> uid(mt) .
                _:ct <Contract.comments> _:cc .

                _:cr <dgraph.type> "Reaction" .
                _:cr <Reaction.reactionid> "%s#ccomment#1" .
                _:cr <Reaction.type_> "1" .
                _:cr <Reaction.user> uid(mt) .
                _:cr <Reaction.comment> _:cc .
                _:cc <Comment.reactions> _:cr .

                _:cf <dgraph.type> "File" .
                _:cf <File.storageKey> "%s/ccomment-file" .
                _:cf <File.filename> "b.png" .
                _:cf <File.contentType> "image/png" .
                _:cf <File.size> "1" .
                _:cf <File.createdAt> "%s" .
                _:cf <File.createdBy> uid(mt) .
                _:cf <File.comment> _:cc .
                _:cc <Comment.files> _:cf .
                `,
				tag,
				now,
				tag, now,
				tag,
				tag, now,
				now,
				tag,
				tag,
				now,
				now,
				tag, now,
				tag, now,
				tag, now,
				tag,
				tag, now,
			),
		}},
	}

	res, err := GetDB().UpsertDql(setup, map[string]string{})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Capture every uid we just minted.
	uids := res.Uids
	required := []string{"t", "c", "r", "f", "b", "bn", "m", "ev", "mev", "ct", "ce", "vt", "cc", "cr", "cf"}
	for _, k := range required {
		if uids[k] == "" {
			t.Fatalf("setup: missing uid for %q", k)
		}
	}
	tid := uids["t"]

	// Sanity check: tension is wired to its emitter/receiver.
	if !edgeContains(t, orgUID, "Node.tensions_out", tid) {
		t.Fatalf("setup sanity: Node.tensions_out missing tid")
	}
	if !edgeContains(t, orgUID, "Node.tensions_in", tid) {
		t.Fatalf("setup sanity: Node.tensions_in missing tid")
	}

	// Capture the Meta() response shape so we can assert storageKey surfacing.
	resp, err := GetDB().Meta("deleteTension", map[string]string{"id": tid})
	if err != nil {
		t.Fatalf("Meta(deleteTension): %v", err)
	}
	keys := make(map[string]bool, len(resp))
	for _, m := range resp {
		if k, ok := m["storageKey"].(string); ok && k != "" {
			keys[k] = true
		}
	}
	if !keys[tag+"/tcomment-file"] {
		t.Errorf("storageKey for tension-comment file not surfaced (got %v)", keys)
	}
	if !keys[tag+"/ccomment-file"] {
		t.Errorf("storageKey for contract-comment file not surfaced (got %v)", keys)
	}

	// Every captured uid must be gone.
	for _, k := range required {
		if uidExists(t, uids[k]) {
			t.Errorf("uid %s (%s) still present after delete", k, uids[k])
		}
	}

	// Reverse edges on the surviving root must no longer reference tid.
	if edgeContains(t, orgUID, "Node.tensions_out", tid) {
		t.Errorf("Node.tensions_out still references deleted tension")
	}
	if edgeContains(t, orgUID, "Node.tensions_in", tid) {
		t.Errorf("Node.tensions_in still references deleted tension")
	}
}

func TestDeleteContractDeep_Integration(t *testing.T) {
	orgUID := resolveUID(t, "Node.nameid", "test-org")
	roleUID := resolveUID(t, "Node.nameid", "test-org##@testuser")
	userUID := resolveUID(t, "User.username", "testuser")

	tag := fmt.Sprintf("dc-%d", time.Now().UnixNano())
	now := time.Now().UTC().Format(time.RFC3339)
	pendingEmail := fmt.Sprintf("pending-%s@test.co", tag)
	pendingUsername := fmt.Sprintf("pending-%s", tag)

	setup := QueryMut{
		Q: fmt.Sprintf(`query {
            org as var(func: uid(%s))
            role as var(func: uid(%s))
            mt as var(func: uid(%s))
        }`, orgUID, roleUID, userUID),
		M: []X{{
			S: fmt.Sprintf(`
                _:t <dgraph.type> "Tension" .
                _:t <Tension.title> "%s-host-tension" .
                _:t <Tension.type_> "Operational" .
                _:t <Tension.status> "Open" .
                _:t <Tension.emitter> uid(org) .
                _:t <Tension.emitterid> "test-org" .
                _:t <Tension.receiver> uid(org) .
                _:t <Tension.receiverid> "test-org" .
                _:t <Post.createdAt> "%s" .
                _:t <Post.createdBy> uid(mt) .
                uid(org) <Node.tensions_out> _:t .
                uid(org) <Node.tensions_in> _:t .

                _:ct <dgraph.type> "Contract" .
                _:ct <Contract.contractid> "%s#c" .
                _:ct <Contract.status> "Open" .
                _:ct <Contract.contract_type> "AnyCoordoDual" .
                _:ct <Post.createdAt> "%s" .
                _:ct <Post.createdBy> uid(mt) .
                _:ct <Contract.tension> _:t .
                _:t <Tension.contracts> _:ct .

                _:ce <dgraph.type> "EventFragment" .
                _:ce <EventFragment.event_type> "Created" .
                _:ct <Contract.event> _:ce .

                _:vt <dgraph.type> "Vote" .
                _:vt <Vote.voteid> "%s#vote" .
                _:vt <Vote.contract> _:ct .
                _:vt <Vote.node> uid(role) .
                _:vt <Post.createdAt> "%s" .
                _:vt <Post.createdBy> uid(mt) .
                _:ct <Contract.participants> _:vt .
                uid(role) <Node.contracts> _:vt .

                _:ct <Contract.candidates> uid(mt) .
                uid(mt) <User.contracts> _:ct .

                _:pu <dgraph.type> "PendingUser" .
                _:pu <PendingUser.username> "%s" .
                _:pu <PendingUser.email> "%s" .
                _:pu <PendingUser.contracts> _:ct .
                _:ct <Contract.pending_candidates> _:pu .

                _:cc <dgraph.type> "Comment" .
                _:cc <Post.message> "%s-ccomment" .
                _:cc <Post.createdAt> "%s" .
                _:cc <Post.createdBy> uid(mt) .
                _:ct <Contract.comments> _:cc .

                _:cr <dgraph.type> "Reaction" .
                _:cr <Reaction.reactionid> "%s#ccomment#1" .
                _:cr <Reaction.type_> "1" .
                _:cr <Reaction.user> uid(mt) .
                _:cr <Reaction.comment> _:cc .
                _:cc <Comment.reactions> _:cr .

                _:cf <dgraph.type> "File" .
                _:cf <File.storageKey> "test/%s-cfile" .
                _:cf <File.filename> "cfile.png" .
                _:cf <File.comment> _:cc .
                _:cf <File.tension> _:t .
                _:cc <Comment.files> _:cf .

                _:ue <dgraph.type> "UserEvent" .
                _:ue <UserEvent.createdAt> "%s" .
                _:ue <UserEvent.isRead> "false" .
                _:ue <UserEvent.user> uid(mt) .
                _:ue <UserEvent.event> _:ct .
                uid(mt) <User.events> _:ue .
                `,
				tag, now,
				tag, now,
				tag, now,
				pendingUsername, pendingEmail,
				tag, now,
				tag,
				tag,
				now,
			),
		}},
	}

	res, err := GetDB().UpsertDql(setup, map[string]string{})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}

	uids := res.Uids
	required := []string{"t", "ct", "ce", "vt", "pu", "cc", "cr", "cf", "ue"}
	for _, k := range required {
		if uids[k] == "" {
			t.Fatalf("setup: missing uid for %q", k)
		}
	}
	cid := uids["ct"]
	tid := uids["t"]

	// Sanity: reverse edges populated.
	if !edgeContains(t, tid, "Tension.contracts", cid) {
		t.Fatalf("setup sanity: Tension.contracts missing cid")
	}
	if !edgeContains(t, userUID, "User.contracts", cid) {
		t.Fatalf("setup sanity: User.contracts missing cid")
	}
	if !edgeContains(t, uids["pu"], "PendingUser.contracts", cid) {
		t.Fatalf("setup sanity: PendingUser.contracts missing cid")
	}
	if !edgeContains(t, roleUID, "Node.contracts", uids["vt"]) {
		t.Fatalf("setup sanity: Node.contracts missing vote uid")
	}

	if err := GetDB().DeleteContractDeep(cid); err != nil {
		t.Fatalf("DeleteContractDeep: %v", err)
	}

	// Cascaded uids gone (everything except tid + pendingUser node which the
	// template nukes — pu IS deleted as `uid(all_ids) * *` removes any uid
	// listed in all_ids; but pu isn't in all_ids. Reverse edge is removed.
	// So pu still exists, just disconnected.)
	gone := []string{"ct", "ce", "vt", "cc", "cr", "cf", "ue"}
	for _, k := range gone {
		if uidExists(t, uids[k]) {
			t.Errorf("uid %s (%s) still present after DeleteContractDeep", k, uids[k])
		}
	}

	// Reverse edges cleaned up.
	if edgeContains(t, tid, "Tension.contracts", cid) {
		t.Errorf("Tension.contracts still references deleted contract")
	}
	if edgeContains(t, userUID, "User.contracts", cid) {
		t.Errorf("User.contracts still references deleted contract")
	}
	if edgeContains(t, uids["pu"], "PendingUser.contracts", cid) {
		t.Errorf("PendingUser.contracts still references deleted contract")
	}
	if edgeContains(t, roleUID, "Node.contracts", uids["vt"]) {
		t.Errorf("Node.contracts still references deleted vote")
	}

	// The host tension and the user node must survive the contract delete.
	if !uidExists(t, tid) {
		t.Errorf("host tension removed by DeleteContractDeep")
	}
	if !uidExists(t, userUID) {
		t.Errorf("owning user removed by DeleteContractDeep")
	}

	// Cleanup: drop the host tension + dangling pending user.
	cleanup := QueryMut{
		Q: fmt.Sprintf(`query {
            t as var(func: uid(%s))
            pu as var(func: uid(%s))
            org as var(func: uid(%s))
        }`, tid, uids["pu"], orgUID),
		M: []X{{
			D: `uid(org) <Node.tensions_out> uid(t) .
                uid(org) <Node.tensions_in> uid(t) .
                uid(t) * * .
                uid(pu) * * .
                `,
		}},
	}
	if _, err := GetDB().Gamma(cleanup, map[string]string{}); err != nil {
		t.Logf("cleanup warning: %v", err)
	}
}

func TestDeleteUser_Integration(t *testing.T) {
	// Build a disposable user with one Post (Comment) authored, plus an avatar.
	tag := fmt.Sprintf("du-%d", time.Now().UnixNano())
	now := time.Now().UTC().Format(time.RFC3339)
	username := "tmpuser-" + tag
	email := "tmpuser-" + tag + "@test.co"

	setup := QueryMut{
		Q: `query {
            var(func: eq(User.username, "testuser"))
        }`,
		M: []X{{
			S: fmt.Sprintf(`
                _:u <dgraph.type> "User" .
                _:u <User.username> "%s" .
                _:u <User.email> "%s" .
                _:u <User.password> "x" .
                _:u <User.createdAt> "%s" .
                _:u <User.lastAck> "%s" .
                _:u <User.notifyByEmail> "false" .
                _:u <User.lang> "EN" .

                _:c <dgraph.type> "Comment" .
                _:c <Post.message> "%s-comment" .
                _:c <Post.createdAt> "%s" .
                _:c <Post.createdBy> _:u .

                _:f <dgraph.type> "File" .
                _:f <File.storageKey> "%s/avatar" .
                _:f <File.filename> "avatar.png" .
                _:f <File.contentType> "image/png" .
                _:f <File.size> "1" .
                _:f <File.createdAt> "%s" .
                _:f <File.createdBy> _:u .
                _:f <File.user> _:u .
                _:u <User.avatar> _:f .
                `, username, email, now, now, tag, now, tag, now),
		}},
	}

	res, err := GetDB().UpsertDql(setup, map[string]string{})
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	uids := res.Uids
	for _, k := range []string{"u", "c", "f"} {
		if uids[k] == "" {
			t.Fatalf("setup: missing uid for %q", k)
		}
	}

	// ghostid: route reauthored posts onto testuser2 to keep the ghost edge testable.
	ghostid := resolveUID(t, "User.username", "testuser2")

	// We bypass the wrapper to capture the storageKey surfacing; the wrapper
	// fires GC into a goroutine and swallows the keys.
	resp, err := GetDB().Meta("deleteUser", map[string]string{
		"username": username,
		"ghostid":  ghostid,
	})
	if err != nil {
		t.Fatalf("Meta(deleteUser): %v", err)
	}

	// (a) The avatar File must be gone.
	if uidExists(t, uids["f"]) {
		t.Errorf("avatar file %s still present", uids["f"])
	}

	// (b) Post.createdBy was rewritten to the ghost user.
	val, err := GetDB().GetByUid(uids["c"], "Post.createdBy", "User.username")
	if err != nil {
		t.Fatalf("GetByUid(comment.createdBy): %v", err)
	}
	got, _ := val.(string)
	if got != "testuser2" {
		t.Errorf("Post.createdBy = %q, want %q (ghost rewrite failed)", got, "testuser2")
	}

	// (c) The avatar's storageKey surfaced in the Meta response.
	found := false
	for _, m := range resp {
		if k, ok := m["storageKey"].(string); ok && k == tag+"/avatar" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("avatar storageKey not surfaced in Meta(deleteUser) response (got %v)", resp)
	}

	// Cleanup the ghost-rewritten comment so it doesn't leak across runs.
	cleanup := QueryMut{
		Q: fmt.Sprintf(`query { c as var(func: uid(%s)) }`, uids["c"]),
		M: []X{{D: `uid(c) * * .`}},
	}
	_, _ = GetDB().Gamma(cleanup, map[string]string{})
}
