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

// Tests for the Moved event action (graph.MoveTension).

package graph_test

import (
	"strings"
	"testing"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph"
	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/internal/testutil"
)

const moveOrg = "move-org"

// seedMoveOrg builds a throwaway org (deleted on cleanup):
//
//	move-org
//	  ├── move-org#mv1 (Circle)   <- governed by tension tc
//	  │     └── move-org#mv2      <- └── move-org#mv3 (Circle)
//	  └── move-org##movebob (Role, governed by tension tr, receives tension ti)
//
// Returns the uids of the circle tension (tc), the role tension (tr) and the
// tension received by the role (ti).
func seedMoveOrg(t *testing.T) (string, string, string) {
	t.Helper()
	seed := db.QueryMut{
		Q: `query { u as var(func: eq(User.username, "` + testutil.TestUser + `")) }`,
		M: []db.X{{
			S: `
            _:org <dgraph.type> "Node" .
            _:org <Node.nameid> "` + moveOrg + `" .
            _:org <Node.rootnameid> "` + moveOrg + `" .
            _:org <Node.name> "Move Org" .
            _:org <Node.type_> "Circle" .
            _:org <Node.isRoot> "true" .
            _:org <Node.isArchived> "false" .
            _:org <Node.visibility> "Public" .
            _:org <Node.mode> "Coordinated" .

            _:mv1 <dgraph.type> "Node" .
            _:mv1 <Node.nameid> "` + moveOrg + `#mv1" .
            _:mv1 <Node.rootnameid> "` + moveOrg + `" .
            _:mv1 <Node.name> "mv1" .
            _:mv1 <Node.type_> "Circle" .
            _:mv1 <Node.isRoot> "false" .
            _:mv1 <Node.isArchived> "false" .
            _:mv1 <Node.parent> _:org .
            _:org <Node.children> _:mv1 .

            _:mv2 <dgraph.type> "Node" .
            _:mv2 <Node.nameid> "` + moveOrg + `#mv2" .
            _:mv2 <Node.rootnameid> "` + moveOrg + `" .
            _:mv2 <Node.name> "mv2" .
            _:mv2 <Node.type_> "Circle" .
            _:mv2 <Node.isRoot> "false" .
            _:mv2 <Node.isArchived> "false" .
            _:mv2 <Node.parent> _:mv1 .
            _:mv1 <Node.children> _:mv2 .

            _:mv3 <dgraph.type> "Node" .
            _:mv3 <Node.nameid> "` + moveOrg + `#mv3" .
            _:mv3 <Node.rootnameid> "` + moveOrg + `" .
            _:mv3 <Node.name> "mv3" .
            _:mv3 <Node.type_> "Circle" .
            _:mv3 <Node.isRoot> "false" .
            _:mv3 <Node.isArchived> "false" .
            _:mv3 <Node.parent> _:mv2 .
            _:mv2 <Node.children> _:mv3 .

            _:bob <dgraph.type> "Node" .
            _:bob <Node.nameid> "` + moveOrg + `##movebob" .
            _:bob <Node.rootnameid> "` + moveOrg + `" .
            _:bob <Node.name> "movebob" .
            _:bob <Node.type_> "Role" .
            _:bob <Node.role_type> "Peer" .
            _:bob <Node.isRoot> "false" .
            _:bob <Node.isArchived> "false" .
            _:bob <Node.parent> _:org .
            _:org <Node.children> _:bob .

            _:tc <dgraph.type> "Tension" .
            _:tc <Tension.title> "move-circle-tension" .
            _:tc <Tension.status> "Open" .
            _:tc <Tension.type_> "Governance" .
            _:tc <Tension.emitter> _:org .
            _:tc <Tension.emitterid> "` + moveOrg + `" .
            _:tc <Tension.receiver> _:org .
            _:tc <Tension.receiverid> "` + moveOrg + `" .
            _:tc <Tension.governed_node> _:mv1 .
            _:tc <Post.createdBy> uid(u) .
            _:tc <Post.createdAt> "2026-01-01T00:00:00Z" .

            _:tr <dgraph.type> "Tension" .
            _:tr <Tension.title> "move-role-tension" .
            _:tr <Tension.status> "Open" .
            _:tr <Tension.type_> "Governance" .
            _:tr <Tension.emitter> _:org .
            _:tr <Tension.emitterid> "` + moveOrg + `" .
            _:tr <Tension.receiver> _:org .
            _:tr <Tension.receiverid> "` + moveOrg + `" .
            _:tr <Tension.governed_node> _:bob .
            _:tr <Post.createdBy> uid(u) .
            _:tr <Post.createdAt> "2026-01-01T00:00:00Z" .

            _:ti <dgraph.type> "Tension" .
            _:ti <Tension.title> "move-role-inbox" .
            _:ti <Tension.status> "Open" .
            _:ti <Tension.type_> "Operational" .
            _:ti <Tension.emitter> _:org .
            _:ti <Tension.emitterid> "` + moveOrg + `" .
            _:ti <Tension.receiver> _:bob .
            _:ti <Tension.receiverid> "` + moveOrg + `##movebob" .
            _:bob <Node.tensions_in> _:ti .
            _:ti <Post.createdBy> uid(u) .
            _:ti <Post.createdAt> "2026-01-01T00:00:00Z" .
            `,
		}},
	}
	res, err := db.GetDB().UpsertDql(seed, map[string]string{})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	uids := map[string]string{}
	for _, k := range []string{"org", "mv1", "mv2", "mv3", "bob", "tc", "tr", "ti"} {
		v, ok := res.Uids[k]
		if !ok || v == "" {
			t.Fatalf("seed: missing uid for %q", k)
		}
		uids[k] = v
	}
	t.Cleanup(func() {
		for _, u := range uids {
			dq := db.QueryMut{
				Q: `query { x as var(func: uid(` + u + `)) }`,
				M: []db.X{{D: `uid(x) * * .`}},
			}
			if _, err := db.GetDB().Gamma(dq, map[string]string{}); err != nil {
				t.Logf("cleanup %s: %v", u, err)
			}
		}
	})
	return uids["tc"], uids["tr"], uids["ti"]
}

