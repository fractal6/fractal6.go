//go:build integration

/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2026 Fractale Co
 *
 * This file is part of Fractale.
 */

package graph_test

import (
	"strings"
	"testing"

	"fractale/fractal6.go/db"
	. "fractale/fractal6.go/graph"
)

// TestSyncCommentSearchMessage_Integration covers the first-comment-edit
// resync end to end: seed a throwaway tension + comments, edit the first
// comment, run the sync (what updateCommentHook fires async), and assert the
// denormalized Tension Post.message picks up the new words.
func TestSyncCommentSearchMessage_Integration(t *testing.T) {
	seed := db.QueryMut{
		Q: `query {
            org as var(func: eq(Node.nameid, "test-org"))
            u   as var(func: eq(User.username, "testuser"))
        }`,
		M: []db.X{{
			S: `
            _:tension <dgraph.type> "Tension" .
            _:tension <Tension.title> "search-sync-test-tension" .
            _:tension <Tension.status> "Open" .
            _:tension <Tension.type_> "Operational" .
            _:tension <Tension.emitter> uid(org) .
            _:tension <Tension.emitterid> "test-org" .
            _:tension <Tension.receiver> uid(org) .
            _:tension <Tension.receiverid> "test-org" .
            _:tension <Post.createdBy> uid(u) .
            _:tension <Post.createdAt> "2026-01-01T00:00:00Z" .
            _:tension <Post.message> "initial body" .

            _:c1 <dgraph.type> "Comment" .
            _:c1 <Post.createdBy> uid(u) .
            _:c1 <Post.createdAt> "2026-01-01T00:00:00Z" .
            _:c1 <Post.message> "initial body" .
            _:tension <Tension.comments> _:c1 .

            _:c2 <dgraph.type> "Comment" .
            _:c2 <Post.createdBy> uid(u) .
            _:c2 <Post.createdAt> "2026-01-01T00:01:00Z" .
            _:c2 <Post.message> "a reply" .
            _:tension <Tension.comments> _:c2 .
            `,
		}},
	}
	res, err := db.GetDB().UpsertDql(seed, map[string]string{})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	tid, c1, c2 := res.Uids["tension"], res.Uids["c1"], res.Uids["c2"]
	if tid == "" || c1 == "" || c2 == "" {
		t.Fatalf("seed: missing uids: %v", res.Uids)
	}
	t.Cleanup(func() {
		drop := db.QueryMut{
			Q: `query {}`,
			M: []db.X{{D: `<` + tid + `> * * .
                <` + c1 + `> * * .
                <` + c2 + `> * * .`}},
		}
		if _, err := db.GetDB().Gamma(drop, map[string]string{}); err != nil {
			t.Logf("cleanup: %v", err)
		}
	})

	tensionMessage := func() string {
		v, err := db.GetDB().GetByUid(tid, "Post.message")
		if err != nil {
			t.Fatalf("read tension message: %v", err)
		}
		s, _ := v.(string)
		return s
	}

	// Simulate the committed updateComment mutation: edit the FIRST comment.
	if err := db.GetDB().SetFieldById(c1, "Post.message", "edited unicornword body"); err != nil {
		t.Fatalf("edit first comment: %v", err)
	}
	if err := SyncCommentSearchMessage(c1); err != nil {
		t.Fatalf("SyncCommentSearchMessage(first): %v", err)
	}
	if got := tensionMessage(); !strings.Contains(got, "unicornword") {
		t.Errorf("tension Post.message = %q, want it to contain %q", got, "unicornword")
	}

	// Editing a NON-first comment must not touch the index.
	before := tensionMessage()
	if err := db.GetDB().SetFieldById(c2, "Post.message", "reply gets othernoise"); err != nil {
		t.Fatalf("edit second comment: %v", err)
	}
	if err := SyncCommentSearchMessage(c2); err != nil {
		t.Fatalf("SyncCommentSearchMessage(non-first): %v", err)
	}
	if got := tensionMessage(); got != before {
		t.Errorf("tension Post.message changed on non-first comment edit: %q -> %q", before, got)
	}
}
