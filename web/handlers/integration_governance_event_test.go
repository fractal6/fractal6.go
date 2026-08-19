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
	"strconv"
	"strings"
	"testing"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/internal/testutil"
)

const governanceMutation = `
mutation UpdateTension($tension: UpdateTensionInput!) {
  updateTension(input: $tension) { numUids }
}`

type governanceFixture struct {
	marker     string
	tensionID  string
	blobID     string
	nodeNameid string
}

type graphqlResult struct {
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func cleanupGovernanceFixture(marker, nodeNameid string) {
	q := `query {
		fixture(func: eq(Post.message, ` + strconv.Quote(marker) + `)) {
			t as uid
			Tension.blobs {
				b as uid
				Blob.node { f as uid }
			}
			Tension.history { e as uid }
		}
		n as var(func: eq(Node.nameid, ` + strconv.Quote(nodeNameid) + `))
		r as var(func: eq(Node.nameid, "test-org"))
	}`
	_, _ = db.GetDB().Gamma(db.QueryMut{Q: q, M: []db.X{{D: `
		uid(r) <Node.children> uid(n) .
		uid(t) * * .
		uid(b) * * .
		uid(f) * * .
		uid(e) * * .
		uid(n) * * .
	`}}}, nil)
}

func createGovernanceFixture(t *testing.T, key, nodeType string, name, roleType *string, withBlob bool) governanceFixture {
	t.Helper()
	marker := "governance-event-matrix:" + key
	localNameid := "matrix-" + key
	nodeNameid := "test-org#" + localNameid
	if nodeType != "Circle" {
		nodeNameid = "test-org##" + localNameid
	}
	cleanupGovernanceFixture(marker, nodeNameid)
	t.Cleanup(func() { cleanupGovernanceFixture(marker, nodeNameid) })

	set := `_:t <dgraph.type> "Tension" .
		_:t <Tension.title> ` + strconv.Quote("Governance matrix "+key) + ` .
		_:t <Tension.status> "Open" .
		_:t <Tension.type_> "Governance" .
		_:t <Tension.emitter> uid(root) .
		_:t <Tension.emitterid> "test-org" .
		_:t <Tension.receiver> uid(root) .
		_:t <Tension.receiverid> "test-org" .
		_:t <Post.createdBy> uid(author) .
		_:t <Post.createdAt> "2027-01-01T00:00:00Z" .
		_:t <Post.message> ` + strconv.Quote(marker) + ` .`
	if withBlob {
		set += `
		_:t <Tension.blobs> _:b .
		_:b <dgraph.type> "Blob" .
		_:b <Blob.tension> _:t .
		_:b <Blob.node> _:f .
		_:b <Post.createdBy> uid(author) .
		_:b <Post.createdAt> "2027-01-01T00:00:01Z" .
		_:f <dgraph.type> "NodeFragment" .
		_:f <NodeFragment.nameid> ` + strconv.Quote(localNameid) + ` .`
		if nodeType != "" {
			set += `
		_:f <NodeFragment.type_> ` + strconv.Quote(nodeType) + ` .`
		}
		if name != nil {
			set += `
		_:f <NodeFragment.name> ` + strconv.Quote(*name) + ` .`
		}
		if roleType != nil {
			set += `
		_:f <NodeFragment.role_type> ` + strconv.Quote(*roleType) + ` .`
		}
	}

	qm := db.QueryMut{
		Q: `query {
			root as var(func: eq(Node.nameid, "test-org"))
			author as var(func: eq(User.username, "` + testutil.TestUser + `"))
		}`,
		M: []db.X{{S: set}},
	}
	if _, err := db.GetDB().Gamma(qm, nil); err != nil {
		t.Fatalf("create governance fixture %s: %v", key, err)
	}

	tids, err := db.GetDB().GetIDs("Post.message", marker, nil, nil)
	if err != nil || len(tids) != 1 {
		t.Fatalf("resolve governance fixture %s: ids=%v err=%v", key, tids, err)
	}
	fixture := governanceFixture{marker: marker, tensionID: tids[0], nodeNameid: nodeNameid}
	if withBlob {
		value, err := db.GetDB().GetByUid(fixture.tensionID, "Tension.blobs", "uid")
		if err != nil {
			t.Fatalf("resolve governance blob %s: %v", key, err)
		}
		ids, ok := value.([]any)
		if !ok || len(ids) != 1 {
			t.Fatalf("governance blob %s = %T(%v), want one blob", key, value, value)
		}
		fixture.blobID, ok = ids[0].(string)
		if !ok {
			t.Fatalf("governance blob UID %s = %T", key, ids[0])
		}
	}
	return fixture
}

func addGovernanceBlob(t *testing.T, fixture governanceFixture, nodeType, name, roleType string) string {
	t.Helper()
	qm := db.QueryMut{
		Q: `query {
			t as var(func: uid(` + fixture.tensionID + `))
			author as var(func: eq(User.username, "` + testutil.TestUser + `"))
		}`,
		M: []db.X{{S: `
			uid(t) <Tension.blobs> _:b .
			_:b <dgraph.type> "Blob" .
			_:b <Blob.tension> uid(t) .
			_:b <Blob.node> _:f .
			_:b <Post.createdBy> uid(author) .
			_:b <Post.createdAt> "2027-01-02T00:00:00Z" .
			_:b <Post.message> ` + strconv.Quote(fixture.marker+":second-blob") + ` .
			_:f <dgraph.type> "NodeFragment" .
			_:f <NodeFragment.nameid> "matrix-role" .
			_:f <NodeFragment.name> ` + strconv.Quote(name) + ` .
			_:f <NodeFragment.type_> ` + strconv.Quote(nodeType) + ` .
			_:f <NodeFragment.role_type> ` + strconv.Quote(roleType) + ` .
		`}},
	}
	if _, err := db.GetDB().Gamma(qm, nil); err != nil {
		t.Fatalf("add second governance blob: %v", err)
	}
	ids, err := db.GetDB().GetIDs("Post.message", fixture.marker+":second-blob", nil, nil)
	if err != nil || len(ids) != 1 {
		t.Fatalf("resolve second governance blob: ids=%v err=%v", ids, err)
	}
	return ids[0]
}

func runTensionEvent(t *testing.T, cookie *http.Cookie, tensionID, eventType string, set map[string]any) graphqlResult {
	t.Helper()
	if set == nil {
		set = map[string]any{}
	}
	event := map[string]any{"event_type": eventType}
	if value, ok := set["type_"]; ok && eventType == "TypeUpdated" {
		event["new"] = value
	}
	set["history"] = []any{event}
	response := doRequest("POST", "/api", map[string]any{
		"query": governanceMutation,
		"variables": map[string]any{
			"tension": map[string]any{
				"filter": map[string]any{"id": []string{tensionID}},
				"set":    set,
			},
		},
	}, cookie)
	if response.Code != http.StatusOK {
		t.Fatalf("GraphQL %s returned HTTP %d: %s", eventType, response.Code, response.Body.String())
	}
	var result graphqlResult
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode GraphQL %s response: %v (%s)", eventType, err, response.Body.String())
	}
	return result
}

