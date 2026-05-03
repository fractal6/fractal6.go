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

// testsetup drops, loads schema, and seeds the shared integration test data
// into a Dgraph instance. Run this once before integration tests.
//
// Usage:
//
//	go run ./cmd/testsetup
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dgraph-io/dgo/v200"
	"github.com/dgraph-io/dgo/v200/protos/api"
	"google.golang.org/grpc"

	"fractale/fractal6.go/internal/storage"
	"fractale/fractal6.go/internal/testutil"
)

const schemaPath = "schema/dgraph_schema.graphql"

func main() {
	log.SetFlags(log.Ltime)

	if err := waitForDgraph(60 * time.Second); err != nil {
		log.Fatalf("Dgraph not ready: %v", err)
	}

	dgc, closeConn, err := newDgraphClient()
	if err != nil {
		log.Fatalf("Dial Dgraph: %v", err)
	}
	defer closeConn()

	if err := dropAllData(dgc); err != nil {
		log.Fatalf("Failed to drop data: %v", err)
	}
	if err := loadSchema(); err != nil {
		log.Fatalf("Failed to load schema: %v", err)
	}
	// /admin/schema returns 200 as soon as the alpha accepts the schema, but
	// predicates may not yet be visible to gRPC mutations — poll until ready.
	if err := waitForPredicate(dgc, "Node.nameid", 30*time.Second); err != nil {
		log.Fatalf("Schema predicate not ready: %v", err)
	}
	if err := seedTestData(dgc); err != nil {
		log.Fatalf("Failed to seed test data: %v", err)
	}

	// Bootstrap the MinIO bucket used by /file/* integration tests. Failures are
	// fatal — the tests need it; if MinIO isn't running, the suite must be told.
	if err := ensureTestBucket(); err != nil {
		log.Fatalf("Failed to bootstrap test bucket: %v", err)
	}

	log.Println("Integration test setup complete.")
}

// ensureTestBucket creates the fixed test bucket on the MinIO instance from
// docker-compose.test.yml. Idempotent: succeeds if the bucket already exists.
func ensureTestBucket() error {
	cli, err := storage.New(storage.Config{
		Endpoint:  testutil.MinioAddr,
		Region:    "us-east-1",
		Bucket:    testutil.TestBucket,
		AccessKey: testutil.MinioAccessKey,
		SecretKey: testutil.MinioSecretKey,
		UseSSL:    false,
	})
	if err != nil {
		return fmt.Errorf("storage.New: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// Poll until MinIO is reachable, then create the bucket. The compose
	// healthcheck normally gates on `up --wait`, but be defensive.
	deadline := time.Now().Add(30 * time.Second)
	for {
		err = cli.EnsureBucket(ctx)
		if err == nil {
			log.Printf("Test bucket %q ready on %s", testutil.TestBucket, testutil.MinioAddr)
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("EnsureBucket: %w", err)
		}
		log.Printf("Waiting for MinIO at %s: %v", testutil.MinioAddr, err)
		time.Sleep(500 * time.Millisecond)
	}
}

func newDgraphClient() (*dgo.Dgraph, func(), error) {
	conn, err := grpc.Dial(testutil.TestGrpcAddr, grpc.WithInsecure()) //nolint:staticcheck
	if err != nil {
		return nil, nil, fmt.Errorf("dial gRPC: %w", err)
	}
	return dgo.NewDgraphClient(api.NewDgraphClient(conn)), func() { conn.Close() }, nil
}

func waitForDgraph(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(testutil.TestHTTPAddr + "/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				log.Println("Dgraph alpha is healthy")
				return nil
			}
		}
		log.Println("Waiting for Dgraph alpha...")
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("dgraph alpha not healthy after %s", timeout)
}

func dropAllData(dgc *dgo.Dgraph) error {
	if err := dgc.Alter(context.Background(), &api.Operation{DropAll: true}); err != nil {
		return err
	}
	log.Println("All data dropped")
	return nil
}

func loadSchema() error {
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("read schema: %w", err)
	}

	// Retry on transient "Server not ready" responses — Dgraph alpha briefly
	// reports as not ready right after DropAll while internal state resets.
	deadline := time.Now().Add(30 * time.Second)
	for attempt := 0; ; attempt++ {
		err := postSchema(schemaBytes)
		if err == nil {
			log.Println("Schema loaded successfully")
			return nil
		}
		if !isTransientErr(err) || time.Now().After(deadline) {
			return err
		}
		log.Printf("Schema upload not ready yet (attempt %d): %v — retrying", attempt+1, err)
		time.Sleep(500 * time.Millisecond)
	}
}

func postSchema(schemaBytes []byte) error {
	resp, err := http.Post(
		testutil.TestHTTPAddr+"/admin/schema",
		"application/octet-stream",
		strings.NewReader(string(schemaBytes)),
	)
	if err != nil {
		return fmt.Errorf("post schema: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("schema upload failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	// Dgraph returns HTTP 200 even when the schema is rejected — errors are
	// reported in the JSON body as {"errors":[{"message":"..."}]}.
	var parsed struct {
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &parsed); err == nil && len(parsed.Errors) > 0 {
		return fmt.Errorf("schema upload rejected: %s", parsed.Errors[0].Message)
	}
	return nil
}

func isTransientErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Server not ready") ||
		strings.Contains(msg, "Unavailable") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "EOF")
}

// waitForPredicate polls Dgraph (via gRPC) until the named predicate is
// visible in the schema, or until the timeout elapses.
func waitForPredicate(dgc *dgo.Dgraph, pred string, timeout time.Duration) error {
	query := fmt.Sprintf(`schema(pred: [%s]) { type }`, pred)
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		resp, err := dgc.NewReadOnlyTxn().Query(ctx, query)
		cancel()
		if err == nil {
			var parsed struct {
				Schema []struct {
					Type string `json:"type"`
				} `json:"schema"`
			}
			if jerr := json.Unmarshal(resp.GetJson(), &parsed); jerr == nil && len(parsed.Schema) > 0 {
				log.Printf("Schema predicate %q ready", pred)
				return nil
			}
		} else {
			lastErr = err
		}
		time.Sleep(200 * time.Millisecond)
	}
	if lastErr != nil {
		return fmt.Errorf("predicate %q not visible after %s (last err: %v)", pred, timeout, lastErr)
	}
	return fmt.Errorf("predicate %q not visible after %s", pred, timeout)
}
