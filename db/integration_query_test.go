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
	"encoding/json"
	"testing"
)

func TestCountHas_Integration(t *testing.T) {
	count := DB.CountHas("Node.nameid")
	if count < 1 {
		t.Errorf("CountHas(Node.nameid) = %d, want >= 1", count)
	}
	t.Logf("CountHas(Node.nameid) = %d", count)
}

func TestExists_Integration(t *testing.T) {
	found, err := DB.Exists("Node.nameid", "test-org#", nil)
	if err != nil {
		t.Fatalf("Exists returned error: %v", err)
	}
	if !found {
		t.Error("Exists(Node.nameid, test-org#) = false, want true")
	}
}

func TestExists_IntegrationNotFound(t *testing.T) {
	found, err := DB.Exists("Node.nameid", "nonexistent-org#", nil)
	if err != nil {
		t.Fatalf("Exists returned error: %v", err)
	}
	if found {
		t.Error("Exists(Node.nameid, nonexistent-org#) = true, want false")
	}
}

func TestGetFieldByEq_Integration(t *testing.T) {
	val, err := DB.GetFieldByEq("Node.nameid", "test-org#", "Node.name")
	if err != nil {
		t.Fatalf("GetFieldByEq returned error: %v", err)
	}
	name, ok := val.(string)
	if !ok {
		t.Fatalf("GetFieldByEq returned type %T, want string", val)
	}
	if name != "Test Org" {
		t.Errorf("GetFieldByEq(Node.name) = %q, want %q", name, "Test Org")
	}
}

func TestIsChild_Integration(t *testing.T) {
	isChild, err := DB.IsChild("test-org#", "test-org#@testuser")
	if err != nil {
		t.Fatalf("IsChild returned error: %v", err)
	}
	if !isChild {
		t.Error("IsChild(test-org#, test-org#@testuser) = false, want true")
	}
}

func TestIsChild_IntegrationNotChild(t *testing.T) {
	isChild, err := DB.IsChild("test-org#", "nonexistent#")
	if err != nil {
		t.Fatalf("IsChild returned error: %v", err)
	}
	if isChild {
		t.Error("IsChild(test-org#, nonexistent#) = true, want false")
	}
}

func TestGetChildren_Integration(t *testing.T) {
	children, err := DB.GetChildren("test-org#")
	if err != nil {
		t.Fatalf("GetChildren returned error: %v", err)
	}
	if len(children) < 2 {
		t.Errorf("GetChildren(test-org#) returned %d children, want >= 2", len(children))
	}
	t.Logf("GetChildren(test-org#) = %v", children)
}

func TestHasCoordos_Integration(t *testing.T) {
	has := DB.HasCoordos("test-org#")
	if !has {
		t.Error("HasCoordos(test-org#) = false, want true")
	}
}

func TestQueryDql_IntegrationRaw(t *testing.T) {
	res, err := DB.QueryDql("exists", map[string]string{
		"fieldName": "Node.nameid",
		"value":     "test-org#",
		"filter":    "",
	})
	if err != nil {
		t.Fatalf("QueryDql returned error: %v", err)
	}

	var resp DqlResp
	if err := json.Unmarshal(res.Json, &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if len(resp.All) == 0 {
		t.Error("QueryDql(exists) returned empty result")
	}
}

func TestMeta_IntegrationGetNodeHistory(t *testing.T) {
	results, err := DB.Meta("getNodeHistory", map[string]string{
		"nameid": "test-org#",
		"query":  "",
	})
	if err != nil {
		t.Fatalf("Meta(getNodeHistory) returned error: %v", err)
	}
	if len(results) < 1 {
		t.Errorf("Meta(getNodeHistory) returned %d results, want >= 1", len(results))
	}
	t.Logf("Meta(getNodeHistory) returned %d events", len(results))
}
