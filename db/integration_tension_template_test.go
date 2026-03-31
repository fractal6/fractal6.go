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
	"testing"

	. "fractale/fractal6.go/db"
	"fractale/fractal6.go/graph/model"
)

// buildTestUserCtx builds a UserCtx for testuser by fetching roles from DB.
func buildTestUserCtx(t *testing.T) model.UserCtx {
	t.Helper()
	roles, err := Meta[*model.Node]("getUserRoles", map[string]string{"userid": "testuser"})
	if err != nil {
		t.Fatalf("failed to get user roles: %v", err)
	}
	return model.UserCtx{
		Username: "testuser",
		Rights:   model.UserRights{Type: model.UserTypeRegular},
		Roles:    roles,
	}
}

func TestTensionTemplateCRUD_Integration(t *testing.T) {
	uctx := buildTestUserCtx(t)
	templateName := "test-bug-report-template"
	rootnameid := "test-org"
	nodeName := "test-org"

	// --- Add ---
	input := []*model.AddTensionTemplateInput{{
		Rootnameid:  rootnameid,
		Name:        templateName,
		Nodes:       []*model.NodeRef{{Nameid: &nodeName}},
		IsRecursive: true,
		Title:       "Bug Report: ",
		Comment:     "## Steps to reproduce\n\n## Expected behavior\n\n## Actual behavior",
		Type:        model.TensionTypeOperational,
	}}
	var addResult model.AddTensionTemplatePayload
	err := GetDB().AddExtra(uctx, "tensionTemplate", input, nil,
		"tensionTemplate { id name rootnameid is_recursive title comment type_ }", &addResult)
	if err != nil {
		t.Fatalf("AddExtra(tensionTemplate) returned error: %v", err)
	}
	if len(addResult.TensionTemplate) == 0 {
		t.Fatal("AddExtra returned no tension templates")
	}
	templateID := addResult.TensionTemplate[0].ID
	if templateID == "" {
		t.Fatal("AddExtra returned empty ID")
	}
	if addResult.TensionTemplate[0].Name != templateName {
		t.Errorf("name = %q, want %q", addResult.TensionTemplate[0].Name, templateName)
	}
	if !addResult.TensionTemplate[0].IsRecursive {
		t.Error("is_recursive = false, want true")
	}
	t.Logf("Created TensionTemplate with ID: %s", templateID)

	// --- Query (verify exists via DQL) ---
	val, err := GetDB().GetFieldByEq("TensionTemplate.name", templateName, "TensionTemplate.title")
	if err != nil {
		t.Fatalf("GetFieldByEq returned error: %v", err)
	}
	if title, ok := val.(string); !ok || title != "Bug Report: " {
		t.Errorf("TensionTemplate.title = %v, want %q", val, "Bug Report: ")
	}

	// --- Delete ---
	filter := model.TensionTemplateFilter{ID: []string{templateID}}
	var delResult model.DeleteTensionTemplatePayload
	err = GetDB().DeleteExtra(uctx, "tensionTemplate", filter,
		"tensionTemplate { id name } msg numUids", &delResult)
	if err != nil {
		t.Fatalf("DeleteExtra(tensionTemplate) returned error: %v", err)
	}
	if delResult.NumUids == nil || *delResult.NumUids == 0 {
		t.Error("DeleteExtra returned numUids=0, expected at least 1")
	}

	// --- Verify deleted ---
	val, err = GetDB().GetFieldByEq("TensionTemplate.name", templateName, "TensionTemplate.title")
	if err != nil {
		t.Fatalf("GetFieldByEq after delete returned error: %v", err)
	}
	if val != nil {
		t.Errorf("expected nil after delete, got %v", val)
	}
}
