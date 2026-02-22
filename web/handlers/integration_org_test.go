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

package handlers_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/internal/testutil"
)

func TestCreateOrga_Success(t *testing.T) {
	// Login as testuser (seeded with canCreateRoot=true)
	jwtCookie := loginAs(testutil.TestUser, testutil.TestPassword)

	// Create a new organisation
	orgNameid := "integration-test-org"
	rr := doRequest("POST", "/auth/createorga", map[string]any{
		"name":    "Integration Test Org",
		"nameid":  orgNameid,
		"purpose": "Testing org creation",
	}, jwtCookie)
	requireStatus(t, rr, http.StatusOK)

	// Check response contains the org nameid
	var node model.Node
	if err := json.Unmarshal(rr.Body.Bytes(), &node); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if node.Nameid != orgNameid {
		t.Errorf("expected nameid %q, got %q", orgNameid, node.Nameid)
	}

	// Verify org Node exists in DB
	val, err := db.GetDB().GetFieldByEq("Node.nameid", orgNameid, "Node.name")
	if err != nil {
		t.Fatalf("failed to query org node: %v", err)
	}
	name, ok := val.(string)
	if !ok || name != "Integration Test Org" {
		t.Errorf("expected org name %q, got %v", "Integration Test Org", val)
	}
}

func TestCreateOrga_NoAuth(t *testing.T) {
	// Try to create org without JWT
	rr := doRequest("POST", "/auth/createorga", map[string]any{
		"name":   "No Auth Org",
		"nameid": "no-auth-org",
	})

	if rr.Code == http.StatusOK {
		t.Fatal("expected non-200 status for unauthenticated request")
	}
}

func TestCreateOrga_InvalidNameid(t *testing.T) {
	jwtCookie := loginAs(testutil.TestUser, testutil.TestPassword)

	// Nameid with # should be rejected
	rr := doRequest("POST", "/auth/createorga", map[string]any{
		"name":   "Bad Org",
		"nameid": "bad#org",
	}, jwtCookie)
	requireStatus(t, rr, http.StatusBadRequest)
}

func TestSetUserCanJoin_Success(t *testing.T) {
	// Login as testuser (coordinator of test-org)
	jwtCookie := loginAs(testutil.TestUser, testutil.TestPassword)

	// Set userCanJoin to false
	rr := doRequest("POST", "/auth/setusercanjoin", map[string]any{
		"nameid": "test-org",
		"val":    false,
	}, jwtCookie)
	requireStatus(t, rr, http.StatusOK)

	// Verify in DB
	val, err := db.GetDB().GetFieldByEq("Node.nameid", "test-org", "Node.userCanJoin")
	if err != nil {
		t.Fatalf("failed to query Node.userCanJoin: %v", err)
	}
	if boolVal, ok := val.(bool); !ok || boolVal != false {
		t.Errorf("expected userCanJoin=false, got %v", val)
	}

	// Restore to true
	rr = doRequest("POST", "/auth/setusercanjoin", map[string]any{
		"nameid": "test-org",
		"val":    true,
	}, jwtCookie)
	requireStatus(t, rr, http.StatusOK)
}

func TestSetGuestCanCreateTension_Success(t *testing.T) {
	// Login as testuser (coordinator of test-org)
	jwtCookie := loginAs(testutil.TestUser, testutil.TestPassword)

	// Set guestCanCreateTension to false
	rr := doRequest("POST", "/auth/setguestcancreatetension", map[string]any{
		"nameid": "test-org",
		"val":    false,
	}, jwtCookie)
	requireStatus(t, rr, http.StatusOK)

	// Verify in DB
	val, err := db.GetDB().GetFieldByEq("Node.nameid", "test-org", "Node.guestCanCreateTension")
	if err != nil {
		t.Fatalf("failed to query Node.guestCanCreateTension: %v", err)
	}
	if boolVal, ok := val.(bool); !ok || boolVal != false {
		t.Errorf("expected guestCanCreateTension=false, got %v", val)
	}

	// Restore to true
	rr = doRequest("POST", "/auth/setguestcancreatetension", map[string]any{
		"nameid": "test-org",
		"val":    true,
	}, jwtCookie)
	requireStatus(t, rr, http.StatusOK)
}

func TestSetLexicon_Success(t *testing.T) {
	// Login as testuser (owner of test-org)
	jwtCookie := loginAs(testutil.TestUser, testutil.TestPassword)

	// Set lexicon to a JSON string
	lexiconVal := `{"tension":"Issue","circle":"Team"}`
	rr := doRequest("POST", "/auth/setlexicon", map[string]any{
		"nameid": "test-org",
		"val":    lexiconVal,
	}, jwtCookie)
	requireStatus(t, rr, http.StatusOK)
	if rr.Body.String() != "true" {
		t.Errorf("expected response body %q, got %q", "true", rr.Body.String())
	}

	// Verify in DB
	val, err := db.GetDB().GetFieldByEq("Node.nameid", "test-org", "Node.lexicon")
	if err != nil {
		t.Fatalf("failed to query Node.lexicon: %v", err)
	}
	if strVal, ok := val.(string); !ok || strVal != lexiconVal {
		t.Errorf("expected lexicon=%q, got %v", lexiconVal, val)
	}

	// Restore to empty
	rr = doRequest("POST", "/auth/setlexicon", map[string]any{
		"nameid": "test-org",
		"val":    "",
	}, jwtCookie)
	requireStatus(t, rr, http.StatusOK)
}

func TestSetLexicon_NoAuth(t *testing.T) {
	// Try without JWT
	rr := doRequest("POST", "/auth/setlexicon", map[string]any{
		"nameid": "test-org",
		"val":    `{"foo":"bar"}`,
	})

	if rr.Code == http.StatusOK {
		t.Fatal("expected non-200 status for unauthenticated request")
	}
}

func TestSetUserCanJoin_NoAuth(t *testing.T) {
	// Try without JWT
	rr := doRequest("POST", "/auth/setusercanjoin", map[string]any{
		"nameid": "test-org",
		"val":    false,
	})

	if rr.Code == http.StatusOK {
		t.Fatal("expected non-200 status for unauthenticated request")
	}
}
