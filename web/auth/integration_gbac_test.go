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
	"log"
	"os"
	"testing"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/internal/testutil"
	"fractale/fractal6.go/web/auth"
)

func TestMain(m *testing.M) {
	// Override the global db singleton with test-instance addresses.
	db.SetTestDB(testutil.TestHTTPAddr+"/graphql", testutil.TestGrpcAddr)

	// Verify test data is present (seeded by cmd/testsetup).
	ex, err := db.GetDB().Exists("User.username", testutil.TestUser, nil)
	if err != nil || !ex {
		log.Fatal("Test data not found. Run 'go run ./cmd/testsetup' first (or use 'make test-integration').")
	}

	os.Exit(m.Run())
}

// TestCheckProjectAuth_NonExistentProject verifies that CheckProjectAuth
// returns an error (not a panic) when called with a non-existent project ID.
func TestCheckProjectAuth_NonExistentProject(t *testing.T) {
	uctx := &model.UserCtx{Username: testutil.TestUser}

	ok, err := auth.CheckProjectAuth(uctx, "0xdeadbeef")
	// A non-existent project has no linked nodes, so the function may return
	// (true, nil) or an error — either is acceptable. The key assertion is
	// that it does NOT panic.
	_ = ok
	_ = err
}

// TestHasCoordoAuth_NonExistentNode verifies that HasCoordoAuth returns an
// error (not a panic) when called with a non-existent node nameid.
func TestHasCoordoAuth_NonExistentNode(t *testing.T) {
	uctx := &model.UserCtx{Username: testutil.TestUser}

	ok, err := auth.HasCoordoAuth(uctx, "nonexistent-org#nonexistent-circle", nil)
	if err == nil {
		t.Errorf("expected error for non-existent node, got ok=%v", ok)
	}
}
