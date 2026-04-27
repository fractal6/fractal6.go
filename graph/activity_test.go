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
 */

package graph

import (
	"testing"

	"fractale/fractal6.go/graph/model"
)

// TestIsTrackedEvent pins the noise-filter contract: which TensionEvents
// bump the daily Activity counter, and which are intentionally dropped.
//
// Toggling any line of `trackedEvents` in activity.go must come with a
// matching toggle here — that's the point of this test.
func TestIsTrackedEvent(t *testing.T) {
	cases := map[model.TensionEvent]bool{
		// Tension lifecycle / metadata
		model.TensionEventCreated:      true,
		model.TensionEventReopened:     true,
		model.TensionEventClosed:       true,
		model.TensionEventTitleUpdated: true,
		model.TensionEventTypeUpdated:  true,

		// Comments
		model.TensionEventCommentPushed: true,

		// Governance
		model.TensionEventBlobCommitted:  true,
		model.TensionEventBlobPushed:     true,
		model.TensionEventBlobUnarchived: true,
		model.TensionEventAuthority:      true,
		model.TensionEventVisibility:     true,
		model.TensionEventMoved:          true,

		// Project board
		model.TensionEventProjectAdded:       true,
		model.TensionEventProjectRemoved:     true,
		model.TensionEventProjectColumnMoved: true,

		// Membership
		model.TensionEventLabelAdded:   true,
		model.TensionEventLabelRemoved: true,
		model.TensionEventUserJoined:   true,
		model.TensionEventMemberLinked: true,

		// Intentionally excluded (noise)
		model.TensionEventBlobCreated: false, // draft state, fires before BlobCommitted
		model.TensionEventMentioned:   false, // mirror of a CommentPushed elsewhere
		model.TensionEventPinned:      false, // low-signal curation
		model.TensionEventUnpinned:    false, // low-signal curation
		// and more...
	}

	for event, want := range cases {
		t.Run(string(event), func(t *testing.T) {
			if got := isTrackedEvent(event); got != want {
				t.Errorf("isTrackedEvent(%s) = %v, want %v", event, got, want)
			}
		})
	}
}

// TestIsTrackedEvent_ZeroValue asserts that a zero-value TensionEvent (which
// can occur if a caller accidentally passes an unset event) is treated as noise.
func TestIsTrackedEvent_ZeroValue(t *testing.T) {
	var zero model.TensionEvent
	if isTrackedEvent(zero) {
		t.Errorf("isTrackedEvent(zero-value) = true, want false")
	}
}
