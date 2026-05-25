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
	"context"
	"fmt"

	"github.com/99designs/gqlgen/graphql"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/internal/notify"
	. "fractale/fractal6.go/internal/tools"
	"fractale/fractal6.go/web/auth"
)

////////////////////////////////////////////////
// Tension Resolver
////////////////////////////////////////////////

// firstCommentMessage returns the Message text of the first comment in the
// input slice — that's the comment created alongside the mutation that the
// upload gate counts inline pastes against. Empty slice / nil messages
// return "".
func firstCommentMessage(comments []*model.CommentRef) string {
	if len(comments) == 0 || comments[0] == nil || comments[0].Message == nil {
		return ""
	}
	return *comments[0].Message
}

// registerInlineUploads counts the bare `![](paste-N.png)` references in
// `message` and pre-registers them on the tension's upload gate so the
// notifier waits for the matching /file/upload calls before reading
// Comment.files. No-op when count is zero or notify.Global() is unset.
func registerInlineUploads(ctx context.Context, tid, message string) {
	n := CountInlineImageCandidates(message)
	if n == 0 {
		return
	}
	if err := notify.Global().Register(ctx, tid, n); err != nil {
		LogErr("upload-gate register", err)
	}
}

func tensionInputHook(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
	data, err := setUpdateContextInfo(ctx, obj, next) // for @hasEvent+@isOwner
	if err != nil {
		return data, err
	}

	// newData := data.([]*model.AddContractInput)

	// Set BlobType -- based on Blob.
	b2i := map[bool]int{false: 0, true: 1}
	switch newData := data.(type) {
	case model.UpdateTensionInput:
		if newData.Set == nil {
			break
		}
		input := newData.Set
		if len(input.Blobs) == 0 {
			break
		}
		// Blob are update OneByOne
		blob := input.Blobs[0]
		if blob.Node == nil {
			break
		}
		// Blob are update OneByOne
		blob_type_lvl := b2i[blob.Node.About != nil] + b2i[blob.Node.Mandate != nil]*2
		var bt model.BlobType
		switch blob_type_lvl {
		case 1:
			bt = model.BlobTypeOnAbout
		case 2:
			bt = model.BlobTypeOnMandate
		case 3:
			bt = model.BlobTypeOnAboutAndMandate
		}
		blob.BlobType = &bt
		return newData, err
	case []*model.AddTensionInput:
		for _, input := range newData {
			if len(input.Blobs) == 0 {
				break
			}
			// Blob are update OneByOne
			blob := input.Blobs[0]
			if blob.Node == nil {
				break
			}
			bt := model.BlobTypeOnNode
			blob.BlobType = &bt
		}
		return newData, err
	}

	return data, err
}

// Add Tension - Hook
func addTensionHook(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
	// Get User context
	ctx, uctx, err := auth.GetUserContext(ctx)
	if err != nil {
		return nil, LogErr("Access denied", err)
	}

	// Validate Input
	inputs := graphql.GetResolverContext(ctx).Args["input"].([]*model.AddTensionInput)
	if len(inputs) != 1 {
		return nil, LogErr("add tension", fmt.Errorf("One and only one tension allowed."))
	}
	if !PayloadContains(ctx, "id") {
		return nil, LogErr("field missing", fmt.Errorf("id field is required in tension payload"))
	}
	input := inputs[0]

	// History and notification Logics --
	// In order to notify user on the given event, we need to know their ids to pass and link them
	// to the notification (UserEvent edge) function. To do so we first cut the history from the original
	// input, and push then the history (see the PushHistory function).
	ctx = context.WithValue(ctx, "cut_history", true) // Used by DgraphQueryResolverRaw
	history := input.History
	input.History = nil
	// Execute query
	data, err := next(ctx)
	if err != nil {
		return data, err
	}
	if data.(*model.AddTensionPayload) == nil {
		return nil, LogErr("add tension", fmt.Errorf("silent error: no tension added."))
	}
	tension := data.(*model.AddTensionPayload).Tension[0]
	id := tension.ID

	// Validate and process Blob Event
	ok, _, err := TensionEventHook(uctx, id, history, nil)
	if !ok || err != nil {
		// Delete the tension just added
		e := db.GetDB().DeleteTensionDeep(id)
		if e != nil {
			panic(e)
		}
	}
	if err != nil {
		return data, err
	}
	if ok {
		GoSyncSearchMessage(id)
		registerInlineUploads(ctx, id, firstCommentMessage(input.Comments))
		PublishTensionEvent(model.EventNotif{Uctx: uctx, Tid: id, History: history})
		return data, err
	}
	return nil, LogErr("Access denied", fmt.Errorf("Contact a coordinator to access this resource."))
}

// Update Tension - Hook
func updateTensionHook(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
	// Get User context
	ctx, uctx, err := auth.GetUserContext(ctx)
	if err != nil {
		return nil, LogErr("Access denied", err)
	}

	// Validate input
	input := graphql.GetResolverContext(ctx).Args["input"].(model.UpdateTensionInput)
	ids := input.Filter.ID
	if len(ids) != 1 {
		return nil, LogErr("update tension", fmt.Errorf("One and only one tension allowed."))
	}

	// Validate Event prior the mutation
	var blob *model.BlobRef
	var contract *model.Contract
	var ok bool
	if input.Set != nil {
		if len(input.Set.Blobs) > 0 {
			blob = input.Set.Blobs[0]
		}
		ok, contract, err = TensionEventHook(uctx, ids[0], input.Set.History, blob)
		if err != nil {
			return nil, err
		}
		if ok {
			// History and notification Logics --
			// In order to notify user on the given event, we need to know
			// their ids to pass and link them to the user's notifications (UserEvent edge).
			// To do so we first cut the history from the original input,
			// and push then the history (see the [[PushHistory]] function).
			ctx = context.WithValue(ctx, "cut_history", true) // Used by DgraphQueryResolverRaw
			history := input.Set.History
			now := Now()
			input.Set.History = nil
			input.Set.UpdatedAt = &now
			// Pre-register the upload gate from the soon-to-be-persisted
			// comment payload. Doing it on the input (before next) means we
			// don't race with the upload handler if the client fires uploads
			// the moment the GraphQL mutation returns.
			registerInlineUploads(ctx, ids[0], firstCommentMessage(input.Set.Comments))
			// Execute query
			data, err := next(ctx)
			if err != nil {
				return data, err
			}
			// Rebuild the denormalized search index AFTER the mutation has been
			// persisted; firing it earlier would read pre-mutation state.
			if HistoryNeedsSearchSync(history) {
				GoSyncSearchMessage(ids[0])
			}
			PublishTensionEvent(model.EventNotif{Uctx: uctx, Tid: ids[0], History: history})
			return data, err
		} else if contract != nil {
			var t model.UpdateTensionPayload
			t.Tension = []*model.Tension{{
				Contracts: []*model.Contract{contract},
			}}
			return &t, err
		} else {
			return nil, LogErr("Access denied", fmt.Errorf("Contact a coordinator to access this resource."))
		}
	}

	return nil, LogErr("Access denied", fmt.Errorf("Input remove not implemented."))
}
