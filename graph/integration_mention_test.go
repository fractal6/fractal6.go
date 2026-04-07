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

	. "fractale/fractal6.go/graph"
	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/internal/testutil"
)

// TestUpdateWithMentionnedUser_MemberIsAdded verifies that mentioning a member
// of the org via "@username" adds them to the notification map.
func TestUpdateWithMentionnedUser_MemberIsAdded(t *testing.T) {
	// sec-org has both testuser and testuser2 as members.
	users := make(map[string]model.UserNotifInfo)
	msg := "Hello @" + testutil.TestUser + ", please check this."

	err := UpdateWithMentionnedUser(msg, "sec-org", users)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ui, ok := users[testutil.TestUser]
	if !ok {
		t.Fatal("expected testuser to be added to users map via mention")
	}
	if ui.Reason != model.ReasonIsMentionned {
		t.Errorf("expected reason ReasonIsMentionned, got %v", ui.Reason)
	}
}

// TestUpdateWithMentionnedUser_NonMemberSkipped verifies that mentioning a user
// who is NOT a member of the org does not add them to the notification map.
func TestUpdateWithMentionnedUser_NonMemberSkipped(t *testing.T) {
	// test-org only has testuser as a member, not testuser2.
	users := make(map[string]model.UserNotifInfo)
	msg := "Hey @" + testutil.TestUser2 + " what do you think?"

	err := UpdateWithMentionnedUser(msg, "test-org", users)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := users[testutil.TestUser2]; ok {
		t.Fatal("non-member should not be added to users map")
	}
}

// TestUpdateWithMentionnedUser_ExistingUserReasonUpdated verifies that if a user
// is already in the map (e.g. as a coordinator), their reason is upgraded to
// ReasonIsMentionned when they are mentioned.
func TestUpdateWithMentionnedUser_ExistingUserReasonUpdated(t *testing.T) {
	users := map[string]model.UserNotifInfo{
		testutil.TestUser: {
			User:   model.User{Username: testutil.TestUser},
			Reason: model.ReasonIsCoordo,
		},
	}
	msg := "cc @" + testutil.TestUser

	err := UpdateWithMentionnedUser(msg, "sec-org", users)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if users[testutil.TestUser].Reason != model.ReasonIsMentionned {
		t.Errorf("expected reason to be updated to ReasonIsMentionned, got %v", users[testutil.TestUser].Reason)
	}
}

// TestUpdateWithMentionnedUser_CodeBlockIgnored verifies that usernames inside
// code blocks are not treated as mentions.
func TestUpdateWithMentionnedUser_CodeBlockIgnored(t *testing.T) {
	users := make(map[string]model.UserNotifInfo)
	msg := "See this code:\n```\n@" + testutil.TestUser + "\n```\nDone."

	err := UpdateWithMentionnedUser(msg, "sec-org", users)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := users[testutil.TestUser]; ok {
		t.Fatal("username inside code block should not trigger a mention")
	}
}