// governanceTension builds the tension payload MoveTension expects: a blob
// fragment matching the governed node (see resolveGovernanceSubject).
func governanceTension(tid, receiverid, nodeUID, nameid, name string, type_ model.NodeType) *model.Tension {
	return &model.Tension{
		ID:         tid,
		Receiverid: receiverid,
		Receiver:   &model.Node{Nameid: receiverid},
		Blobs: []*model.Blob{{
			Node: &model.NodeFragment{Name: &name, Type: &type_},
		}},
		GovernedNode: &model.Node{ID: nodeUID, Nameid: nameid, Type: type_},
	}
}

func TestMoveTension_Integration(t *testing.T) {
	tc, tr, ti := seedMoveOrg(t)
	uctx := &model.UserCtx{Username: testutil.TestUser}

	// A circle cannot be moved below itself: mv3 is a grandchild of mv1, which
	// the IsChild guard must catch (a direct child would not be enough).
	t.Run("into_own_descendant_rejected", func(t *testing.T) {
		old, new_ := moveOrg, moveOrg+"#mv3"
		tension := governanceTension(tc, old, nodeUID(t, moveOrg+"#mv1"), moveOrg+"#mv1", "mv1", model.NodeTypeCircle)
		err := graph.MoveTension(uctx, tension, &model.EventRef{
			EventType: eventType(model.TensionEventMoved), Old: &old, New: &new_,
		})
		if err == nil || !strings.Contains(err.Error(), "children") {
			t.Fatalf("MoveTension into own grandchild: err = %v, want children error", err)
		}
		// The rejected move must not have touched the tree.
		if got := parentOf(t, moveOrg+"#mv1"); got != moveOrg {
			t.Errorf("mv1 parent = %q, want %q (move was rejected)", got, moveOrg)
		}
	})

	// A role moved into another circle is renamed, and its tensions follow.
	t.Run("into_circle_renames_role", func(t *testing.T) {
		old, new_ := moveOrg, moveOrg+"#mv1"
		nameidNew := moveOrg + "#mv1#movebob"
		tension := governanceTension(tr, old, nodeUID(t, moveOrg+"##movebob"), moveOrg+"##movebob", "movebob", model.NodeTypeRole)
		if err := graph.MoveTension(uctx, tension, &model.EventRef{
			EventType: eventType(model.TensionEventMoved), Old: &old, New: &new_,
		}); err != nil {
			t.Fatalf("MoveTension: %v", err)
		}

		if got := parentOf(t, nameidNew); got != new_ {
			t.Errorf("role parent = %q, want %q", got, new_)
		}
		if tension.Receiverid != new_ {
			t.Errorf("tension.Receiverid = %q, want %q", tension.Receiverid, new_)
		}
		// Receiver of the governance tension follows the move...
		if got := tensionField(t, tr, "Tension.receiverid"); got != new_ {
			t.Errorf("governance tension receiverid = %q, want %q", got, new_)
		}
		// ...and patchNameid rewrites the ids of the tensions of the moved node.
		if got := tensionField(t, ti, "Tension.receiverid"); got != nameidNew {
			t.Errorf("inbox tension receiverid = %q, want %q", got, nameidNew)
		}
	})
}

func eventType(e model.TensionEvent) *model.TensionEvent { return &e }

func nodeUID(t *testing.T, nameid string) string {
	t.Helper()
	uids, err := db.GetDB().GetIDs("Node.nameid", nameid, nil, nil)
	if err != nil || len(uids) != 1 {
		t.Fatalf("uid of %s: %v (%v)", nameid, err, uids)
	}
	return uids[0]
}

func parentOf(t *testing.T, nameid string) string {
	t.Helper()
	v, err := db.GetDB().GetByEq("Node.nameid", nameid, "Node.parent", "Node.nameid")
	if err != nil {
		t.Fatalf("parent of %s: %v", nameid, err)
	}
	s, _ := v.(string)
	return s
}

func tensionField(t *testing.T, tid, field string) string {
	t.Helper()
	v, err := db.GetDB().GetByUid(tid, field)
	if err != nil {
		t.Fatalf("GetByUid(%s, %s): %v", tid, field, err)
	}
	s, _ := v.(string)
	return s
}
