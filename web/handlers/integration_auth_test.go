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
	"strings"
	"testing"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/internal/testutil"
)

// --- Login Tests ---

func TestLogin_Success(t *testing.T) {
	rr := doRequest("POST", "/auth/login", map[string]string{
		"username": testutil.TestUser,
		"password": testutil.TestPassword,
	})
	requireStatus(t, rr, http.StatusOK)

	// Check response contains UserCtx JSON
	var uctx model.UserCtx
	if err := json.Unmarshal(rr.Body.Bytes(), &uctx); err != nil {
		t.Fatalf("failed to decode response as UserCtx: %v", err)
	}
	if uctx.Username != testutil.TestUser {
		t.Errorf("expected username %q, got %q", testutil.TestUser, uctx.Username)
	}

	requireJWTCookie(t, rr)
}

func TestLogin_WrongPassword(t *testing.T) {
	rr := doRequest("POST", "/auth/login", map[string]string{
		"username": testutil.TestUser,
		"password": "WrongPassword999!",
	})
	requireStatus(t, rr, http.StatusUnauthorized)
}

func TestLogin_NonexistentUser(t *testing.T) {
	rr := doRequest("POST", "/auth/login", map[string]string{
		"username": "nonexistent_user_xyz",
		"password": "SomePassword123!",
	})
	requireStatus(t, rr, http.StatusUnauthorized)
}

// --- Logout Tests ---

func TestLogout(t *testing.T) {
	rr := doRequest("GET", "/auth/logout", nil)
	requireStatus(t, rr, http.StatusOK)

	// Check Set-Cookie clears the jwt cookie
	for _, c := range rr.Result().Cookies() {
		if c.Name == "jwt" {
			if c.Value != "" && c.MaxAge != 0 {
				t.Errorf("expected jwt cookie to be cleared, got value=%q maxAge=%d", c.Value, c.MaxAge)
			}
			return
		}
	}
	t.Error("expected jwt Set-Cookie header in response")
}

// --- Signup Tests ---

func TestSignup_Success(t *testing.T) {
	rr := doRequest("POST", "/auth/signup", map[string]any{
		"username": "signupuser",
		"email":    "signupuser@test.co",
		"password": "SignupPassword123!",
		"lang":     "FR",
	})
	requireStatus(t, rr, http.StatusOK)

	body := strings.TrimSpace(rr.Body.String())
	if body != "true" {
		t.Errorf("expected response body %q, got %q", "true", body)
	}

	// Verify PendingUser exists in DB
	val, err := db.GetDB().GetFieldByEq("PendingUser.email", "signupuser@test.co", "PendingUser.username")
	if err != nil {
		t.Fatalf("failed to query PendingUser: %v", err)
	}
	username, ok := val.(string)
	if !ok || username != "signupuser" {
		t.Errorf("expected PendingUser.username %q, got %v", "signupuser", val)
	}
}

func TestSignup_InvalidUsername(t *testing.T) {
	rr := doRequest("POST", "/auth/signup", map[string]any{
		"username": "ab",
		"email":    "short@test.co",
		"password": "ValidPassword123!",
	})
	requireStatus(t, rr, http.StatusUnauthorized)
}

func TestSignup_DuplicateEmail(t *testing.T) {
	// testuser@test.co already exists from seed data
	rr := doRequest("POST", "/auth/signup", map[string]any{
		"username": "newuser123",
		"email":    testutil.TestEmail,
		"password": "ValidPassword123!",
	})
	requireStatus(t, rr, http.StatusUnauthorized)
}

// --- SignupValidate Tests ---

