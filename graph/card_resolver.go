/*
 * Fractale - Self-organisation for humans.
 * Copyright (C) 2023 Fractale Co
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
	"log"
	"strconv"

	"github.com/99designs/gqlgen/graphql"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph/model"
	. "fractale/fractal6.go/tools"
	"fractale/fractal6.go/web/auth"
)

// ProjectCard Resolver
// --
// We update the card position in list when thery are
// moved to respect the shifting.

type ProjectCardLoc struct {
	ID        string `json:"id"`
	Colid     string
	Projectid string
	Pos       int
	Contentid string
	Typenames []string
}

var QueryCardLoc db.QueryMut = db.QueryMut{
	Q: `query {
            all(func: uid({{.cardid}})) @normalize {
                uid
                ProjectCard.pc {
                    colid: uid
                    ProjectColumn.project { projectid: uid }
                }
                pos: ProjectCard.pos
                ProjectCard.card {
                    contentid: uid
                    typenames: dgraph.type
                }
            }
        }`,
}

// Add "ProjectCard"
func addProjectCardHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
	// Pre-processing:
	// - Auth

	// Get User context
	ctx, uctx, err := auth.GetUserContext(ctx)
	if err != nil {
		return nil, LogErr("Access denied", err)
	}

	// Validate input
	var inputs []model.AddProjectCardInput
	ExtractInputs(ctx, &inputs)
	for _, input := range inputs {
		x, err := db.GetDB().GetSubFieldById(*input.Pc.ID, "ProjectColumn.project", "uid")
		if err != nil {
			return nil, err
		}
		projectid := x.(string)

		// Check project auth
		if err = auth.CheckProjectAuth(uctx, projectid); err != nil {
			return nil, err
		}
	}

	// Forward query
	data, err := next(ctx)
	if err != nil {
		return data, err
	}
	d := data.(*model.AddProjectCardPayload)
	if d == nil {
		return nil, LogErr("add ProjectCard", fmt.Errorf("no card added"))
	}

	// Post-processing:
	// - Shift card position in columns list

	for _, card := range d.ProjectCard {
		if card.ID == "" {
			return data, fmt.Errorf("id payload required for project card mutation")
		}
		_, err := db.GetDB().Meta("incrementCardPos", map[string]string{"cardid": card.ID, "now": Now()})
		if err != nil {
			return data, err
		}
	}

	return data, err
}

// Add "ProjectCard"
func deleteProjectCardHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
	// Pre-processing:
	// - get values prior mutations
	// - Auth

	// Get User context
	ctx, uctx, err := auth.GetUserContext(ctx)
	if err != nil {
		return nil, LogErr("Access denied", err)
	}

	// Get input
	var filter model.ProjectCardFilter
	ExtractFilter(ctx, &filter)
	if len(filter.ID) == 0 {
		return nil, fmt.Errorf("Query requires id filters.")
	}
	// Prior to remove, get information about that object for post-processing
	oldCards := []ProjectCardLoc{}
	for _, uid := range filter.ID {
		card := ProjectCardLoc{}
		err := db.GetDB().Gamma1(QueryCardLoc, map[string]string{"cardid": uid}, &card)
		if err != nil {
			return nil, err
		}
		oldCards = append(oldCards, card)

		// Check project auth
		if err = auth.CheckProjectAuth(uctx, card.Projectid); err != nil {
			return nil, err
		}
	}

	// Forward query
	data, err := next(ctx)
	if err != nil {
		return data, err
	}
	d := data.(*model.DeleteProjectCardPayload)
	if d == nil {
		return nil, LogErr("delete ProjectCard", fmt.Errorf("no card deleted"))
	}

	// Post-processing:
	// - shift card positions in columns list
	// - eventually delete draft

	for _, card := range d.ProjectCard {
		if card.ID == "" {
			return data, fmt.Errorf("id payload required for project card mutation")
		}
		// Search for ids that as been actually deleted
		cardLoc, ok := Find(oldCards, func(c ProjectCardLoc) bool {
			return c.ID == card.ID
		})
		if !ok {
			log.Printf("Error: ProjectCard loc not found for card: %s", card.ID)
			continue
		}
		// Shift card position
		_, err := db.GetDB().Meta("decrementCardPos", map[string]string{
			"pos":   strconv.Itoa(card.Pos),
			"colid": cardLoc.Colid,
			"tid":   cardLoc.Contentid,
		})
		if err != nil {
			return data, err
		}
		if l := IndexOf(cardLoc.Typenames, "ProjectDraft"); l >= 0 {
			// Delete draft
			_, err := db.GetDB().Meta("deleteCardDraft", map[string]string{"cardid": card.ID})
			if err != nil {
				return data, err
			}
		}
	}

	return data, err
}

// Update "ProjectCard"
// @warning: update of card position only supported for one card at a time.
func updateProjectCardHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
	// Pre-processing:
	// - Auth

	// Get User context
	ctx, uctx, err := auth.GetUserContext(ctx)
	if err != nil {
		return nil, LogErr("Access denied", err)
	}

	// Get input
	var input model.UpdateProjectCardInput
	ExtractInput(ctx, &input)
	isMoved := false
	oldCard := ProjectCardLoc{}
	if input.Set != nil && len(input.Filter.ID) == 1 {
		id := input.Filter.ID[0]
		projectid := ""
		if input.Set.Pos != nil && input.Set.Pc != nil {
			// Extract the value before moving
			isMoved = true
			err := db.GetDB().Gamma1(QueryCardLoc, map[string]string{"cardid": id}, &oldCard)
			if err != nil {
				return nil, err
			}
			projectid = oldCard.Projectid
		} else {
			x, err := db.GetDB().GetSubFieldById(id, "ProjectColumn.project", "uid")
			if err != nil {
				return nil, err
			}
			projectid = x.(string)
		}

		// Check project auth
		if err = auth.CheckProjectAuth(uctx, projectid); err != nil {
			return nil, err
		}
	} else {
		// Review Auth + auto increment
		return nil, fmt.Errorf("Not implemented")
	}

	if input.Remove != nil {
		return nil, fmt.Errorf("remove is not allowed for this mutation")
	}

	// Forward query
	data, err := next(ctx)
	if err != nil {
		return data, err
	}

	// Post-processing:
	// - shift card positiun in columns list

	// Auto increment card position only when updating a single card,
	// otherwise, assume that user know what they are doing.
	if isMoved {
		newPos := *input.Set.Pos
		newColid := *input.Set.Pc.ID
		q := "moveCardPos"
		if newColid == oldCard.Colid {
			if oldCard.Pos > newPos {
				q = "moveCardPosUp"
			} else if newPos > oldCard.Pos {
				q = "moveCardPosDown"
			}
		}

		_, err := db.GetDB().Meta(q, map[string]string{
			"cardid":    oldCard.ID,
			"old_pos":   strconv.Itoa(oldCard.Pos),
			"old_colid": oldCard.Colid,
			"now":       Now(),
		})
		if err != nil {
			return data, err
		}
	}

	return data, err
}
