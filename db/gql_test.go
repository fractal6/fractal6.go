/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2026 Fractale Co
 *
 * This file is part of Fractale.
 */

// Characterization tests of the GQL API (gql.go): they pin down the exact
// Graphql request sent to Dgraph and how its answer is decoded, for every
// entrypoint of the Query/Add/Update/Delete family.

package db

import (
	"testing"

	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/internal/testutil"
)

// fakeDgraph returns a client bound to a fake Dgraph endpoint answering
// `resp`, along with the request it received (filled after the call).
func fakeDgraph(t *testing.T, resp string) (*Dgraph, *testutil.GqlReq) {
	t.Helper()
	SetTestJWTKeys()
	url, got := testutil.FakeGqlServer(t, resp)
	return NewDgraph(url, ""), got
}

var testUctx = model.UserCtx{Username: "testuser"}

func TestAdd(t *testing.T) {
	dg, got := fakeDgraph(t, `{"data":{"addNode":{"node":[{"id":"0x1"}]}}}`)

	id, err := dg.Add(testUctx, "node", model.AddNodeInput{Name: "n1", Nameid: "o#n1"})
	if err != nil {
		t.Fatalf("Add returned error: %v", err)
	}
	if id != "0x1" {
		t.Errorf("id = %q, want 0x1", id)
	}
	testutil.CheckRequest(t, got, `mutation addNode($input:[AddNodeInput!]!) { addNode(input: $input) { node { id } } }`, `{"input":[{"createdAt":"","isArchived":false,"isRoot":false,"mode":"","name":"n1","nameid":"o#n1","rights":0,"rootnameid":"","type_":"","visibility":""}]}`)
}

func TestAddMany(t *testing.T) {
	dg, got := fakeDgraph(t, `{"data":{"addLabel":{"label":[{"id":"0x1"},{"id":"0x2"}]}}}`)

	ids, err := dg.AddMany(testUctx, "label", []model.AddLabelInput{{Name: "l1"}, {Name: "l2"}})
	if err != nil {
		t.Fatalf("AddMany returned error: %v", err)
	}
	if len(ids) != 2 || ids[0] != "0x1" || ids[1] != "0x2" {
		t.Errorf("ids = %v, want [0x1 0x2]", ids)
	}
	testutil.CheckRequest(t, got, `mutation addLabel($input:[AddLabelInput!]!) { addLabel(input: $input) { label { id } } }`, `{"input":[{"name":"l1","rootnameid":""},{"name":"l2","rootnameid":""}]}`)
}

func TestUpdate(t *testing.T) {
	dg, got := fakeDgraph(t, `{"data":{"updateUser":{"user":[{"id":"0x1"}]}}}`)

	username := "testuser"
	nameid := "o#@testuser"
	input := model.UpdateUserInput{
		Filter: &model.UserFilter{Username: &model.StringHashFilterStringRegExpFilter{Eq: &username}},
		Set:    &model.UserPatch{Roles: []*model.NodeRef{{Nameid: &nameid}}},
	}
	if err := dg.Update(testUctx, "user", input); err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	testutil.CheckRequest(t, got, `mutation updateUser($input:UpdateUserInput!) { updateUser(input: $input) { user { id } } }`, `{"input":{"filter":{"username":{"eq":"testuser"}},"set":{"roles":[{"nameid":"o#@testuser"}]}}}`)
}

func TestDelete(t *testing.T) {
	dg, got := fakeDgraph(t, `{"data":{"deleteLabel":{"label":[{"id":"0x1"}]}}}`)

	if err := dg.Delete(testUctx, "label", model.LabelFilter{ID: []string{"0x1"}}); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	testutil.CheckRequest(t, got, `mutation deleteLabel($input:LabelFilter!) { deleteLabel(filter: $input) { label { id } } }`, `{"input":{"id":["0x1"]}}`)
}

func TestQuery(t *testing.T) {
	dg, got := fakeDgraph(t, `{"data":{"queryNode":[{"id":"0x1","nameid":"o#n1"}]}}`)

	res, err := dg.Query(testUctx, "node", "nameid", []string{"o#n1", "o#n2"}, "id nameid")
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if len(res) != 1 || res[0]["id"] != "0x1" || res[0]["nameid"] != "o#n1" {
		t.Errorf("res = %v, want [{id:0x1 nameid:o#n1}]", res)
	}
	testutil.CheckRequest(t, got, `query queryNode($filter:NodeFilter) { queryNode(filter: $filter)  { id nameid } }`, `{"filter":{"nameid":{"in":["o#n1","o#n2"]}}}`)
}

