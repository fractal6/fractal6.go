/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2026 Fractale Co
 *
 * This file is part of Fractale.
 */

// Tests of the Dgraph-Gqlgen bridges (dgraph_resolver.go): what the resolver
// context (query name + preloaded payload graph + user) turns into on the
// wire, and how the answer is post-processed.

package graph

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/ast"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/internal/testutil"
)

// fakeResolver returns a resolver bound to a fake Dgraph endpoint answering
// `resp`, along with the request it received (filled after the call).
func fakeResolver(t *testing.T, resp string) (*Resolver, *testutil.GqlReq) {
	t.Helper()
	db.SetTestJWTKeys()
	url, got := testutil.FakeGqlServer(t, resp)
	return &Resolver{db: db.NewDgraph(url, "")}, got
}

// field builds an ast field with the object definition GetNestedPreloads needs.
func field(name string, children ...ast.Selection) *ast.Field {
	return &ast.Field{
		Name:             name,
		Alias:            name,
		ObjectDefinition: &ast.Definition{Kind: ast.Object, Name: name},
		SelectionSet:     ast.SelectionSet(children),
	}
}

// resolverCtx mimics what gqlgen puts in the context when resolving
// `queryName`: the field being resolved with its payload selections, plus an
// authenticated user.
func resolverCtx(queryName string, payload ...ast.Selection) context.Context {
	ctx := graphql.WithOperationContext(context.Background(), &graphql.OperationContext{
		Variables: map[string]any{},
	})
	ctx = graphql.WithFieldContext(ctx, &graphql.FieldContext{
		Field: graphql.CollectedField{
			Field:      &ast.Field{Name: queryName, Alias: queryName},
			Selections: ast.SelectionSet(payload),
		},
	})
	ctx = context.WithValue(ctx, "user_ctx", &model.UserCtx{Username: "testuser", Hit: 1})
	return context.WithValue(ctx, "iat", "2026-01-01T00:00:00Z")
}

// withRawQuery sets what the raw bridge reads off the operation context: the
// client query string and its variables.
func withRawQuery(ctx context.Context, rawQuery string, variables map[string]any) context.Context {
	oc := graphql.GetOperationContext(ctx)
	oc.RawQuery = rawQuery
	oc.Variables = variables
	return ctx
}

func TestDgraphQueryBridge(t *testing.T) {
	r, got := fakeResolver(t, `{"data":{"queryLabel":[{"id":"0x1","name":"l1"}]}}`)
	ctx := resolverCtx("queryLabel", field("id"), field("name"))

	var data []*model.Label
	first := 10
	filter := model.LabelFilter{ID: []string{"0x1"}}
	err := (&queryResolver{r}).DgraphQueryBridge(ctx, filter, nil, &first, nil, &data)
	if err != nil {
		t.Fatalf("DgraphQueryBridge returned error: %v", err)
	}
	if len(data) != 1 || data[0].Name != "l1" {
		t.Fatalf("data = %v, want one label l1", data)
	}
	testutil.CheckRequest(t, got,
		`query queryLabel($filter:LabelFilter, $first:Int) { queryLabel(filter: $filter, first: $first)  { id name } }`,
		`{"filter":{"id":["0x1"]},"first":10}`)
}

func TestDgraphAddBridge(t *testing.T) {
	r, got := fakeResolver(t, `{"data":{"addLabel":{"label":[{"id":"0x1"}]}}}`)
	ctx := resolverCtx("addLabel", field("label", field("id")))

	var data model.AddLabelPayload
	upsert := true
	input := []*model.AddLabelInput{{Name: "l1", Rootnameid: "test-org"}}
	err := (&mutationResolver{r}).DgraphAddBridge(ctx, input, &upsert, &data)
	if err != nil {
		t.Fatalf("DgraphAddBridge returned error: %v", err)
	}
	if len(data.Label) != 1 || data.Label[0].ID != "0x1" {
		t.Fatalf("data = %v, want one label 0x1", data.Label)
	}
	testutil.CheckRequest(t, got,
		`mutation addLabel($input:[AddLabelInput!]!) { addLabel(input: $input, upsert: true) { label { id } } }`,
		`{"input":[{"name":"l1","rootnameid":"test-org"}]}`)
}