func TestSignupValidate_Success(t *testing.T) {
	signupEmail := "validateuser@test.co"
	signupUsername := "validateuser"
	signupLang := "FR"

	// 1. Signup to create PendingUser
	rr := doRequest("POST", "/auth/signup", map[string]any{
		"username": signupUsername,
		"email":    signupEmail,
		"password": "ValidatePassword123!",
		"lang":     signupLang,
	})
	requireStatus(t, rr, http.StatusOK)

	// 2. Read email_token from DB
	tokenVal, err := db.GetDB().GetFieldByEq("PendingUser.email", signupEmail, "PendingUser.email_token")
	if err != nil {
		t.Fatalf("failed to query email_token: %v", err)
	}
	emailToken, ok := tokenVal.(string)
	if !ok || emailToken == "" {
		t.Fatalf("expected non-empty email_token, got %v", tokenVal)
	}

	// 3. Validate with the email_token
	rr = doRequest("POST", "/auth/validate", map[string]any{
		"email_token": emailToken,
	})
	requireStatus(t, rr, http.StatusOK)

	// 4. Check response contains UserCtx
	var uctx model.UserCtx
	if err := json.Unmarshal(rr.Body.Bytes(), &uctx); err != nil {
		t.Fatalf("failed to decode UserCtx: %v", err)
	}
	if uctx.Username != signupUsername {
		t.Errorf("expected username %q, got %q", signupUsername, uctx.Username)
	}

	// 5. Check jwt cookie is set
	requireJWTCookie(t, rr)

	// 6. Verify User was created in DB with correct lang
	langVal, err := db.GetDB().GetFieldByEq("User.username", signupUsername, "User.lang")
	if err != nil {
		t.Fatalf("failed to query User.lang: %v", err)
	}
	if lang, ok := langVal.(string); !ok || lang != signupLang {
		t.Errorf("expected User.lang %q, got %v", signupLang, langVal)
	}
}

// --- TokenAck Tests ---

func TestTokenAck_Success(t *testing.T) {
	// Login first to get JWT cookie
	jwtCookie := loginAs(testutil.TestUser, testutil.TestPassword)

	// Call tokenack with the JWT cookie
	rr := doRequest("POST", "/auth/tokenack", nil, jwtCookie)
	requireStatus(t, rr, http.StatusOK)

	// Check response contains refreshed UserCtx
	var uctx model.UserCtx
	if err := json.Unmarshal(rr.Body.Bytes(), &uctx); err != nil {
		t.Fatalf("failed to decode UserCtx: %v", err)
	}
	if uctx.Username != testutil.TestUser {
		t.Errorf("expected username %q, got %q", testutil.TestUser, uctx.Username)
	}

	requireJWTCookie(t, rr)
}

func TestTokenAck_NoAuth(t *testing.T) {
	// Call tokenack without any JWT cookie
	rr := doRequest("POST", "/auth/tokenack", nil)
	requireStatus(t, rr, http.StatusUnauthorized)
}

// --- UpdatePassword Tests ---

func TestUpdatePassword_Success(t *testing.T) {
	newPassword := "NewPassword789!"

	// Login first to ensure user2 works
	_ = loginAs(testutil.TestUser2, testutil.TestPassword2)

	// Update password
	rr := doRequest("POST", "/auth/updatepassword", map[string]string{
		"username":        testutil.TestUser2,
		"password":        testutil.TestPassword2,
		"newPassword":     newPassword,
		"confirmPassword": newPassword,
	})
	requireStatus(t, rr, http.StatusOK)

	// Verify login with new password succeeds
	rr = doRequest("POST", "/auth/login", map[string]string{
		"username": testutil.TestUser2,
		"password": newPassword,
	})
	requireStatus(t, rr, http.StatusOK)

	// Restore original password for other tests
	rr = doRequest("POST", "/auth/updatepassword", map[string]string{
		"username":        testutil.TestUser2,
		"password":        newPassword,
		"newPassword":     testutil.TestPassword2,
		"confirmPassword": testutil.TestPassword2,
	})
	requireStatus(t, rr, http.StatusOK)
}

func TestUpdatePassword_WrongOldPassword(t *testing.T) {
	rr := doRequest("POST", "/auth/updatepassword", map[string]string{
		"username":        testutil.TestUser2,
		"password":        "WrongOldPassword!",
		"newPassword":     "Irrelevant123!",
		"confirmPassword": "Irrelevant123!",
	})
	requireStatus(t, rr, http.StatusUnauthorized)
}