// Query on "id" uses the uid list form, not the {in: […]} filter.
func TestQuery_ById(t *testing.T) {
	dg, got := fakeDgraph(t, `{"data":{"queryNode":[]}}`)

	if _, err := dg.Query(testUctx, "node", "id", []string{"0x1"}, "id"); err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	testutil.CheckRequest(t, got, `query queryNode($filter:NodeFilter) { queryNode(filter: $filter)  { id } }`, `{"filter":{"id":["0x1"]}}`)
}

func TestQueryGraph(t *testing.T) {
	dg, got := fakeDgraph(t, `{"data":{"queryTension":[{"id":"0x1"}]}}`)

	var data []model.Tension
	first := 10
	filter := model.TensionFilter{ID: []string{"0x1"}}
	asc := model.TensionOrderableCreatedAt
	order := model.TensionOrder{Asc: &asc}
	if err := dg.QueryGraph(testUctx, "tension", filter, order, &first, nil, "id", &data); err != nil {
		t.Fatalf("QueryGraph returned error: %v", err)
	}
	if len(data) != 1 || data[0].ID != "0x1" {
		t.Errorf("data = %v, want one tension 0x1", data)
	}
	testutil.CheckRequest(t, got, `query queryTension($filter:TensionFilter, $order:TensionOrder, $first:Int) { queryTension(filter: $filter, order: $order, first: $first)  { id } }`, `{"filter":{"id":["0x1"]},"first":10,"order":{"asc":"createdAt"}}`)
}

// The `cascade_directive` marker in the payload graph turns into @cascade.
func TestQueryGraph_CascadeDirective(t *testing.T) {
	dg, got := fakeDgraph(t, `{"data":{"queryNode":[]}}`)

	var data []model.Node
	if err := dg.QueryGraph(testUctx, "node", nil, nil, nil, nil, "id cascade_directive nameid", &data); err != nil {
		t.Fatalf("QueryGraph returned error: %v", err)
	}
	testutil.CheckRequest(t, got, `query queryNode { queryNode @cascade { id nameid } }`, `{}`)
}

// Only the arguments actually given are declared: a vertex without an
// <Vertex>Order input type (e.g ProjectField) must not get an $order.
func TestQueryGraph_OmitsNilArguments(t *testing.T) {
	dg, got := fakeDgraph(t, `{"data":{"queryProjectField":[]}}`)

	var data []model.ProjectField
	has := model.ProjectFieldHasFilterIsVisible
	filter := model.ProjectFieldFilter{Has: []*model.ProjectFieldHasFilter{&has}}
	if err := dg.QueryGraph(testUctx, "projectField", filter, nil, nil, nil, "id", &data); err != nil {
		t.Fatalf("QueryGraph returned error: %v", err)
	}
	testutil.CheckRequest(t, got, `query queryProjectField($filter:ProjectFieldFilter) { queryProjectField(filter: $filter)  { id } }`, `{"filter":{"has":["isVisible"]}}`)
}

func TestAddGraph_Upsert(t *testing.T) {
	dg, got := fakeDgraph(t, `{"data":{"addLabel":{"label":[{"id":"0x1","name":"l1"}]}}}`)

	upsert := true
	var data model.AddLabelPayload
	input := []*model.AddLabelInput{{Name: "l1"}}
	if err := dg.AddGraph(testUctx, "label", input, &upsert, "label { id name }", &data); err != nil {
		t.Fatalf("AddGraph returned error: %v", err)
	}
	if len(data.Label) != 1 || data.Label[0].Name != "l1" {
		t.Errorf("data = %v, want one label l1", data.Label)
	}
	testutil.CheckRequest(t, got, `mutation addLabel($input:[AddLabelInput!]!) { addLabel(input: $input, upsert: true) { label { id name } } }`, `{"input":[{"name":"l1","rootnameid":""}]}`)
}

