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

	"fractale/fractal6.go/internal/testutil"
	. "fractale/fractal6.go/internal/tools"
)

const (
	schemaPath = "schema/dgraph_schema.graphql"
)

func main() {
	log.SetFlags(log.Ltime)

	// 1. Wait for Dgraph alpha to be healthy
	if err := waitForDgraph(60 * time.Second); err != nil {
		log.Fatalf("Dgraph not ready: %v", err)
	}

	// 2. Drop all data
	if err := dropAllData(); err != nil {
		log.Fatalf("Failed to drop data: %v", err)
	}

	// 3. Load schema
	if err := loadSchema(); err != nil {
		log.Fatalf("Failed to load schema: %v", err)
	}

	// Wait for schema to be applied
	time.Sleep(2 * time.Second)

	// 4. Seed test data
	if err := seedTestData(); err != nil {
		log.Fatalf("Failed to seed test data: %v", err)
	}

	log.Println("Integration test setup complete.")
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

func newDgraphClient() (*dgo.Dgraph, func()) {
	conn, err := grpc.Dial(testutil.TestGrpcAddr, grpc.WithInsecure()) //nolint:staticcheck
	if err != nil {
		log.Fatal("While trying to dial gRPC: ", err)
	}
	dgClient := dgo.NewDgraphClient(api.NewDgraphClient(conn))
	return dgClient, func() { conn.Close() }
}

func dropAllData() error {
	dgc, cancel := newDgraphClient()
	defer cancel()
	err := dgc.Alter(context.Background(), &api.Operation{DropAll: true})
	if err != nil {
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

	log.Println("Schema loaded successfully")
	return nil
}

// seedTestData inserts the shared test dataset via gRPC N-Quads mutation.
// This dataset is the superset used by all integration test packages:
//   - 1 root Node (Circle): test-org (with userCanJoin, guestCanCreateTension, source blob)
//   - 2 Users: testuser, testuser2 (with real bcrypt password hashes and UserRights)
//   - 1 Owner role node: test-org##@testuser (child of org, linked to testuser)
//   - 1 Coordinator role node: test-org##:coordo (child of org, linked to testuser)
//   - 1 Tension (Open, Operational) with blob and event
func seedTestData() error {
	dgc, cancel := newDgraphClient()
	defer cancel()

	hashedPw1 := HashPassword(testutil.TestPassword)
	hashedPw2 := HashPassword(testutil.TestPassword2)

	nquads := fmt.Sprintf(`
		_:org <dgraph.type> "Node" .
		_:org <Node.nameid> "test-org" .
		_:org <Node.rootnameid> "test-org" .
		_:org <Node.name> "Test Org" .
		_:org <Node.about> "A test organisation" .
		_:org <Node.isRoot> "true" .
		_:org <Node.type_> "Circle" .
		_:org <Node.visibility> "Public" .
		_:org <Node.mode> "Coordinated" .
		_:org <Node.rights> "0" .
		_:org <Node.isArchived> "false" .
		_:org <Node.createdAt> "2026-01-01T00:00:00Z" .
		_:org <Node.userCanJoin> "true" .
		_:org <Node.guestCanCreateTension> "true" .

		_:rights1 <dgraph.type> "UserRights" .
		_:rights1 <UserRights.type_> "Regular" .
		_:rights1 <UserRights.canLogin> "true" .
		_:rights1 <UserRights.canCreateRoot> "true" .
		_:rights1 <UserRights.maxPublicOrga> "5" .
		_:rights1 <UserRights.maxPrivateOrga> "5" .
		_:rights1 <UserRights.hasEmailNotifications> "false" .

		_:user1 <dgraph.type> "User" .
		_:user1 <User.username> "testuser" .
		_:user1 <User.email> "testuser@test.co" .
		_:user1 <User.password> "%s" .
		_:user1 <User.name> "Test User" .
		_:user1 <User.createdAt> "2026-01-01T00:00:00Z" .
		_:user1 <User.lastAck> "2026-01-01T00:00:00Z" .
		_:user1 <User.notifyByEmail> "false" .
		_:user1 <User.lang> "EN" .
		_:user1 <User.rights> _:rights1 .

		_:org <Node.createdBy> _:user1 .

		_:rights2 <dgraph.type> "UserRights" .
		_:rights2 <UserRights.type_> "Regular" .
		_:rights2 <UserRights.canLogin> "true" .
		_:rights2 <UserRights.canCreateRoot> "true" .
		_:rights2 <UserRights.maxPublicOrga> "5" .
		_:rights2 <UserRights.maxPrivateOrga> "5" .
		_:rights2 <UserRights.hasEmailNotifications> "false" .

		_:user2 <dgraph.type> "User" .
		_:user2 <User.username> "testuser2" .
		_:user2 <User.email> "testuser2@test.co" .
		_:user2 <User.password> "%s" .
		_:user2 <User.name> "Test User 2" .
		_:user2 <User.createdAt> "2026-01-01T00:00:00Z" .
		_:user2 <User.lastAck> "2026-01-01T00:00:00Z" .
		_:user2 <User.notifyByEmail> "false" .
		_:user2 <User.lang> "EN" .
		_:user2 <User.rights> _:rights2 .

		_:member1 <dgraph.type> "Node" .
		_:member1 <Node.nameid> "test-org##@testuser" .
		_:member1 <Node.rootnameid> "test-org" .
		_:member1 <Node.name> "testuser" .
		_:member1 <Node.isRoot> "false" .
		_:member1 <Node.type_> "Role" .
		_:member1 <Node.role_type> "Owner" .
		_:member1 <Node.visibility> "Public" .
		_:member1 <Node.mode> "Coordinated" .
		_:member1 <Node.rights> "0" .
		_:member1 <Node.isArchived> "false" .
		_:member1 <Node.parent> _:org .
		_:member1 <Node.first_link> _:user1 .
		_:member1 <Node.createdBy> _:user1 .
		_:member1 <Node.createdAt> "2026-01-01T00:00:00Z" .
		_:org <Node.children> _:member1 .
		_:user1 <User.roles> _:member1 .

		_:coordo <dgraph.type> "Node" .
		_:coordo <Node.nameid> "test-org##:coordo" .
		_:coordo <Node.rootnameid> "test-org" .
		_:coordo <Node.name> "Coordinator" .
		_:coordo <Node.isRoot> "false" .
		_:coordo <Node.type_> "Role" .
		_:coordo <Node.role_type> "Coordinator" .
		_:coordo <Node.visibility> "Public" .
		_:coordo <Node.mode> "Coordinated" .
		_:coordo <Node.rights> "0" .
		_:coordo <Node.isArchived> "false" .
		_:coordo <Node.parent> _:org .
		_:coordo <Node.first_link> _:user1 .
		_:coordo <Node.createdBy> _:user1 .
		_:coordo <Node.createdAt> "2026-01-01T00:00:00Z" .
		_:org <Node.children> _:coordo .

		_:tension <dgraph.type> "Tension" .
		_:tension <Tension.title> "Test tension" .
		_:tension <Tension.status> "Open" .
		_:tension <Tension.type_> "Operational" .
		_:tension <Tension.emitter> _:org .
		_:tension <Tension.emitterid> "test-org" .
		_:tension <Tension.receiver> _:org .
		_:tension <Tension.receiverid> "test-org" .
		_:tension <Post.createdBy> _:user1 .
		_:tension <Post.createdAt> "2026-01-01T00:00:00Z" .
		_:tension <Post.message> "Initial tension message" .
		_:org <Node.tensions_out> _:tension .
		_:org <Node.tensions_in> _:tension .

		_:blob <dgraph.type> "Blob" .
		_:blob <Blob.blob_type> "OnNode" .
		_:blob <Blob.tension> _:tension .
		_:blob <Post.createdBy> _:user1 .
		_:blob <Post.createdAt> "2026-01-01T00:00:00Z" .
		_:tension <Tension.blobs> _:blob .
		_:org <Node.source> _:blob .

		_:event <dgraph.type> "Event" .
		_:event <Event.event_type> "Created" .
		_:event <Event.tension> _:tension .
		_:event <Post.createdBy> _:user1 .
		_:event <Post.createdAt> "2026-01-01T00:00:00Z" .
		_:tension <Tension.history> _:event .
	`, hashedPw1, hashedPw2)

	txn := dgc.NewTxn()
	defer txn.Discard(context.Background())

	_, err := txn.Mutate(context.Background(), &api.Mutation{
		SetNquads: []byte(nquads),
		CommitNow: true,
	})
	if err != nil {
		return fmt.Errorf("seed mutation: %w", err)
	}

	log.Println("Test data seeded successfully")
	return nil
}
