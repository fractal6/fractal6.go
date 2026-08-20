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

// Tests for the recursive archive DQL upsert.

package db_test

import (
	"testing"

	. "fractale/fractal6.go/db"
	"fractale/fractal6.go/internal/testutil"
)

// seedArchiveTree builds an isolated 3-level subtree under test-org:
//
//	test-org#arc1 (Circle)
//	  └── test-org#arc1#arc2 (Circle, already archived child arc2b)
//	        └── test-org#arc1#arc2#@bob (Role, first_link testuser)
//
// with one Open tension received by arc1 and one by the deep role, plus one
// already Closed tension on arc1 (must not be touched).
func seedArchiveTree(t *testing.T) []string {
	t.Helper()
	seed := QueryMut{
		Q: `query {
            org as var(func: eq(Node.nameid, "test-org"))
            u   as var(func: eq(User.username, "` + testutil.TestUser + `"))
        }`,
		M: []X{{
			S: `
            _:c1 <dgraph.type> "Node" .
            _:c1 <Node.nameid> "test-org#arc1" .
            _:c1 <Node.rootnameid> "test-org" .
            _:c1 <Node.name> "arc1" .
            _:c1 <Node.type_> "Circle" .
            _:c1 <Node.isArchived> "false" .
            _:c1 <Node.isRoot> "false" .
            _:c1 <Node.parent> uid(org) .
            uid(org) <Node.children> _:c1 .

            _:c2 <dgraph.type> "Node" .
            _:c2 <Node.nameid> "test-org#arc1#arc2" .
            _:c2 <Node.rootnameid> "test-org" .
            _:c2 <Node.name> "arc2" .
            _:c2 <Node.type_> "Circle" .
            _:c2 <Node.isArchived> "false" .
            _:c2 <Node.isRoot> "false" .
            _:c2 <Node.parent> _:c1 .
            _:c1 <Node.children> _:c2 .

            _:r3 <dgraph.type> "Node" .
            _:r3 <Node.nameid> "test-org#arc1#arc2#@bob" .
            _:r3 <Node.rootnameid> "test-org" .
            _:r3 <Node.name> "bob" .
            _:r3 <Node.type_> "Role" .
            _:r3 <Node.role_type> "Peer" .
            _:r3 <Node.isArchived> "false" .
            _:r3 <Node.isRoot> "false" .
            _:r3 <Node.parent> _:c2 .
            _:r3 <Node.first_link> uid(u) .
            _:c2 <Node.children> _:r3 .

            _:t1 <dgraph.type> "Tension" .
            _:t1 <Tension.title> "arc-open-1" .
            _:t1 <Tension.status> "Open" .
            _:t1 <Tension.type_> "Operational" .
            _:t1 <Tension.emitter> uid(org) .
            _:t1 <Tension.emitterid> "test-org" .
            _:t1 <Tension.receiver> _:c1 .
            _:t1 <Tension.receiverid> "test-org#arc1" .
            _:t1 <Post.createdBy> uid(u) .
            _:t1 <Post.createdAt> "2026-01-01T00:00:00Z" .

            _:t2 <dgraph.type> "Tension" .
            _:t2 <Tension.title> "arc-open-2" .
            _:t2 <Tension.status> "Open" .
            _:t2 <Tension.type_> "Operational" .
            _:t2 <Tension.emitter> uid(org) .
            _:t2 <Tension.emitterid> "test-org" .
            _:t2 <Tension.receiver> _:r3 .
            _:t2 <Tension.receiverid> "test-org#arc1#arc2#@bob" .
            _:t2 <Post.createdBy> uid(u) .
            _:t2 <Post.createdAt> "2026-01-01T00:00:00Z" .

            _:t3 <dgraph.type> "Tension" .
            _:t3 <Tension.title> "arc-closed-1" .
            _:t3 <Tension.status> "Closed" .
            _:t3 <Tension.type_> "Operational" .
            _:t3 <Tension.emitter> uid(org) .
            _:t3 <Tension.emitterid> "test-org" .
            _:t3 <Tension.receiver> _:c1 .
            _:t3 <Tension.receiverid> "test-org#arc1" .
            _:t3 <Post.createdBy> uid(u) .
            _:t3 <Post.createdAt> "2026-01-01T00:00:00Z" .
            `,
		}},
	}
	res, err := GetDB().UpsertDql(seed, map[string]string{})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	uids := make([]string, 0, 6)
	for _, k := range []string{"c1", "c2", "r3", "t1", "t2", "t3"} {
		v, ok := res.Uids[k]
		if !ok || v == "" {
			t.Fatalf("seed: missing uid for %q", k)
		}
		uids = append(uids, v)
	}
	t.Cleanup(func() {
		for _, u := range uids {
			dq := QueryMut{
				Q: `query { x as var(func: uid(` + u + `)) }`,
				M: []X{{D: `uid(x) * * .`}},
			}
			if _, err := GetDB().Gamma(dq, map[string]string{}); err != nil {
				t.Logf("cleanup %s: %v", u, err)
			}
		}
	})
	return uids
}

func TestArchiveNodesRecursive_Integration(t *testing.T) {
	seedArchiveTree(t)

	nodes, err := GetDB().ArchiveNodesRecursive("test-org#arc1")
	if err != nil {
		t.Fatalf("ArchiveNodesRecursive: %v", err)
	}

	got := map[string]string{}
	for _, n := range nodes {
		got[n.Nameid] = n.FirstLink
	}
	// Depth 3: the @recurse var must accumulate every level.
	for _, nid := range []string{"test-org#arc1", "test-org#arc1#arc2", "test-org#arc1#arc2#@bob"} {
		if _, ok := got[nid]; !ok {
			t.Errorf("missing archived node %q, got %v", nid, got)
		}
		v, err := GetDB().GetByEq("Node.nameid", nid, "Node.isArchived")
		if err != nil {
			t.Fatalf("GetByEq(%s): %v", nid, err)
		}
		if b, _ := v.(bool); !b {
			t.Errorf("node %q isArchived = %v, want true", nid, v)
		}
	}
	if got["test-org#arc1#arc2#@bob"] != testutil.TestUser {
		t.Errorf("first_link = %q, want %q", got["test-org#arc1#arc2#@bob"], testutil.TestUser)
	}
	// The root org must not be touched.
	v, err := GetDB().GetByEq("Node.nameid", "test-org", "Node.isArchived")
	if err != nil {
		t.Fatalf("GetByEq(test-org): %v", err)
	}
	if b, _ := v.(bool); b {
		t.Error("test-org isArchived = true, want false (recursion must go down only)")
	}
}
