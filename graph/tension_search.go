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

package graph

import (
	"fmt"
	"strings"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph/model"
	. "fractale/fractal6.go/internal/tools"
)

// searchSyncEvents lists tension events whose effect requires rebuilding the
// denormalized Post.message search index. Triggered post-mutation in
// updateTensionHook.
//
// Comment events (CommentPushed/CommentDeleted) are intentionally absent:
//   - CommentPushed via updateTension is always a NEW (non-first) comment, so
//     the synthesized message — which only includes the FIRST comment — does
//     not change.
//   - First-comment edits (via updateComment) and deletions (via the
//     CommentDeleted EMAP action) would require a cid → tid + first-comment
//     lookup; doing that without a reverse edge means scanning all tensions
//     with uid_in, which is too costly for the rare-path. The denormalized
//     index will be refreshed the next time a label changes on the tension.
var searchSyncEvents = map[model.TensionEvent]bool{
	model.TensionEventLabelAdded:   true,
	model.TensionEventLabelRemoved: true,
	model.TensionEventReopened:     true,
	model.TensionEventClosed:       true,
}

// HistoryNeedsSearchSync reports whether any event in history affects the
// denormalized search index and therefore requires a post-mutation sync.
func HistoryNeedsSearchSync(history []*model.EventRef) bool {
	for _, e := range history {
		if e != nil && e.EventType != nil && searchSyncEvents[*e.EventType] {
			return true
		}
	}
	return false
}

// SyncTensionSearchMessage rebuilds the denormalized Post.message field on a
// tension so that label names and the first comment body become searchable
// via the fulltext index on Post.message.
//
// Returns nil and writes nothing when there is no synthesizable content
// (no labels and no first comment) — this preserves any existing
// Post.message (e.g. a body set on creation) instead of clobbering it with
// an empty string.
func SyncTensionSearchMessage(tid string) error {
	labels, firstComment, err := db.GetDB().GetTensionSearchData(tid)
	if err != nil {
		return fmt.Errorf("fetch: %w", err)
	}

	var parts []string
	if len(labels) > 0 {
		parts = append(parts, "---\n"+strings.Join(labels, ", ")+"\n---")
	}
	if firstComment != "" {
		parts = append(parts, firstComment)
	}
	if len(parts) == 0 {
		return nil
	}

	msg := QuoteString(strings.Join(parts, "\n\n"))
	if err := db.GetDB().SetFieldById(tid, "Post.message", msg); err != nil {
		return fmt.Errorf("set: %w", err)
	}
	return nil
}

// GoSyncSearchMessage runs SyncTensionSearchMessage in a goroutine. Call this
// from resolvers/handlers AFTER the mutation that changed the tension content
// has been persisted; firing it before the write completes races with Dgraph
// and yields a stale index.
func GoSyncSearchMessage(tid string) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("error: SyncTensionSearchMessage panic for %s: %v\n", tid, r)
			}
		}()
		if err := SyncTensionSearchMessage(tid); err != nil {
			fmt.Printf("error: SyncTensionSearchMessage for %s: %v\n", tid, err)
		}
	}()
}
