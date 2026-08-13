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
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/internal/testutil"
)

func TestImportOrga_PersistsGovernanceLinks(t *testing.T) {
	jwtCookie := loginAs(testutil.TestUser, testutil.TestPassword)
	const (
		orgNameid  = "integration-import-org"
		roleNameid = orgNameid + "##developer"
	)
	csv := `Circle ID,Circle,Role ID,Role,IsCircle,Purpose
,,root-role,Spreadsheet Root,TRUE,Root mandate
root-ref,Spreadsheet Root,role-1,Developer,FALSE,Role mandate
`

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range map[string]string{
		"name":   "Integration Import Org",
		"nameid": orgNameid,
		"format": "holaspirit",
	} {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("writing multipart field %s: %v", key, err)
		}
	}
	part, err := writer.CreateFormFile("file", "Circles & Roles.csv")
	if err != nil {
		t.Fatalf("creating spreadsheet part: %v", err)
	}
	if _, err := part.Write([]byte(csv)); err != nil {
		t.Fatalf("writing spreadsheet: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("closing multipart body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/auth/createorga/spreadsheet", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.AddCookie(jwtCookie)
	rr := httptest.NewRecorder()
	testRouter.ServeHTTP(rr, req)
	requireStatus(t, rr, http.StatusOK)

	requireGovernanceLink(t, orgNameid)
	sourceID, _ := requireGovernanceLink(t, roleNameid)
	fragmentValue, err := db.GetDB().GetByUid(sourceID, "Blob.node", "NodeFragment.nameid NodeFragment.name NodeFragment.type_ NodeFragment.role_type")
	if err != nil {
		t.Fatalf("querying imported node fragment: %v", err)
	}
	fragment, ok := fragmentValue.(map[string]any)
	if !ok {
		t.Fatalf("imported node fragment has type %T, want map[string]any", fragmentValue)
	}
	if fragment["nameid"] != "developer" || fragment["name"] != "Developer" ||
		fragment["type_"] != "Role" || fragment["role_type"] != "Peer" {
		t.Fatalf("imported node fragment is incomplete: %#v", fragment)
	}
}

// requireGovernanceLink verifies the persisted Node -> source Blob -> Tension -> governed Node chain.
func requireGovernanceLink(t *testing.T, nodeNameid string) (sourceID, tensionID string) {
	t.Helper()
	nodeValue, err := db.GetDB().GetByEq("Node.nameid", nodeNameid, "uid")
	if err != nil {
		t.Fatalf("querying node %s: %v", nodeNameid, err)
	}
	nodeID, ok := nodeValue.(string)
	if !ok {
		t.Fatalf("node %s uid has type %T", nodeNameid, nodeValue)
	}
	sourceValue, err := db.GetDB().GetByUid(nodeID, "Node.source", "uid")
	if err != nil {
		t.Fatalf("querying node %s source: %v", nodeNameid, err)
	}
	sourceID, ok = sourceValue.(string)
	if !ok {
		t.Fatalf("node %s source uid has type %T", nodeNameid, sourceValue)
	}
	tensionValue, err := db.GetDB().GetByUid(sourceID, "Blob.tension", "uid")
	if err != nil {
		t.Fatalf("querying node %s governance tension: %v", nodeNameid, err)
	}
	tensionID, ok = tensionValue.(string)
	if !ok {
		t.Fatalf("node %s tension uid has type %T", nodeNameid, tensionValue)
	}
	governedValue, err := db.GetDB().GetByUid(tensionID, "Tension.governed_node", "uid")
	if err != nil {
		t.Fatalf("querying node %s governed relation: %v", nodeNameid, err)
	}
	if governedValue != nodeID {
		t.Fatalf("tension %s governs %v, want node %s", tensionID, governedValue, nodeID)
	}
	return sourceID, tensionID
}
