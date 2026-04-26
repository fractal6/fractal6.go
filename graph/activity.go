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

import "fractale/fractal6.go/graph/model"

// trackedEvents is the set of TensionEvents that contribute to the daily
// Activity counter. Events absent from this set are considered noise and
// skipped by trackActivity (see leaveTrace in tension_op.go).
//
// Curation rationale:
//   - Authoring         : Created, Reopened, Closed, TitleUpdated, TypeUpdated, Pinned, Unpinned
//   - Commenting        : CommentPushed, CommentDeleted
//   - Governance        : BlobCommitted, BlobPushed, BlobArchived, BlobUnarchived,
//     Authority, Visibility, Moved
//   - Project board     : ProjectAdded, ProjectRemoved, ProjectColumnMoved
//   - Membership        : AssigneeAdded/Removed, LabelAdded/Removed,
//     UserJoined/Left, MemberLinked/Unlinked
//   - Skipped (noise)   : BlobCreated (draft state, fires before BlobCommitted),
//     Mentioned (mirror event of a CommentPushed elsewhere)
var trackedEvents = map[model.TensionEvent]bool{
	model.TensionEventCreated:      true,
	model.TensionEventReopened:     true,
	model.TensionEventClosed:       true,
	model.TensionEventTitleUpdated: true,
	model.TensionEventTypeUpdated:  true,
	// model.TensionEventPinned:       true,
	// model.TensionEventUnpinned:     true,

	model.TensionEventCommentPushed:  true,
	model.TensionEventCommentDeleted: true,

	model.TensionEventBlobCommitted:  true,
	model.TensionEventBlobPushed:     true,
	model.TensionEventBlobArchived:   true,
	model.TensionEventBlobUnarchived: true,
	model.TensionEventAuthority:      true,
	model.TensionEventVisibility:     true,
	model.TensionEventMoved:          true,

	model.TensionEventProjectAdded:       true,
	model.TensionEventProjectRemoved:     true,
	model.TensionEventProjectColumnMoved: true,

	model.TensionEventAssigneeAdded:   true,
	model.TensionEventAssigneeRemoved: true,
	model.TensionEventLabelAdded:      true,
	model.TensionEventLabelRemoved:    true,
	model.TensionEventUserJoined:      true,
	model.TensionEventUserLeft:        true,
	model.TensionEventMemberLinked:    true,
	model.TensionEventMemberUnlinked:  true,
}

// isTrackedEvent reports whether the given event should bump the daily
// activity counter. Returns false for noise events.
func isTrackedEvent(e model.TensionEvent) bool {
	return trackedEvents[e]
}
