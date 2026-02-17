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

package db

import (
	"log"
	"os"
	"testing"

	"fractale/fractal6.go/internal/testutil"
)

func TestMain(m *testing.M) {
	// Override the global DB singleton with test-instance addresses.
	DB = &Dgraph{
		gqlAddr:  testutil.TestHTTPAddr + "/graphql",
		grpcAddr: testutil.TestGrpcAddr,
	}

	// Verify test data is present (seeded by cmd/testsetup).
	ex, err := DB.Exists("User.username", testutil.TestUser, nil)
	if err != nil || !ex {
		log.Fatal("Test data not found. Run 'go run ./cmd/testsetup' first (or use 'make test-integration').")
	}

	os.Exit(m.Run())
}
