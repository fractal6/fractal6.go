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
	"context"
	"log"
	"os"
	"testing"
	"time"

	. "fractale/fractal6.go/db"
	"fractale/fractal6.go/internal/testutil"
)

func TestMain(m *testing.M) {
	// Override the global db singleton with test-instance addresses.
	SetTestDB(testutil.TestHTTPAddr+"/graphql", testutil.TestGrpcAddr)

	// Verify test data is present (seeded by cmd/testsetup).
	ex, err := GetDB().Exists("User.username", testutil.TestUser, nil)
	if err != nil || !ex {
		log.Fatal("Test data not found. Run 'go run ./cmd/testsetup' first (or use 'make test-integration').")
	}

	os.Exit(m.Run())
}

func TestDgraphPing(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := GetDB().Ping(ctx); err != nil {
		t.Fatalf("Ping returned error: %v", err)
	}
}