// A single (non slice) input is still sent as a GQL list: `add` takes a list.
func TestAddGraph_SingleInputIsListed(t *testing.T) {
	dg, got := fakeDgraph(t, `{"data":{"addLabel":{"label":[{"id":"0x1"}]}}}`)

	var data model.AddLabelPayload
	if err := dg.AddGraph(testUctx, "label", model.AddLabelInput{Name: "l1"}, nil, "label { id }", &data); err != nil {
		t.Fatalf("AddGraph returned error: %v", err)
	}
	testutil.CheckRequest(t, got,
		`mutation addLabel($input:[AddLabelInput!]!) { addLabel(input: $input) { label { id } } }`,
		`{"input":[{"name":"l1","rootnameid":""}]}`)
}

func TestUpdateGraph(t *testing.T) {
	dg, got := fakeDgraph(t, `{"data":{"updateLabel":{"label":[{"id":"0x1"}]}}}`)

	var data model.UpdateLabelPayload
	name := "l2"
	input := model.UpdateLabelInput{
		Filter: &model.LabelFilter{ID: []string{"0x1"}},
		Set:    &model.LabelPatch{Name: &name},
	}
	if err := dg.UpdateGraph(testUctx, "label", input, "label { id }", &data); err != nil {
		t.Fatalf("UpdateGraph returned error: %v", err)
	}
	testutil.CheckRequest(t, got, `mutation updateLabel($input:UpdateLabelInput!) { updateLabel(input: $input) { label { id } } }`, `{"input":{"filter":{"id":["0x1"]},"set":{"name":"l2"}}}`)
}

func TestDeleteGraph(t *testing.T) {
	dg, got := fakeDgraph(t, `{"data":{"deleteLabel":{"numUids":1}}}`)

	var data model.DeleteLabelPayload
	if err := dg.DeleteGraph(testUctx, "label", model.LabelFilter{ID: []string{"0x1"}}, "numUids", &data); err != nil {
		t.Fatalf("DeleteGraph returned error: %v", err)
	}
	if data.NumUids == nil || *data.NumUids != 1 {
		t.Errorf("numUids = %v, want 1", data.NumUids)
	}
	testutil.CheckRequest(t, got, `mutation deleteLabel($input:LabelFilter!) { deleteLabel(filter: $input) { numUids } }`, `{"input":{"id":["0x1"]}}`)
}

// @auth rules can filter out the created vertex: AddMany still succeeds with
// no id, while Add — which promises one id — reports it.
func TestAdd_FilteredPayload(t *testing.T) {
	dg, _ := fakeDgraph(t, `{"data":{"addUserEvent":{"userEvent":[]}}}`)

	ids, err := dg.AddMany(testUctx, "userEvent", []model.AddUserEventInput{{}})
	if err != nil {
		t.Fatalf("AddMany returned error: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("ids = %v, want none", ids)
	}
	if _, err := dg.Add(testUctx, "userEvent", model.AddUserEventInput{}); err == nil {
		t.Error("Add: expected an error when no id is returned, got nil")
	}
}

// A missing payload means the @auth rules filtered the mutation out.
func TestMutations_EmptyPayloadIsUnauthorized(t *testing.T) {
	dg, _ := fakeDgraph(t, `{"data":{"addNode":null,"updateNode":null,"deleteNode":null}}`)

	if _, err := dg.Add(testUctx, "node", model.AddNodeInput{Name: "n1"}); err == nil {
		t.Error("Add: expected an unauthorized error, got nil")
	}
	if err := dg.Update(testUctx, "node", model.UpdateNodeInput{}); err == nil {
		t.Error("Update: expected an unauthorized error, got nil")
	}
	if err := dg.Delete(testUctx, "node", model.NodeFilter{}); err == nil {
		t.Error("Delete: expected an unauthorized error, got nil")
	}
}

// Dgraph errors are surfaced as GraphQLError.
func TestGqlErrorIsReturned(t *testing.T) {
	dg, _ := fakeDgraph(t, `{"errors":[{"message":"boom"}]}`)

	_, err := dg.Add(testUctx, "node", model.AddNodeInput{Name: "n1"})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if _, ok := err.(*GraphQLError); !ok {
		t.Errorf("error type = %T, want *GraphQLError", err)
	}
}
