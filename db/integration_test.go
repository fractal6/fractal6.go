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
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dgraph-io/dgo/v200/protos/api"
)

const (
	testGrpcAddr = "localhost:9180"
	testHTTPAddr = "http://localhost:8180"
	testSchemaPath = "../schema/dgraph_schema.graphql"
)

func TestMain(m *testing.M) {
	// Override the global DB singleton with test-instance addresses.
	// init() only stores address strings, no connections are opened.
	DB = &Dgraph{
		gqlAddr:  testHTTPAddr + "/graphql",
		grpcAddr: testGrpcAddr,
	}

	// Wait for Dgraph alpha to be healthy
	if err := waitForDgraph(60 * time.Second); err != nil {
		log.Fatalf("Dgraph not ready: %v", err)
	}

	// Drop all data from previous runs to avoid duplicates
	if err := dropAllData(); err != nil {
		log.Fatalf("Failed to drop data: %v", err)
	}

	// Load the GraphQL schema into Dgraph
	if err := loadSchema(); err != nil {
		log.Fatalf("Failed to load schema: %v", err)
	}

	// Wait briefly for schema to be applied
	time.Sleep(2 * time.Second)

	// Seed test data via gRPC/DQL
	if err := seedTestData(); err != nil {
		log.Fatalf("Failed to seed test data: %v", err)
	}

	os.Exit(m.Run())
}

// waitForDgraph polls the Dgraph alpha health endpoint until ready or timeout.
func waitForDgraph(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(testHTTPAddr + "/health")
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

// loadSchema reads the generated Dgraph schema file and POSTs it to the admin endpoint.
func loadSchema() error {
	schemaBytes, err := ioutil.ReadFile(testSchemaPath)
	if err != nil {
		return fmt.Errorf("read schema: %w", err)
	}

	resp, err := http.Post(
		testHTTPAddr+"/admin/schema",
		"application/octet-stream",
		strings.NewReader(string(schemaBytes)),
	)
	if err != nil {
		return fmt.Errorf("post schema: %w", err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("schema upload failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	log.Println("Schema loaded successfully")
	return nil
}

// seedTestData inserts the minimum test dataset via gRPC N-Quads mutation.
// Data layout:
//   - 1 root Node (Circle): test-org#
//   - 1 User: testuser (with UserRights)
//   - 1 Member role node: test-org#@testuser (child of org, linked to user)
//   - 1 Coordinator role node: test-org#:coordo (child of org)
//   - 1 Tension (Open, Operational): emitter=org, receiver=org
//   - 1 Event (Created): linked to tension history
func seedTestData() error {
	dgc, cancel := DB.getDgraphClient()
	defer cancel()
	ctx := context.Background()

	nquads := `
		_:org <dgraph.type> "Node" .
		_:org <Node.nameid> "test-org#" .
		_:org <Node.rootnameid> "test-org#" .
		_:org <Node.name> "Test Org" .
		_:org <Node.about> "A test organisation" .
		_:org <Node.isRoot> "true" .
		_:org <Node.type_> "Circle" .
		_:org <Node.visibility> "Public" .
		_:org <Node.mode> "Coordinated" .
		_:org <Node.rights> "0" .
		_:org <Node.isArchived> "false" .
		_:org <Node.createdAt> "2026-01-01T00:00:00Z" .

		_:rights <dgraph.type> "UserRights" .
		_:rights <UserRights.type_> "Regular" .
		_:rights <UserRights.canLogin> "true" .
		_:rights <UserRights.canCreateRoot> "false" .
		_:rights <UserRights.maxPublicOrga> "5" .
		_:rights <UserRights.maxPrivateOrga> "5" .
		_:rights <UserRights.hasEmailNotifications> "false" .

		_:user <dgraph.type> "User" .
		_:user <User.username> "testuser" .
		_:user <User.email> "testuser@test.co" .
		_:user <User.password> "hashed" .
		_:user <User.name> "Test User" .
		_:user <User.createdAt> "2026-01-01T00:00:00Z" .
		_:user <User.lastAck> "2026-01-01T00:00:00Z" .
		_:user <User.notifyByEmail> "false" .
		_:user <User.lang> "EN" .
		_:user <User.rights> _:rights .

		_:org <Node.createdBy> _:user .

		_:member <dgraph.type> "Node" .
		_:member <Node.nameid> "test-org#@testuser" .
		_:member <Node.rootnameid> "test-org#" .
		_:member <Node.name> "testuser" .
		_:member <Node.isRoot> "false" .
		_:member <Node.type_> "Role" .
		_:member <Node.role_type> "Member" .
		_:member <Node.visibility> "Public" .
		_:member <Node.mode> "Coordinated" .
		_:member <Node.rights> "0" .
		_:member <Node.isArchived> "false" .
		_:member <Node.parent> _:org .
		_:member <Node.first_link> _:user .
		_:member <Node.createdBy> _:user .
		_:member <Node.createdAt> "2026-01-01T00:00:00Z" .
		_:org <Node.children> _:member .
		_:user <User.roles> _:member .

		_:coordo <dgraph.type> "Node" .
		_:coordo <Node.nameid> "test-org#:coordo" .
		_:coordo <Node.rootnameid> "test-org#" .
		_:coordo <Node.name> "Coordinator" .
		_:coordo <Node.isRoot> "false" .
		_:coordo <Node.type_> "Role" .
		_:coordo <Node.role_type> "Coordinator" .
		_:coordo <Node.visibility> "Public" .
		_:coordo <Node.mode> "Coordinated" .
		_:coordo <Node.rights> "0" .
		_:coordo <Node.isArchived> "false" .
		_:coordo <Node.parent> _:org .
		_:coordo <Node.first_link> _:user .
		_:coordo <Node.createdBy> _:user .
		_:coordo <Node.createdAt> "2026-01-01T00:00:00Z" .
		_:org <Node.children> _:coordo .

		_:tension <dgraph.type> "Tension" .
		_:tension <Tension.title> "Test tension" .
		_:tension <Tension.status> "Open" .
		_:tension <Tension.type_> "Operational" .
		_:tension <Tension.emitter> _:org .
		_:tension <Tension.emitterid> "test-org#" .
		_:tension <Tension.receiver> _:org .
		_:tension <Tension.receiverid> "test-org#" .
		_:tension <Post.createdBy> _:user .
		_:tension <Post.createdAt> "2026-01-01T00:00:00Z" .
		_:tension <Post.message> "Initial tension message" .
		_:org <Node.tensions_out> _:tension .
		_:org <Node.tensions_in> _:tension .

		_:event <dgraph.type> "Event" .
		_:event <Event.event_type> "Created" .
		_:event <Event.tension> _:tension .
		_:event <Post.createdBy> _:user .
		_:event <Post.createdAt> "2026-01-01T00:00:00Z" .
		_:tension <Tension.history> _:event .
	`

	mu := &api.Mutation{
		SetNquads: []byte(nquads),
		CommitNow: true,
	}

	txn := dgc.NewTxn()
	defer txn.Discard(ctx)

	_, err := txn.Mutate(ctx, mu)
	if err != nil {
		return fmt.Errorf("seed mutation: %w", err)
	}

	log.Println("Test data seeded successfully")
	return nil
}

// dropAllData drops all data in the test Dgraph instance (available for manual use).
func dropAllData() error {
	dgc, cancel := DB.getDgraphClient()
	defer cancel()
	ctx := context.Background()
	return dgc.Alter(ctx, &api.Operation{DropAll: true})
}
