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
	"context"
	"log"
	"os"
	"testing"
	"time"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/internal/storage"
	"fractale/fractal6.go/internal/testutil"
)

// testStorageCli is the MinIO-backed client used by the file-attachment
// portion of TestRemoveComment_DeletesAttachedFiles. Initialised in TestMain
// and registered as the storage package global so the cascade-delete async
// GC (deleteStorageKeysAsync) resolves to the same backing.
var testStorageCli *storage.Client

func TestMain(m *testing.M) {
	// Override DB with test-instance addresses.
	db.SetTestDB(testutil.TestHTTPAddr+"/graphql", testutil.TestGrpcAddr)

	// Verify test data is present (seeded by cmd/testsetup).
	ex, err := db.GetDB().Exists("User.username", testutil.TestUser, nil)
	if err != nil || !ex {
		log.Fatal("Test data not found. Run 'go run ./cmd/testsetup' first (or use 'make test-integration').")
	}

	// Storage client. Failure here is non-fatal: only the file-attachment GC
	// test depends on it, and it skips when testStorageCli is nil.
	cli, err := storage.New(storage.Config{
		Endpoint:  testutil.MinioAddr,
		Region:    "us-east-1",
		Bucket:    testutil.TestBucket,
		AccessKey: testutil.MinioAccessKey,
		SecretKey: testutil.MinioSecretKey,
		UseSSL:    false,
	})
	if err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if bErr := cli.EnsureBucket(ctx); bErr != nil {
			log.Printf("MinIO not ready (%v) — graph file-GC test will skip", bErr)
			cli = nil
		}
		cancel()
	} else {
		log.Printf("storage.New failed (%v) — graph file-GC test will skip", err)
		cli = nil
	}
	testStorageCli = cli
	storage.SetGlobal(cli)

	os.Exit(m.Run())
}