func TestDgraphUpdateBridge(t *testing.T) {
	r, got := fakeResolver(t, `{"data":{"updateLabel":{"label":[{"id":"0x1"}]}}}`)
	ctx := resolverCtx("updateLabel", field("label", field("id")))

	var data model.UpdateLabelPayload
	name := "l2"
	input := model.UpdateLabelInput{
		Filter: &model.LabelFilter{ID: []string{"0x1"}},
		Set:    &model.LabelPatch{Name: &name},
	}
	err := (&mutationResolver{r}).DgraphUpdateBridge(ctx, input, &data)
	if err != nil {
		t.Fatalf("DgraphUpdateBridge returned error: %v", err)
	}
	testutil.CheckRequest(t, got,
		`mutation updateLabel($input:UpdateLabelInput!) { updateLabel(input: $input) { label { id } } }`,
		`{"input":{"filter":{"id":["0x1"]},"set":{"name":"l2"}}}`)
}

func TestDgraphDeleteBridge(t *testing.T) {
	r, got := fakeResolver(t, `{"data":{"deleteLabel":{"numUids":1}}}`)
	ctx := resolverCtx("deleteLabel", field("numUids"))

	var data model.DeleteLabelPayload
	err := (&mutationResolver{r}).DgraphDeleteBridge(ctx, model.LabelFilter{ID: []string{"0x1"}}, &data)
	if err != nil {
		t.Fatalf("DgraphDeleteBridge returned error: %v", err)
	}
	if data.NumUids == nil || *data.NumUids != 1 {
		t.Errorf("numUids = %v, want 1", data.NumUids)
	}
	testutil.CheckRequest(t, got,
		`mutation deleteLabel($input:LabelFilter!) { deleteLabel(filter: $input) { numUids } }`,
		`{"input":{"id":["0x1"]}}`)
}

// Mutation bridges require a valid token: nothing must reach Dgraph without one.
func TestDgraphAddBridge_AccessDenied(t *testing.T) {
	r, got := fakeResolver(t, `{"data":{"addLabel":{"label":[{"id":"0x1"}]}}}`)
	ctx := context.WithValue(resolverCtx("addLabel", field("label", field("id"))),
		"user_ctx_err", fmt.Errorf("token expired"))

	var data model.AddLabelPayload
	err := (&mutationResolver{r}).DgraphAddBridge(ctx, []*model.AddLabelInput{{Name: "l1"}}, nil, &data)
	if err == nil {
		t.Fatal("expected an access denied error, got nil")
	}
	if got.Query != "" {
		t.Errorf("a request was sent to Dgraph: %s", got.Query)
	}
}

// Unknown query name (not add/update/delete/query…) must not reach Dgraph.
func TestDgraphQueryBridge_UnknownQueryType(t *testing.T) {
	r, got := fakeResolver(t, `{}`)

	var data []*model.Label
	err := (&queryResolver{r}).DgraphQueryBridge(resolverCtx("frobnicate"), nil, nil, nil, nil, &data)
	if err == nil {
		t.Fatal("expected an error for an unknown query type, got nil")
	}
	if got.Query != "" {
		t.Errorf("a request was sent to Dgraph: %s", got.Query)
	}
}

