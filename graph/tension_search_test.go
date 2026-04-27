/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2026 Fractale Co
 *
 * This file is part of Fractale.
 */

package graph

import (
	"testing"

	"fractale/fractal6.go/graph/model"
)

// TestHistoryNeedsSearchSync pins the contract for which TensionEvents
// require rebuilding the denormalized Post.message search index. Adding or
// removing an event from `searchSyncEvents` in tension_search.go must come
// with a matching update here.
//
// Comment events are intentionally excluded — see the doc comment on
// `searchSyncEvents` for the rationale.
func TestHistoryNeedsSearchSync(t *testing.T) {
	cases := map[model.TensionEvent]bool{
		// Synced
		model.TensionEventLabelAdded:   true,
		model.TensionEventLabelRemoved: true,

		// Not synced
		model.TensionEventCreated:        false,
		model.TensionEventCommentPushed:  false,
		model.TensionEventCommentDeleted: false,
		model.TensionEventTitleUpdated:   false,
		model.TensionEventAssigneeAdded:  false,
		model.TensionEventBlobCommitted:  false,
	}

	for event, want := range cases {
		t.Run(string(event), func(t *testing.T) {
			ev := event
			history := []*model.EventRef{{EventType: &ev}}
			if got := HistoryNeedsSearchSync(history); got != want {
				t.Errorf("HistoryNeedsSearchSync(%s) = %v, want %v", event, got, want)
			}
		})
	}
}

// TestHistoryNeedsSearchSync_AnyMatchInBatch verifies that a batch
// containing at least one syncing event triggers a sync, even when other
// events in the batch don't.
func TestHistoryNeedsSearchSync_AnyMatchInBatch(t *testing.T) {
	title := model.TensionEventTitleUpdated
	label := model.TensionEventLabelAdded
	comment := model.TensionEventCommentPushed

	history := []*model.EventRef{
		{EventType: &title},
		{EventType: &comment},
		{EventType: &label},
	}
	if !HistoryNeedsSearchSync(history) {
		t.Error("expected sync for batch containing LabelAdded")
	}
}

// TestHistoryNeedsSearchSync_Empty asserts empty/nil history is a no-op.
func TestHistoryNeedsSearchSync_Empty(t *testing.T) {
	if HistoryNeedsSearchSync(nil) {
		t.Error("nil history should not require sync")
	}
	if HistoryNeedsSearchSync([]*model.EventRef{}) {
		t.Error("empty history should not require sync")
	}
}
