//go:build !integration

/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2026 Fractale Co
 *
 * This file is part of Fractale.
 */

// Tagged !integration: these tests point the db singleton at a fake Dgraph
// endpoint, which would break the integration suite sharing the same binary.

package graph

import (
	"testing"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/internal/testutil"
)

// eventNotif builds a notif with n events on a dummy tension.
func eventNotif(n int) model.EventNotif {
	createdAt := "2026-01-01T00:00:00Z"
	ev := model.TensionEventCommentPushed
	notif := model.EventNotif{Uctx: &model.UserCtx{Username: "testuser"}, Tid: "0x9"}
	for i := 0; i < n; i++ {
		notif.History = append(notif.History, &model.EventRef{CreatedAt: &createdAt, EventType: &ev})
	}
	return notif
}

// PushHistory zips the returned ids back onto the events, but @auth rules can
// filter created events out of the mutation payload: a partial list must error
// instead of assigning the id of event k to event i.
func TestPushHistory_Ids(t *testing.T) {
	db.SetTestJWTKeys()

	t.Run("all ids returned", func(t *testing.T) {
		url, _ := testutil.FakeGqlServer(t, `{"data":{"addEvent":{"event":[{"id":"0x1"},{"id":"0x2"}]}}}`)
		db.SetTestDB(url, "")

		notif := eventNotif(2)
		if err := PushHistory(&notif); err != nil {
			t.Fatalf("PushHistory returned error: %v", err)
		}
		if notif.History[0].ID == nil || *notif.History[0].ID != "0x1" ||
			notif.History[1].ID == nil || *notif.History[1].ID != "0x2" {
			t.Errorf("ids = %v %v, want 0x1 0x2", notif.History[0].ID, notif.History[1].ID)
		}
	})

	t.Run("partial ids returned", func(t *testing.T) {
		url, _ := testutil.FakeGqlServer(t, `{"data":{"addEvent":{"event":[{"id":"0x1"}]}}}`)
		db.SetTestDB(url, "")

		notif := eventNotif(2)
		if err := PushHistory(&notif); err == nil {
			t.Fatal("expected an error for a partial id list, got nil")
		}
		if notif.History[0].ID != nil || notif.History[1].ID != nil {
			t.Errorf("ids were zipped back: %v %v", notif.History[0].ID, notif.History[1].ID)
		}
	})
}