// The deprecated raw bridge forwards the client query and variables verbatim.
func TestDgraphBridgeRaw(t *testing.T) {
	r, got := fakeResolver(t, `{"data":{"queryTension":[{"id":"0x1","title":"t1"}]}}`)
	raw := `query queryTension($f:TensionFilter) { queryTension(filter:$f) { id title } }`
	ctx := withRawQuery(resolverCtx("queryTension"), raw, map[string]any{"f": map[string]any{"id": []any{"0x1"}}})

	var data []*model.Tension
	if err := (&queryResolver{r}).DgraphBridgeRaw(ctx, &data); err != nil {
		t.Fatalf("DgraphBridgeRaw returned error: %v", err)
	}
	if len(data) != 1 || data[0].Title != "t1" {
		t.Fatalf("data = %v, want one tension t1", data)
	}
	testutil.CheckRequest(t, got, raw, `{"f":{"id":["0x1"]}}`)
}

// The raw path rewrites nothing: a query carrying a history argument goes out
// verbatim. Cutting the history is the hooks' job, on the Go input
// (graph/tension_resolver.go), for the structured bridges.
func TestDgraphBridgeRaw_NoRewrite(t *testing.T) {
	r, got := fakeResolver(t, `{"data":{"updateTension":{"tension":[{"id":"0x1"}]}}}`)
	raw := `mutation updateTension { updateTension(input:{filter:{id:["0x1"]}, set:{history:[{createdAt:"now"}]}}) { tension { id } } }`
	vars := map[string]any{"tension": map[string]any{"set": map[string]any{
		"history": []any{map[string]any{"createdAt": "now"}},
	}}}
	ctx := withRawQuery(resolverCtx("updateTension"), raw, vars)

	var data *model.UpdateTensionPayload
	if err := (&mutationResolver{r}).DgraphBridgeRaw(ctx, &data); err != nil {
		t.Fatalf("DgraphBridgeRaw returned error: %v", err)
	}
	testutil.CheckRequest(t, got, raw, `{"tension":{"set":{"history":[{"createdAt":"now"}]}}}`)
}

// Raw mutations require a valid token: nothing must reach Dgraph without one.
func TestDgraphBridgeRaw_AccessDenied(t *testing.T) {
	r, got := fakeResolver(t, `{"data":{"addTension":{"tension":[{"id":"0x1"}]}}}`)
	raw := `mutation { addTension(input:[{title:"t1"}]) { tension { id } } }`
	ctx := context.WithValue(withRawQuery(resolverCtx("addTension"), raw, nil),
		"user_ctx_err", fmt.Errorf("token expired"))

	var data *model.AddTensionPayload
	if err := (&mutationResolver{r}).DgraphBridgeRaw(ctx, &data); err == nil {
		t.Fatal("expected an access denied error, got nil")
	}
	if got.Query != "" {
		t.Errorf("a request was sent to Dgraph: %s", got.Query)
	}
}

// postGqlProcess trades Dgraph errors for partial data: @auth rules hide
// fields and report an error alongside the data we still want to return.
func TestPostGqlProcess(t *testing.T) {
	dbc := db.NewDgraph("", "")
	boom := fmt.Errorf("boom")
	partial := []*model.Label{{ID: "0x1"}}

	cases := []struct {
		name    string
		data    any
		err     error
		wantErr bool
	}{
		{"no error", partial, nil, false},
		{"error with data is ignored", partial, boom, false},
		{"error with null data is returned", []*model.Label(nil), boom, true},
		{"error without data is returned", nil, boom, true},
		{"no data, no error", nil, nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := postGqlProcess(resolverCtx("queryLabel"), dbc, c.data, c.err)
			if (err != nil) != c.wantErr {
				t.Errorf("error = %v, wantErr = %v", err, c.wantErr)
			}
		})
	}
}

// The payload graph sent to Dgraph is the gqlgen preloads, nested fields included.
func TestGetQueryGraph_NestedPreloads(t *testing.T) {
	ctx := resolverCtx("queryTension",
		field("id"),
		field("receiver", field("nameid"), field("name")),
	)
	if got, want := GetQueryGraph(ctx), "id receiver { nameid name }"; got != want {
		t.Errorf("GetQueryGraph() = %q, want %q", got, want)
	}
	if !strings.Contains(GetQueryGraph(ctx), "receiver {") {
		t.Error("nested selection lost")
	}
}
