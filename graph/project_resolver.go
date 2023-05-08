/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2022 Fractale Co
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
	"strconv"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph/model"
	. "fractale/fractal6.go/tools"
)

//
// ProjectCard Resolver
// --
// We update the card position in list when thery are
// moved to respect the shifting.

// Add "ProjectCard"
func addProjectCardHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
	// Pre-processing: Authorization
	// --

	// Get User context
	//ctx, uctx, err := auth.GetUserContext(ctx)
	//if err != nil {
	//    return nil, LogErr("Access denied", err)
	//}

	data, err := next(ctx)
	if err != nil {
		return data, err
	}
	d := data.(*model.AddProjectCardPayload)
	if d == nil {
		return nil, LogErr("add projectCard", fmt.Errorf("no card added"))
	}

	// Post-processing: fix pos in columns list
	// --

	for _, card := range d.ProjectCard {
		if card.ID == "" {
			return data, fmt.Errorf("id payload required for project card mutation")
		}

		_, err := db.GetDB().Meta("incrementCardPos", map[string]string{"cardid": card.ID})
		if err != nil {
			return data, err
		}
	}

	return data, err
}

// Update "ProjectCard"
func updateProjectCardHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
	// Pre-processing: Authorization
	// --

	// Validate input
	input := graphql.GetResolverContext(ctx).Args["input"].(model.UpdateProjectCardInput)
	ids := input.Filter.ID
	isMoved := false
	var uid string
	var old_pos string
	var old_colid string
	if input.Set != nil && len(ids) == 1 {
		if input.Set.Pos != nil && input.Set.Pc != nil {
			// Extract the value before moving
			isMoved = true
			uid = ids[0]
			// @TODO: optimize the number of query !
			x, err := db.GetDB().GetFieldById(uid, "ProjectCard.pos")
			if err != nil {
				return nil, err
			}
			old_pos = strconv.Itoa(int(x.(float64)))

			x, err = db.GetDB().GetSubFieldById(uid, "ProjectCard.pc", "uid")
			if err != nil {
				return nil, err
			}
			old_colid = x.(string)
		}
	}

	if input.Remove != nil {
		return nil, fmt.Errorf("remove is not allowed for this mutation")
	}

	data, err := next(ctx)

	// Post-processing: fix pos in columns list
	// --

	// Auto increment card position only when updating a single card,
	// otherwise, assume that the user is know what he is doing.
	if isMoved {
		_, err := db.GetDB().Meta("incrementCardPos2", map[string]string{
			"cardid":    uid,
			"old_pos":   old_pos,
			"old_colid": old_colid,
		})
		if err != nil {
			return data, err
		}
	}

	return data, err
}
