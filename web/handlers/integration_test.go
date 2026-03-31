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
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/internal/testutil"
	"fractale/fractal6.go/web/auth"
	"fractale/fractal6.go/web/email"
	. "fractale/fractal6.go/web/handlers"
	middle6 "fractale/fractal6.go/web/middleware"
)

// testRouter is the shared chi router used by all integration tests.
var testRouter chi.Router

// mockEmailServer captures email requests sent during tests.
var mockEmailServer *httptest.Server

func TestMain(m *testing.M) {
	// 1. Override db singleton with test Dgraph addresses
	db.SetTestDB(testutil.TestHTTPAddr+"/graphql", testutil.TestGrpcAddr)

	// 2. Verify test data is present (seeded by cmd/testsetup).
	ex, err := db.GetDB().Exists("User.username", testutil.TestUser, nil)
	if err != nil || !ex {
		log.Fatal("Test data not found. Run 'go run ./cmd/testsetup' first (or use 'make test-integration').")
	}

	// 3. Start mock email server
	mockEmailServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success"}`))
	}))
	defer mockEmailServer.Close()

	// 4. Override email URL to use mock server
	email.SetTestConfig(mockEmailServer.URL, "test-secret")

	// 5. Build the test router
	testRouter = buildTestRouter()

	os.Exit(m.Run())
}

// buildTestRouter creates a chi router with JWT middleware and all handler routes,
// mirroring the production setup in cmd/server.go.
func buildTestRouter() chi.Router {
	r := chi.NewRouter()

	tkMaster := auth.GetTokenMaster()

	// Middleware stack (minimal for tests)
	r.Use(middleware.RequestID)
	r.Use(middle6.JwtVerifier(tkMaster.GetAuth()))
	r.Use(middle6.JwtDecode)
	r.Use(middleware.Timeout(30 * time.Second))

	// Query routes
	r.Route("/q", func(r chi.Router) {
		r.Route("/nodes", func(r chi.Router) {
			r.Post("/sub", SubNodes)
		})
		r.Route("/members", func(r chi.Router) {
			r.Post("/sub", SubMembers)
		})
		r.Route("/labels", func(r chi.Router) {
			r.Post("/top", NodeHolderHandler(db.GetDB().GetTopLabels))
			r.Post("/sub", NodeHolderHandler(db.GetDB().GetSubLabels))
		})
		r.Route("/roles", func(r chi.Router) {
			r.Post("/top", NodeHolderHandler(db.GetDB().GetTopRoles))
			r.Post("/sub", NodeHolderHandler(db.GetDB().GetSubRoles))
		})
		r.Route("/projects", func(r chi.Router) {
			r.Post("/sub", NodeHolderHandler(db.GetDB().GetSubProjects))
		})
		r.Route("/tensions", func(r chi.Router) {
			r.Post("/light", TensionsHandler("light"))
			r.Post("/int", TensionsHandler("int"))
			r.Post("/ext", TensionsHandler("ext"))
			r.Post("/all", TensionsHandler("all"))
			r.Post("/count", TensionsCount)
		})
	})

	// Auth routes
	r.Route("/auth", func(r chi.Router) {
		r.Post("/signup", Signup)
		r.Post("/validate", SignupValidate)
		r.Post("/login", Login)
		r.Get("/logout", Logout)
		r.Post("/tokenack", TokenAck)
		r.Post("/updatepassword", UpdatePassword)
		// Organisation
		r.Post("/createorga", CreateOrga)
		r.Post("/setusercanjoin", SetUserCanJoin)
		r.Post("/setguestcancreatetension", SetGuestCanCreateTension)
		r.Post("/setlexicon", SetLexicon)
		r.Post("/setistemplatetensiononly", SetIsTemplateTensionOnly)
	})

	return r
}

// doRequest performs an HTTP request against the test router and returns the response recorder.
func doRequest(method, path string, body any, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	var reqBody io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			panic(fmt.Sprintf("failed to marshal request body: %v", err))
		}
		reqBody = bytes.NewReader(jsonBytes)
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	for _, c := range cookies {
		req.AddCookie(c)
	}

	rr := httptest.NewRecorder()
	testRouter.ServeHTTP(rr, req)
	return rr
}

// requireStatus fails the test if the response code doesn't match.
func requireStatus(t *testing.T, rr *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rr.Code != want {
		t.Fatalf("expected status %d, got %d: %s", want, rr.Code, rr.Body.String())
	}
}

// requireJWTCookie fails the test if no non-empty jwt cookie is present.
func requireJWTCookie(t *testing.T, rr *httptest.ResponseRecorder) {
	t.Helper()
	for _, c := range rr.Result().Cookies() {
		if c.Name == "jwt" && c.Value != "" {
			return
		}
	}
	t.Fatal("expected jwt cookie in response")
}

// loginAs performs a login and returns the JWT cookie.
func loginAs(username, password string) *http.Cookie {
	rr := doRequest("POST", "/auth/login", map[string]string{
		"username": username,
		"password": password,
	})
	if rr.Code != http.StatusOK {
		panic(fmt.Sprintf("loginAs %s failed with status %d: %s", username, rr.Code, rr.Body.String()))
	}

	for _, c := range rr.Result().Cookies() {
		if c.Name == "jwt" {
			return c
		}
	}
	panic(fmt.Sprintf("loginAs %s: no jwt cookie in response", username))
}
