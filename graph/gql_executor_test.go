//go:build !integration

/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2026 Fractale Co
 *
 * This file is part of Fractale.
 */

// Tagged !integration: points the db singleton at a fake Dgraph endpoint,
// which would break the integration suite sharing the same binary.

package graph

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"

	"fractale/fractal6.go/db"
	gen "fractale/fractal6.go/graph/generated"
	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/internal/testutil"
)

// gqlServer wires the real executable schema (same as web/handlers.GraphqlHandler)
// onto a fake Dgraph endpoint answering resp.
func gqlServer(t *testing.T, resp string) (*handler.Server, *testutil.GqlReq) {
	t.Helper()
	db.SetTestJWTKeys()
	url, got := testutil.FakeGqlServer(t, resp)
	db.SetTestDB(url, "")

	srv := handler.New(gen.NewExecutableSchema(Init()))
	srv.AddTransport(transport.POST{})
	return srv, got
}

// post runs query/variables through the gqlgen server as a real HTTP request.
func post(t *testing.T, srv *handler.Server, query string, variables map[string]any) map[string]any {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"query": query, "variables": variables})
	req := httptest.NewRequest("POST", "/api", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), "user_ctx", &model.UserCtx{Username: "testuser", Hit: 1})
	ctx = context.WithValue(ctx, "iat", "2026-01-01T00:00:00Z")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req.WithContext(ctx))

	var out map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("response is not JSON (%d): %s", w.Code, w.Body.String())
	}
	if out["errors"] != nil {
		t.Fatalf("graphql errors: %v", out["errors"])
	}
	return out
}

// The raw bridge forwards gc.RawQuery / gc.Variables untouched, so what a real
// gqlgen request carries is what Dgraph receives. Everything above the bridge
// (parsing, variable coercion, the operation context) is gqlgen's, and this is
// the only test that exercises it instead of a hand-built OperationContext.
func TestGqlServerRawBridgeForwardsVariables(t *testing.T) {
	srv, got := gqlServer(t, `{"data":{"getTension":{"id":"0x1","title":"t1"}}}`)
	query := `query getT($id: ID!) { getTension(id: $id) { id title } }`

	out := post(t, srv, query, map[string]any{"id": "0x1"})

	testutil.CheckRequest(t, got, query, `{"id":"0x1"}`)
	tension, _ := out["data"].(map[string]any)["getTension"].(map[string]any)
	if tension["title"] != "t1" {
		t.Errorf("title = %v, want t1", tension["title"])
	}
}