func requireGraphQLSuccess(t *testing.T, result graphqlResult) {
	t.Helper()
	if len(result.Errors) != 0 {
		t.Fatalf("unexpected GraphQL errors: %+v", result.Errors)
	}
}

func requireGraphQLError(t *testing.T, result graphqlResult, contains string) {
	t.Helper()
	if len(result.Errors) == 0 {
		t.Fatalf("expected GraphQL error containing %q", contains)
	}
	for _, gqlErr := range result.Errors {
		if strings.Contains(gqlErr.Message, contains) {
			return
		}
	}
	messages, _ := json.Marshal(result.Errors)
	t.Fatalf("GraphQL errors %s do not contain %q", messages, contains)
}

func governedNodeID(t *testing.T, tensionID string) string {
	t.Helper()
	value, err := db.GetDB().GetByUid(tensionID, "Tension.governed_node", "uid")
	if err != nil {
		t.Fatalf("query governed node: %v", err)
	}
	uid, ok := value.(string)
	if !ok || uid == "" {
		t.Fatalf("governed node = %T(%v), want UID", value, value)
	}
	return uid
}

func requireGovernanceState(t *testing.T, nodeID string, archived bool) {
	t.Helper()
	state, err := db.GetDB().GetByUid(nodeID, "Node.isArchived")
	if err != nil || state != archived {
		t.Fatalf("Node.isArchived = %v, want %v (err=%v)", state, archived, err)
	}
}

