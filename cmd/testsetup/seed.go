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

package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"strings"

	"github.com/dgraph-io/dgo/v200"
	"github.com/dgraph-io/dgo/v200/protos/api"

	"fractale/fractal6.go/internal/testutil"
	. "fractale/fractal6.go/internal/tools"
)

//go:embed seed.nq
var seedNquads string

// seedTestData inserts the shared integration test dataset (see seed.nq for
// contents). Password placeholders are substituted with bcrypt hashes here so
// the file stays a static, editor-friendly N-Quads document.
func seedTestData(dgc *dgo.Dgraph) error {
	nquads := strings.NewReplacer(
		"__PW1__", HashPassword(testutil.TestPassword),
		"__PW2__", HashPassword(testutil.TestPassword2),
	).Replace(seedNquads)

	txn := dgc.NewTxn()
	defer txn.Discard(context.Background())

	if _, err := txn.Mutate(context.Background(), &api.Mutation{
		SetNquads: []byte(nquads),
		CommitNow: true,
	}); err != nil {
		return fmt.Errorf("seed mutation: %w", err)
	}

	log.Println("Test data seeded successfully")
	return nil
}
