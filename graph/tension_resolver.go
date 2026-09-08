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

	"fractale/fractal6.go/graph/model"
	. "fractale/fractal6.go/internal/tools"
	"fractale/fractal6.go/web/auth"
)

////////////////////////////////////////////////
// Tension Resolver
////////////////////////////////////////////////

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
	if input.GovernedNode != nil {
		return nil, LogErr("Forbiden", fmt.Errorf("governed_node is backend-owned"))
	}

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
	id := data.(*model.AddTensionPayload).Tension[0].ID
	if err := CreateTensionHook(uctx, id, history, nil); err != nil {
		return nil, err
	}
	return data, nil
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
