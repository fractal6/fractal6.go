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
//   - 1 root Node (Circle): test-org (Public, with userCanJoin, guestCanCreateTension, source blob)
//   - 2 Users: testuser, testuser2 (with real bcrypt password hashes and UserRights)
//   - 1 Owner role node: test-org##@testuser (child of org, linked to testuser)
//   - 1 Coordinator role node: test-org##:coordo (child of org, linked to testuser)
//   - 1 Tension (Open, Operational) with blob and event
//   - 1 root Node (Circle): sec-org (Private, for visibility/security tests)
//   - 2 Sub-circles: sec-org#private-circle (Private), sec-org#secret-circle (Secret)
//   - Roles: testuser=Member of sec-org, testuser2=Owner of sec-org + Coordinator of secret-circle
//   - 3 Projects: root-project (on sec-org), private-project (on private-circle), secret-project (on secret-circle)
func seedTestData() error {
	dgc, cancel := newDgraphClient()
	defer cancel()

	hashedPw1 := HashPassword(testutil.TestPassword)
	hashedPw2 := HashPassword(testutil.TestPassword2)

	nquads := fmt.Sprintf(`
		# --- test-org: Public organisation for general tests ---
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

		# --- User 1: testuser (Owner of test-org, Member of sec-org) ---
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

		# --- User 2: testuser2 (Owner of sec-org, Coordinator of secret-circle) ---
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

		# --- test-org roles ---

		# Owner role for testuser in test-org
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

		# Coordinator role for testuser in test-org
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

		# --- test-org tension with blob and event ---

		# Tension (Open, Operational)
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

		# Blob (OnNode, source for org)
		_:blob <dgraph.type> "Blob" .
		_:blob <Blob.blob_type> "OnNode" .
		_:blob <Blob.tension> _:tension .
		_:blob <Post.createdBy> _:user1 .
		_:blob <Post.createdAt> "2026-01-01T00:00:00Z" .
		_:tension <Tension.blobs> _:blob .
		_:org <Node.source> _:blob .

		# Event (Created)
		_:event <dgraph.type> "Event" .
		_:event <Event.event_type> "Created" .
		_:event <Event.tension> _:tension .
		_:event <Post.createdBy> _:user1 .
		_:event <Post.createdAt> "2026-01-01T00:00:00Z" .
		_:tension <Tension.history> _:event .

		# --- sec-org: Private organisation for security/visibility tests ---
		_:secorg <dgraph.type> "Node" .
		_:secorg <Node.nameid> "sec-org" .
		_:secorg <Node.rootnameid> "sec-org" .
		_:secorg <Node.name> "Security Org" .
		_:secorg <Node.about> "Organisation for security tests" .
		_:secorg <Node.isRoot> "true" .
		_:secorg <Node.type_> "Circle" .
		_:secorg <Node.visibility> "Private" .
		_:secorg <Node.mode> "Coordinated" .
		_:secorg <Node.rights> "0" .
		_:secorg <Node.isArchived> "false" .
		_:secorg <Node.createdAt> "2026-01-01T00:00:00Z" .
		_:secorg <Node.createdBy> _:user2 .
		_:secorg <Node.userCanJoin> "false" .
		_:secorg <Node.guestCanCreateTension> "false" .

		# --- sec-org roles ---

		# Owner role for testuser2 in sec-org
		_:secorg_owner <dgraph.type> "Node" .
		_:secorg_owner <Node.nameid> "sec-org##@testuser2" .
		_:secorg_owner <Node.rootnameid> "sec-org" .
		_:secorg_owner <Node.name> "testuser2" .
		_:secorg_owner <Node.isRoot> "false" .
		_:secorg_owner <Node.type_> "Role" .
		_:secorg_owner <Node.role_type> "Owner" .
		_:secorg_owner <Node.visibility> "Private" .
		_:secorg_owner <Node.mode> "Coordinated" .
		_:secorg_owner <Node.rights> "0" .
		_:secorg_owner <Node.isArchived> "false" .
		_:secorg_owner <Node.parent> _:secorg .
		_:secorg_owner <Node.first_link> _:user2 .
		_:secorg_owner <Node.createdBy> _:user2 .
		_:secorg_owner <Node.createdAt> "2026-01-01T00:00:00Z" .
		_:secorg <Node.children> _:secorg_owner .
		_:user2 <User.roles> _:secorg_owner .

		# Member role for testuser in sec-org (regular member, no coordo)
		_:secorg_member <dgraph.type> "Node" .
		_:secorg_member <Node.nameid> "sec-org##@testuser" .
		_:secorg_member <Node.rootnameid> "sec-org" .
		_:secorg_member <Node.name> "testuser" .
		_:secorg_member <Node.isRoot> "false" .
		_:secorg_member <Node.type_> "Role" .
		_:secorg_member <Node.role_type> "Member" .
		_:secorg_member <Node.visibility> "Private" .
		_:secorg_member <Node.mode> "Coordinated" .
		_:secorg_member <Node.rights> "0" .
		_:secorg_member <Node.isArchived> "false" .
		_:secorg_member <Node.parent> _:secorg .
		_:secorg_member <Node.first_link> _:user1 .
		_:secorg_member <Node.createdBy> _:user2 .
		_:secorg_member <Node.createdAt> "2026-01-01T00:00:00Z" .
		_:secorg <Node.children> _:secorg_member .
		_:user1 <User.roles> _:secorg_member .

		# --- sec-org sub-circles ---

		# Private sub-circle (visible to org members)
		_:secorg_private <dgraph.type> "Node" .
		_:secorg_private <Node.nameid> "sec-org#private-circle" .
		_:secorg_private <Node.rootnameid> "sec-org" .
		_:secorg_private <Node.name> "Private Circle" .
		_:secorg_private <Node.isRoot> "false" .
		_:secorg_private <Node.type_> "Circle" .
		_:secorg_private <Node.visibility> "Private" .
		_:secorg_private <Node.mode> "Coordinated" .
		_:secorg_private <Node.rights> "0" .
		_:secorg_private <Node.isArchived> "false" .
		_:secorg_private <Node.parent> _:secorg .
		_:secorg_private <Node.createdBy> _:user2 .
		_:secorg_private <Node.createdAt> "2026-01-01T00:00:00Z" .
		_:secorg <Node.children> _:secorg_private .

		# Secret sub-circle (visible only to role holders in this circle)
		_:secorg_secret <dgraph.type> "Node" .
		_:secorg_secret <Node.nameid> "sec-org#secret-circle" .
		_:secorg_secret <Node.rootnameid> "sec-org" .
		_:secorg_secret <Node.name> "Secret Circle" .
		_:secorg_secret <Node.isRoot> "false" .
		_:secorg_secret <Node.type_> "Circle" .
		_:secorg_secret <Node.visibility> "Secret" .
		_:secorg_secret <Node.mode> "Coordinated" .
		_:secorg_secret <Node.rights> "0" .
		_:secorg_secret <Node.isArchived> "false" .
		_:secorg_secret <Node.parent> _:secorg .
		_:secorg_secret <Node.createdBy> _:user2 .
		_:secorg_secret <Node.createdAt> "2026-01-01T00:00:00Z" .
		_:secorg <Node.children> _:secorg_secret .

		# Coordinator role for testuser2 in secret circle (so they can see it)
		_:secret_coordo <dgraph.type> "Node" .
		_:secret_coordo <Node.nameid> "sec-org#secret-circle#:coordo" .
		_:secret_coordo <Node.rootnameid> "sec-org" .
		_:secret_coordo <Node.name> "Secret Coordinator" .
		_:secret_coordo <Node.isRoot> "false" .
		_:secret_coordo <Node.type_> "Role" .
		_:secret_coordo <Node.role_type> "Coordinator" .
		_:secret_coordo <Node.visibility> "Secret" .
		_:secret_coordo <Node.mode> "Coordinated" .
		_:secret_coordo <Node.rights> "0" .
		_:secret_coordo <Node.isArchived> "false" .
		_:secret_coordo <Node.parent> _:secorg_secret .
		_:secret_coordo <Node.first_link> _:user2 .
		_:secret_coordo <Node.createdBy> _:user2 .
		_:secret_coordo <Node.createdAt> "2026-01-01T00:00:00Z" .
		_:secorg_secret <Node.children> _:secret_coordo .
		_:user2 <User.roles> _:secret_coordo .

		# --- Projects linked to sec-org nodes ---

		# Project on root circle (testuser can see, unauthenticated cannot)
		_:proj_root <dgraph.type> "Project" .
		_:proj_root <Project.nameid> "root-project" .
		_:proj_root <Project.rootnameid> "sec-org" .
		_:proj_root <Project.parentnameid> "sec-org" .
		_:proj_root <Project.name> "Root Project" .
		_:proj_root <Project.description> "Project on the root private circle" .
		_:proj_root <Project.status> "Open" .
		_:proj_root <Project.createdAt> "2026-01-01T00:00:00Z" .
		_:proj_root <Project.updatedAt> "2026-01-01T00:00:00Z" .
		_:proj_root <Project.createdBy> _:user2 .
		_:proj_root <Project.nodes> _:secorg .
		_:secorg <Node.projects> _:proj_root .

		# Project on private sub-circle (testuser can see as org member)
		_:proj_private <dgraph.type> "Project" .
		_:proj_private <Project.nameid> "private-project" .
		_:proj_private <Project.rootnameid> "sec-org" .
		_:proj_private <Project.parentnameid> "sec-org#private-circle" .
		_:proj_private <Project.name> "Private Project" .
		_:proj_private <Project.description> "Project on the private sub-circle" .
		_:proj_private <Project.status> "Open" .
		_:proj_private <Project.createdAt> "2026-01-01T00:00:00Z" .
		_:proj_private <Project.updatedAt> "2026-01-01T00:00:00Z" .
		_:proj_private <Project.createdBy> _:user2 .
		_:proj_private <Project.nodes> _:secorg_private .
		_:secorg_private <Node.projects> _:proj_private .

		# Project on secret sub-circle (only testuser2 can see)
		_:proj_secret <dgraph.type> "Project" .
		_:proj_secret <Project.nameid> "secret-project" .
		_:proj_secret <Project.rootnameid> "sec-org" .
		_:proj_secret <Project.parentnameid> "sec-org#secret-circle" .
		_:proj_secret <Project.name> "Secret Project" .
		_:proj_secret <Project.description> "Project on the secret sub-circle" .
		_:proj_secret <Project.status> "Open" .
		_:proj_secret <Project.createdAt> "2026-01-01T00:00:00Z" .
		_:proj_secret <Project.updatedAt> "2026-01-01T00:00:00Z" .
		_:proj_secret <Project.createdBy> _:user2 .
		_:proj_secret <Project.nodes> _:secorg_secret .
		_:secorg_secret <Node.projects> _:proj_secret .
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