func TestGovernanceEventsThroughGraphQL(t *testing.T) {
	cookie := loginAs(testutil.TestUser, testutil.TestPassword)
	peer := "Peer"

	t.Run("role publication and lifecycle", func(t *testing.T) {
		name := "Matrix Role"
		fixture := createGovernanceFixture(t, "role", "Role", &name, &peer, true)

		requireGraphQLSuccess(t, runTensionEvent(t, cookie, fixture.tensionID, "BlobPushed", nil))
		nodeID := governedNodeID(t, fixture.tensionID)
		requireGovernanceState(t, nodeID, false)
		if source, _ := db.GetDB().GetByUid(nodeID, "Node.source", "uid"); source != fixture.blobID {
			t.Fatalf("first publication source = %v, want %s", source, fixture.blobID)
		}
		if kind, _ := db.GetDB().GetByUid(nodeID, "Node.type_"); kind != "Role" {
			t.Fatalf("first publication kind = %v, want Role", kind)
		}

		secondBlobID := addGovernanceBlob(t, fixture, "Role", "Matrix Role Updated", peer)
		requireGraphQLSuccess(t, runTensionEvent(t, cookie, fixture.tensionID, "BlobPushed", nil))
		if current := governedNodeID(t, fixture.tensionID); current != nodeID {
			t.Fatalf("second publication governed node = %s, want %s", current, nodeID)
		}
		if ids, err := db.GetDB().GetIDs("Node.nameid", fixture.nodeNameid, nil, nil); err != nil || len(ids) != 1 {
			t.Fatalf("published node count = %d, want 1 (err=%v)", len(ids), err)
		}
		if got, _ := db.GetDB().GetByUid(nodeID, "Node.name"); got != "Matrix Role Updated" {
			t.Fatalf("second publication name = %v", got)
		}
		if source, _ := db.GetDB().GetByUid(nodeID, "Node.source", "uid"); source != secondBlobID {
			t.Fatalf("second publication source = %v, want %s", source, secondBlobID)
		}

		requireGraphQLSuccess(t, runTensionEvent(t, cookie, fixture.tensionID, "BlobArchived", nil))
		requireGovernanceState(t, nodeID, true)
		requireGraphQLError(t, runTensionEvent(t, cookie, fixture.tensionID, "BlobArchived", nil), "already archived")
		requireGraphQLError(t, runTensionEvent(t, cookie, fixture.tensionID, "BlobPushed", nil), "cannot publish an archived node")

		requireGraphQLSuccess(t, runTensionEvent(t, cookie, fixture.tensionID, "BlobUnarchived", nil))
		requireGovernanceState(t, nodeID, false)
		requireGraphQLError(t, runTensionEvent(t, cookie, fixture.tensionID, "BlobUnarchived", nil), "not archived")

		fragmentID, err := db.GetDB().GetByUid(secondBlobID, "Blob.node", "uid")
		if err != nil {
			t.Fatalf("resolve latest fragment: %v", err)
		}
		if err := db.GetDB().SetFieldById(fragmentID.(string), "NodeFragment.type_", "Circle"); err != nil {
			t.Fatalf("change fixture fragment kind: %v", err)
		}
		requireGraphQLError(t, runTensionEvent(t, cookie, fixture.tensionID, "BlobPushed", nil), "does not match governed node type")
	})

	t.Run("circle first publication", func(t *testing.T) {
		name := "Matrix Circle"
		fixture := createGovernanceFixture(t, "circle", "Circle", &name, nil, true)
		requireGraphQLSuccess(t, runTensionEvent(t, cookie, fixture.tensionID, "BlobPushed", nil))
		nodeID := governedNodeID(t, fixture.tensionID)
		requireGovernanceState(t, nodeID, false)
		if kind, _ := db.GetDB().GetByUid(nodeID, "Node.type_"); kind != "Circle" {
			t.Fatalf("circle publication kind = %v, want Circle", kind)
		}
	})

	t.Run("root archive is a flag", func(t *testing.T) {
		name := "Test Org"
		fixture := createGovernanceFixture(t, "root", "Circle", &name, nil, true)
		rootIDs, err := db.GetDB().GetIDs("Node.nameid", "test-org", nil, nil)
		if err != nil || len(rootIDs) != 1 {
			t.Fatalf("resolve root node: ids=%v err=%v", rootIDs, err)
		}
		rootID := rootIDs[0]
		qm := db.QueryMut{
			Q: `query {
				t as var(func: uid(` + fixture.tensionID + `))
				root as var(func: uid(` + rootID + `))
			}`,
			M: []db.X{{S: `uid(t) <Tension.governed_node> uid(root) .`}},
		}
		if _, err := db.GetDB().Gamma(qm, nil); err != nil {
			t.Fatalf("link root as governed node: %v", err)
		}
		t.Cleanup(func() { _ = db.GetDB().SetFieldById(rootID, "Node.isRootArchived", "false") })

		children, err := db.GetDB().GetChildren("test-org")
		if err != nil {
			t.Fatalf("get root children: %v", err)
		}

		requireGraphQLSuccess(t, runTensionEvent(t, cookie, fixture.tensionID, "BlobArchived", nil))
		if got, _ := db.GetDB().GetByUid(rootID, "Node.isRootArchived"); got != true {
			t.Fatalf("Node.isRootArchived = %v, want true", got)
		}
		requireGovernanceState(t, rootID, false)
		if got, err := db.GetDB().GetChildren("test-org"); err != nil || len(got) != len(children) {
			t.Fatalf("root children = %d, want %d (err=%v)", len(got), len(children), err)
		}
		requireGraphQLError(t, runTensionEvent(t, cookie, fixture.tensionID, "BlobArchived", nil), "already archived")

		requireGraphQLSuccess(t, runTensionEvent(t, cookie, fixture.tensionID, "BlobUnarchived", nil))
		if got, _ := db.GetDB().GetByUid(rootID, "Node.isRootArchived"); got != false {
			t.Fatalf("Node.isRootArchived = %v, want false", got)
		}
		requireGraphQLError(t, runTensionEvent(t, cookie, fixture.tensionID, "BlobUnarchived", nil), "not archived")
	})

	t.Run("null fragment fields return errors", func(t *testing.T) {
		name := "Malformed Role"
		cases := []struct {
			key, nodeType  string
			name, roleType *string
			want           string
		}{
			{key: "null-type", name: &name, roleType: &peer, want: "requires a valid type"},
			{key: "null-name", nodeType: "Role", roleType: &peer, want: "requires a name"},
			{key: "null-role-type", nodeType: "Role", name: &name, want: "requires a role_type"},
		}
		for _, test := range cases {
			t.Run(test.key, func(t *testing.T) {
				fixture := createGovernanceFixture(t, test.key, test.nodeType, test.name, test.roleType, true)
				requireGraphQLError(t, runTensionEvent(t, cookie, fixture.tensionID, "BlobPushed", nil), test.want)
			})
		}
	})

	t.Run("ordinary tension without node blob", func(t *testing.T) {
		fixture := createGovernanceFixture(t, "ordinary", "", nil, nil, false)
		requireGraphQLSuccess(t, runTensionEvent(t, cookie, fixture.tensionID, "TypeUpdated", map[string]any{
			"type_": "Operational",
		}))
		if got, _ := db.GetDB().GetByUid(fixture.tensionID, "Tension.type_"); got != "Operational" {
			t.Fatalf("ordinary tension type = %v, want Operational", got)
		}
	})
}
