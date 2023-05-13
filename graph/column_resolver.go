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
	"log"
	"strconv"

	"github.com/99designs/gqlgen/graphql"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph/model"
	. "fractale/fractal6.go/tools"
)

// ProjectColumn Resolver
// --
// We update the column position when thery are
// moved to respect the shifting.

type ProjectColumnLoc struct {
	ID        string `json:"id"`
	Projectid string `json:"projectid"`
	Pos       int    `json:"pos"`
	CardsLen  int    `json:"cardslen"`
}

var QueryColumnLoc db.QueryMut = db.QueryMut{
	Q: `query {
            all(func: uid({{.colid}})) @normalize {
                uid
                ProjectColumn.project { projectid: uid  }
                pos: ProjectColumn.pos
                cardslen: count(ProjectColumn.cards)
            }
        }`,
}

// On remove column
var DecrementColumnPos db.QueryMut = db.QueryMut{
	Q: `query {
            var(func: uid({{.projectid}})) {
                Project.columns @filter(gt(ProjectColumn.pos, {{.pos}})) {
                    decrme as uid
                    p2 as ProjectColumn.pos
                    new_pos_decr as math(p2 - 1)
                }
            }
        }`,
	M: []db.X{db.X{
		S: `uid(decrme) <ProjectColumn.pos> val(new_pos_decr) . `,
	}},
}

// On update column up
var MoveColumnPosUp db.QueryMut = db.QueryMut{
	Q: `query {
            var(func: uid({{.colid}})) {
                pos as ProjectColumn.pos
            }

            var(func: uid({{.projectid}})) {
                Project.columns @filter(ge(ProjectColumn.pos, val(pos)) AND lt(ProjectColumn.pos, {{.old_pos}}) AND not uid({{.colid}})) {
                    incrme as uid
                    p1 as ProjectCard.pos
                    new_pos_incr as math(p1 + 1)
                }
            }
        }`,
	M: []db.X{
		db.X{
			S: `uid(incrme) <ProjectColumn.pos> val(new_pos_incr) . `,
		},
	},
}

// On update column down
var MoveColumnPosDown db.QueryMut = db.QueryMut{
	Q: `query {
            var(func: uid({{.colid}})) {
                pos as ProjectColumn.pos
            }

            var(func: uid({{.projectid}})) {
                ProjectColumn.cards @filter(gt(ProjectColumn.pos, {{.old_pos}}) AND le(ProjectColumn.pos, val(pos)) AND not uid({{.colid}})) {
                    decrme as uid
                    p1 as ProjectColumn.pos
                    new_pos_decr as math(p1 - 1)
                }
            }
        }`,
	M: []db.X{
		db.X{
			S: `uid(decrme) <ProjectColumn.pos> val(new_pos_decr) . `,
		},
	},
}

// Add "ProjectColumn"
func addProjectColumnHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
	// Pre-processing:
	// - Auth
	// - get values prior mutattions

	// Get User context
	//ctx, uctx, err := auth.GetUserContext(ctx)
	//if err != nil {
	//    return nil, LogErr("Access denied", err)
	//}

	// Forward query
	data, err := next(ctx)
	if err != nil {
		return data, err
	}
	d := data.(*model.AddProjectColumnPayload)
	if d == nil {
		return nil, LogErr("add ProjectColumn", fmt.Errorf("no col added"))
	}

	// Post-processing:

	return data, err
}

// Add "ProjectColumn"
func deleteProjectColumnHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
	// Pre-processing:
	// - Auth
	// - get values prior mutattions

	// Get input
	var filter model.ProjectColumnFilter
	ExtractFilter(ctx, &filter)
	if len(filter.ID) == 0 {
		return nil, fmt.Errorf("Delete project required id filters.")
	}
	// Prior to remove, get information about that object for post-processing
	oldColumns := []ProjectColumnLoc{}
	for _, uid := range filter.ID {
		col := ProjectColumnLoc{}
		err := db.GetDB().Gamma1(QueryColumnLoc, map[string]string{"colid": uid}, &col)
		if err != nil {
			return nil, err
		}
		if col.CardsLen > 0 {
			return nil, fmt.Errorf("This column has cards...please, move or remove it before deletion.")
		}
		oldColumns = append(oldColumns, col)
	}

	// Forward query
	data, err := next(ctx)
	if err != nil {
		return data, err
	}
	d := data.(*model.DeleteProjectColumnPayload)
	if d == nil {
		return nil, LogErr("delete ProjectColumn", fmt.Errorf("no column deleted"))
	}

	// Post-processing:
	// - shift column positions in project columns list

	for _, column := range d.ProjectColumn {
		if column.ID == "" {
			return data, fmt.Errorf("id payload required for project column mutation")
		}
		// Search for ids that as been actually deleted
		columnLoc, ok := Find(oldColumns, func(c ProjectColumnLoc) bool {
			return c.ID == column.ID
		})
		if !ok {
			log.Printf("Error: ProjectColumn loc not found for col: %s", column.ID)
			continue
		}
		// Shift column position
		_, err := db.GetDB().Gamma(DecrementColumnPos, map[string]string{
			"projectid": columnLoc.Projectid,
			"pos":       strconv.Itoa(column.Pos),
		})
		if err != nil {
			return data, err
		}
	}

	return data, err
}

// Update "ProjectColumn"
// @warning: update of col position only supported for one col at a time.
func updateProjectColumnHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
	// Pre-processing:
	// - Auth

	// Get input
	var input model.UpdateProjectColumnInput
	ExtractInput(ctx, &input)
	isMoved := false
	oldColumn := ProjectColumnLoc{}
	if input.Set != nil && len(input.Filter.ID) == 1 {
		if input.Set.Pos != nil {
			// Extract the value before moving
			isMoved = true
			id := input.Filter.ID[0]
			err := db.GetDB().Gamma1(QueryColumnLoc, map[string]string{"colid": id}, &oldColumn)
			if err != nil {
				return nil, err
			}
		}
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
	// - shift col positiun in columns list

	// Auto increment col position only when updating a single col,
	// otherwise, assume that user know what they are doing.
	if isMoved {
		newPos := *input.Set.Pos
		var q db.QueryMut
		if oldColumn.Pos > newPos {
			q = MoveColumnPosUp
		} else if newPos > oldColumn.Pos {
			q = MoveColumnPosDown
		}

		_, err := db.GetDB().Gamma(q, map[string]string{
			"projectid": oldColumn.Projectid,
			"colid":     oldColumn.ID,
			"old_pos":   strconv.Itoa(oldColumn.Pos),
		})
		if err != nil {
			return data, err
		}
	}

	return data, err
}
