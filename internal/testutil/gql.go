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

package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// GqlReq is the request body posted to the Dgraph graphql endpoint.
type GqlReq struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

// FakeGqlServer starts a fake Dgraph graphql endpoint always answering `resp`,
// and returns its url along with the request it received (filled after the call).
func FakeGqlServer(t *testing.T, resp string) (string, *GqlReq) {
	t.Helper()
	got := &GqlReq{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(got); err != nil {
			t.Errorf("request body is not valid JSON: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(resp))
	}))
	t.Cleanup(srv.Close)
	return srv.URL, got
}

// CheckRequest asserts the exact query string and variables sent to Dgraph.
func CheckRequest(t *testing.T, got *GqlReq, query, variables string) {
	t.Helper()
	if got.Query != query {
		t.Errorf("query =\n  %s\nwant\n  %s", got.Query, query)
	}
	raw, _ := json.Marshal(got.Variables)
	if string(raw) != variables {
		t.Errorf("variables = %s, want %s", raw, variables)
	}
}
