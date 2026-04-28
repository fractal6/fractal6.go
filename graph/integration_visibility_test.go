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

package graph_test

import (
	"testing"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/internal/testutil"
	"fractale/fractal6.go/web/auth"
)

// makeUctx returns a UserCtx with roles populated, so $ROOTIDS / $OWNIDS
// resolve correctly in BuildGqlToken. Pass an empty username for anonymous.
func makeUctx(t *testing.T, username string) model.UserCtx {
	t.Helper()
	uctx := &model.UserCtx{
		Username: username,
		Rights:   model.UserRights{Type: model.UserTypeRegular},
	}
	if username == "" {
		return *uctx
	}
	refreshed, err := auth.MaybeRefresh(uctx)
	if err != nil {
		t.Fatalf("MaybeRefresh(%s): %v", username, err)
	}
	return *refreshed
}

// canSeeNode runs queryNode(filter:{nameid:{in:[nameid]}}) and reports whether
// the @auth rule lets this user see it.
func canSeeNode(t *testing.T, uctx model.UserCtx, nameid string) bool {
	t.Helper()
	res, err := db.GetDB().Query(uctx, "node", "nameid", []string{nameid}, "id")
	if err != nil {
		t.Fatalf("queryNode(nameid=%q): %v", nameid, err)
	}
	return len(res) > 0
}

// canSeeProject runs queryProject(filter:{nameid:{in:[nameid]}}).
func canSeeProject(t *testing.T, uctx model.UserCtx, nameid string) bool {
	t.Helper()
	res, err := db.GetDB().Query(uctx, "project", "nameid", []string{nameid}, "id")
	if err != nil {
		t.Fatalf("queryProject(nameid=%q): %v", nameid, err)
	}
	return len(res) > 0
}

// canSeeColumn runs queryProjectColumn { id name } and checks whether the named
// column appears. ProjectColumn.name has no @id/@search index, so we list all
// visible columns and match in Go.
func canSeeColumn(t *testing.T, uctx model.UserCtx, name string) bool {
	t.Helper()
	payload := make(model.JsonAtom)
	err := db.GetDB().QueryGql(uctx, "rawQuery", map[string]string{
		"QueryName": "queryProjectColumn",
		"RawQuery":  "{ queryProjectColumn { name } }",
		"Variables": "{}",
	}, payload)
	if err != nil {
		t.Fatalf("queryProjectColumn: %v", err)
	}
	cols, _ := payload["queryProjectColumn"].([]any)
	for _, c := range cols {
		m, _ := c.(map[string]any)
		if m["name"] == name {
			return true
		}
	}
	return false
}

