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

package graph_test

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"fractale/fractal6.go/db"
	. "fractale/fractal6.go/graph"
	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/internal/testutil"
)

// createTestComment creates a comment on the given tension, authored by the given user UID.
// Returns the comment UID. Caller is responsible for cleanup.
func createTestComment(t *testing.T, tensionUID, userUID string) string {
	t.Helper()
	qm := db.QueryMut{
		Q: `query {
			t as var(func: uid(` + tensionUID + `))
			u as var(func: uid(` + userUID + `))
		}`,
		M: []db.X{{
			S: `_:comment <dgraph.type> "Comment" .
				_:comment <Post.createdBy> uid(u) .
				_:comment <Post.createdAt> "2026-01-15T00:00:00Z" .
				_:comment <Post.message> "test comment for deletion" .
				uid(t) <Tension.comments> _:comment .`,
		}},
	}
	results, err := db.GetDB().Gamma(qm, map[string]string{})
	if err != nil {
		t.Fatalf("createTestComment: %v", err)
	}
	// Gamma returns the Uids map in the response; extract comment UID
	_ = results
	// Use GetIDs to find the comment we just created by querying tension comments
	// Instead, we use the mutation response. Gamma returns []map[string]any.
	// The uid is in the Uids map of the dgraph response, but Gamma returns query results.
	// We'll query for the comment we just created.
	ids, err := db.GetDB().GetIDs("Post.message", "test comment for deletion", nil, nil)
	if err != nil || len(ids) == 0 {
		t.Fatalf("createTestComment: could not find created comment: %v", err)
	}
	return ids[len(ids)-1] // return the latest one
}

// deleteTestComment removes a comment node by UID (cleanup helper).
func deleteTestComment(t *testing.T, tensionUID, commentUID string) {
	t.Helper()
	qm := db.QueryMut{
		Q: `query {
			t as var(func: uid(` + tensionUID + `))
			c as var(func: uid(` + commentUID + `))
		}`,
		M: []db.X{{
			D: `uid(t) <Tension.comments> uid(c) .
				uid(c) * * .`,
		}},
	}
	_, _ = db.GetDB().Gamma(qm, map[string]string{})
}

func TestRemoveComment_OwnerCanDelete(t *testing.T) {
	// Get the seeded tension UID
	tids, err := db.GetDB().GetIDs("Tension.title", "Test tension", nil, nil)
	if err != nil || len(tids) == 0 {
		t.Fatalf("could not find seeded tension: %v", err)
	}
	tensionUID := tids[0]

	// Get testuser UID
	uids, err := db.GetDB().GetIDs("User.username", testutil.TestUser, nil, nil)
	if err != nil || len(uids) == 0 {
		t.Fatalf("could not find testuser: %v", err)
	}
	userUID := uids[0]

	// Create a comment authored by testuser
	commentUID := createTestComment(t, tensionUID, userUID)
	defer deleteTestComment(t, tensionUID, commentUID) // cleanup in case test fails early

	// Build the UserCtx and call RemoveComment as the comment author
	uctx := &model.UserCtx{Username: testutil.TestUser}
	tension := &model.Tension{ID: tensionUID}
	event := &model.EventRef{Old: &commentUID}

	ok, err := RemoveComment(uctx, tension, event, nil)
	if err != nil {
		t.Fatalf("RemoveComment by author should succeed, got error: %v", err)
	}
	if !ok {
		t.Fatal("RemoveComment by author returned false, expected true")
	}
}

func TestRemoveComment_NonOwnerCannotDelete(t *testing.T) {
	// Get the seeded tension UID
	tids, err := db.GetDB().GetIDs("Tension.title", "Test tension", nil, nil)
	if err != nil || len(tids) == 0 {
		t.Fatalf("could not find seeded tension: %v", err)
	}
	tensionUID := tids[0]

	// Get testuser UID (comment author)
	uids, err := db.GetDB().GetIDs("User.username", testutil.TestUser, nil, nil)
	if err != nil || len(uids) == 0 {
		t.Fatalf("could not find testuser: %v", err)
	}
	userUID := uids[0]

	// Create a comment authored by testuser
	commentUID := createTestComment(t, tensionUID, userUID)
	defer deleteTestComment(t, tensionUID, commentUID)

	// Try to delete as testuser2 (not the comment author)
	uctx := &model.UserCtx{Username: testutil.TestUser2}
	tension := &model.Tension{ID: tensionUID}
	event := &model.EventRef{Old: &commentUID}

	ok, err := RemoveComment(uctx, tension, event, nil)
	if err == nil {
		t.Fatal("RemoveComment by non-author should fail, but got no error")
	}
	if !strings.Contains(err.Error(), "Only the author of the comment can delete it") {
		t.Fatalf("expected 'Only the author of the comment can delete it' error, got: %v", err)
	}
	if ok {
		t.Fatal("RemoveComment by non-author returned true, expected false")
	}
}