// TestVisibility_Auth covers the Dgraph @auth gates on Node, Project, and
// ProjectColumn for the three visibility levels (Public / Private / Secret),
// across three actors:
//   - testuser:  Member of sec-org, no role in private-circle or secret-circle
//   - testuser2: Owner of sec-org, Coordinator inside secret-circle
//   - anonymous: no JWT
//
// Regression for the Secret-circle bootstrap rule: testuser is a sibling Member
// of secret-circle (i.e. has a role in sec-org##@testuser, sibling of secret-circle).
// Before the fix, the bootstrap rule allowed any first_link of any sibling — so
// testuser could see secret-circle. After tightening to role_type=Coordinator,
// testuser must not.
func TestVisibility_Auth(t *testing.T) {
	user := makeUctx(t, testutil.TestUser)   // Member of sec-org
	owner := makeUctx(t, testutil.TestUser2) // Owner of sec-org + Coordinator of secret-circle
	anon := makeUctx(t, "")

	t.Run("Public", func(t *testing.T) {
		// test-org root is Public; everyone (including anonymous) must see it,
		// its public-project, and its column.
		cases := []struct {
			actor string
			uctx  model.UserCtx
		}{
			{"anonymous", anon},
			{"member", user},
			{"owner", owner},
		}
		for _, c := range cases {
			t.Run(c.actor, func(t *testing.T) {
				if !canSeeNode(t, c.uctx, "test-org") {
					t.Errorf("queryNode(test-org): expected visible to %s, got hidden", c.actor)
				}
				if !canSeeProject(t, c.uctx, testutil.PublicProjectNameid) {
					t.Errorf("queryProject(%s): expected visible to %s, got hidden",
						testutil.PublicProjectNameid, c.actor)
				}
				if !canSeeColumn(t, c.uctx, testutil.PublicColumnName) {
					t.Errorf("queryProjectColumn(%s): expected visible to %s, got hidden",
						testutil.PublicColumnName, c.actor)
				}
			})
		}
	})

	t.Run("Private", func(t *testing.T) {
		// sec-org#private-circle is Private; only org members and the org owner
		// can see it. Anonymous must be denied.
		t.Run("anonymous_denied", func(t *testing.T) {
			if canSeeNode(t, anon, testutil.SecOrgPrivateCircle) {
				t.Errorf("queryNode(%s): expected hidden to anonymous, got visible",
					testutil.SecOrgPrivateCircle)
			}
			if canSeeProject(t, anon, testutil.PrivateProjectNameid) {
				t.Errorf("queryProject(%s): expected hidden to anonymous, got visible",
					testutil.PrivateProjectNameid)
			}
			if canSeeColumn(t, anon, testutil.PrivateColumnName) {
				t.Errorf("queryProjectColumn(%s): expected hidden to anonymous, got visible",
					testutil.PrivateColumnName)
			}
		})
		t.Run("member_allowed", func(t *testing.T) {
			if !canSeeNode(t, user, testutil.SecOrgPrivateCircle) {
				t.Errorf("queryNode(%s): expected visible to org member, got hidden",
					testutil.SecOrgPrivateCircle)
			}
			if !canSeeProject(t, user, testutil.PrivateProjectNameid) {
				t.Errorf("queryProject(%s): expected visible to org member, got hidden",
					testutil.PrivateProjectNameid)
			}
			if !canSeeColumn(t, user, testutil.PrivateColumnName) {
				t.Errorf("queryProjectColumn(%s): expected visible to org member, got hidden",
					testutil.PrivateColumnName)
			}
		})
		t.Run("owner_allowed", func(t *testing.T) {
			if !canSeeNode(t, owner, testutil.SecOrgPrivateCircle) {
				t.Errorf("queryNode(%s): expected visible to org owner, got hidden",
					testutil.SecOrgPrivateCircle)
			}
			if !canSeeProject(t, owner, testutil.PrivateProjectNameid) {
				t.Errorf("queryProject(%s): expected visible to org owner, got hidden",
					testutil.PrivateProjectNameid)
			}
			if !canSeeColumn(t, owner, testutil.PrivateColumnName) {
				t.Errorf("queryProjectColumn(%s): expected visible to org owner, got hidden",
					testutil.PrivateColumnName)
			}
		})
	})

	t.Run("Secret", func(t *testing.T) {
		// sec-org#secret-circle is Secret. Only the org Owner (via $OWNIDS) and
		// users with an explicit role inside the circle can see it.
		t.Run("anonymous_denied", func(t *testing.T) {
			if canSeeNode(t, anon, testutil.SecOrgSecretCircle) {
				t.Errorf("queryNode(%s): expected hidden to anonymous, got visible",
					testutil.SecOrgSecretCircle)
			}
			if canSeeProject(t, anon, testutil.SecretProjectNameid) {
				t.Errorf("queryProject(%s): expected hidden to anonymous, got visible",
					testutil.SecretProjectNameid)
			}
			if canSeeColumn(t, anon, testutil.SecretColumnName) {
				t.Errorf("queryProjectColumn(%s): expected hidden to anonymous, got visible",
					testutil.SecretColumnName)
			}
		})
		// Regression: org-Member-but-not-Coordinator must be denied. Pre-fix,
		// the broad bootstrap rule (parent.children { first_link == user })
		// allowed any sibling Member through.
		t.Run("sibling_member_denied", func(t *testing.T) {
			if canSeeNode(t, user, testutil.SecOrgSecretCircle) {
				t.Errorf("queryNode(%s): expected hidden to non-coordinator member (regression), got visible",
					testutil.SecOrgSecretCircle)
			}
			if canSeeProject(t, user, testutil.SecretProjectNameid) {
				t.Errorf("queryProject(%s): expected hidden to non-coordinator member, got visible",
					testutil.SecretProjectNameid)
			}
			if canSeeColumn(t, user, testutil.SecretColumnName) {
				t.Errorf("queryProjectColumn(%s): expected hidden to non-coordinator member, got visible",
					testutil.SecretColumnName)
			}
		})
		// testuser2 has both: Owner role at root ($OWNIDS shortcut) AND an explicit
		// Coordinator role inside secret-circle. Either path grants access; the
		// test only requires that access is granted.
		t.Run("owner_or_inner_coordo_allowed", func(t *testing.T) {
			if !canSeeNode(t, owner, testutil.SecOrgSecretCircle) {
				t.Errorf("queryNode(%s): expected visible to org owner, got hidden",
					testutil.SecOrgSecretCircle)
			}
			if !canSeeProject(t, owner, testutil.SecretProjectNameid) {
				t.Errorf("queryProject(%s): expected visible to org owner, got hidden",
					testutil.SecretProjectNameid)
			}
			if !canSeeColumn(t, owner, testutil.SecretColumnName) {
				t.Errorf("queryProjectColumn(%s): expected visible to org owner, got hidden",
					testutil.SecretColumnName)
			}
		})
	})
}