// TestRemoveComment_DeletesAttachedFiles is the end-to-end test for the
// comment-delete S3 GC hook. It exercises the actual resolver path
// (RemoveComment → DeleteCommentDeep → deleteStorageKeysAsync → storage.Delete)
// so a regression that breaks the wiring between RemoveComment and
// storage.Global() will be caught here.
func TestRemoveComment_DeletesAttachedFiles(t *testing.T) {
	if testStorageCli == nil {
		t.Skip("storage client not initialised — see graph/integration_test.go::TestMain")
	}

	tids, err := db.GetDB().GetIDs("Tension.title", "Test tension", nil, nil)
	if err != nil || len(tids) == 0 {
		t.Fatalf("could not find seeded tension: %v", err)
	}
	tensionUID := tids[0]

	uids, err := db.GetDB().GetIDs("User.username", testutil.TestUser, nil, nil)
	if err != nil || len(uids) == 0 {
		t.Fatalf("could not find testuser: %v", err)
	}
	userUID := uids[0]

	commentUID := createTestComment(t, tensionUID, userUID)
	defer deleteTestComment(t, tensionUID, commentUID) // safety net if test fails before RemoveComment

	// Drop a small object straight via the storage client and register it as
	// a File node attached to the comment. This sidesteps the HTTP handler
	// (which lives in another package) but exercises the same db + S3 surface
	// that RemoveComment will GC.
	body := []byte("hello-from-graph-gc-test")
	storageKey := fmt.Sprintf("comments/%s/test-%d.bin", commentUID, time.Now().UnixNano())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if putErr := testStorageCli.Put(ctx, storageKey, bytes.NewReader(body), int64(len(body)), "application/octet-stream"); putErr != nil {
		cancel()
		t.Fatalf("storage Put: %v", putErr)
	}
	cancel()

	fileUID, err := db.GetDB().AddCommentFile(
		tensionUID, commentUID, testutil.TestUser, "test.bin", "application/octet-stream",
		storageKey, int64(len(body)), time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		// Best-effort: drop the orphan S3 object before failing.
		ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
		_ = testStorageCli.Delete(ctx2, storageKey)
		cancel2()
		t.Fatalf("AddCommentFile: %v", err)
	}
	t.Cleanup(func() { _ = db.GetDB().DeleteFile(fileUID) })

	// Sanity: object visible to MinIO before we remove the comment.
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	exists, err := testStorageCli.Exists(ctx, storageKey)
	cancel()
	if err != nil {
		t.Fatalf("storage Exists pre-check: %v", err)
	}
	if !exists {
		t.Fatalf("precondition: object %q should exist before RemoveComment", storageKey)
	}

	// Drive the actual resolver path. RemoveComment reads storage.Global() —
	// TestMain registered testStorageCli there.
	uctx := &model.UserCtx{Username: testutil.TestUser}
	tension := &model.Tension{ID: tensionUID}
	event := &model.EventRef{Old: &commentUID}
	ok, err := RemoveComment(uctx, tension, event, nil)
	if err != nil {
		t.Fatalf("RemoveComment failed: %v", err)
	}
	if !ok {
		t.Fatal("RemoveComment returned false")
	}

	// Object must be gone from MinIO. RemoveComment fires the S3 delete in a
	// goroutine, so poll briefly. The comment node + File node are dropped
	// synchronously by the deleteComment template, so deleteTestComment in
	// defer is idempotent (the existing helper tolerates missing nodes).
	deadline := time.Now().Add(5 * time.Second)
	for {
		ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
		exists, err = testStorageCli.Exists(ctx, storageKey)
		cancel()
		if err != nil {
			t.Fatalf("storage Exists post-check: %v", err)
		}
		if !exists {
			break
		}
		if time.Now().After(deadline) {
			t.Errorf("expected object %q to be removed by RemoveComment", storageKey)
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
}
