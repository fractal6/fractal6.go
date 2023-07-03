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

	"github.com/99designs/gqlgen/graphql"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph/model"
	. "fractale/fractal6.go/tools"
	"fractale/fractal6.go/web/auth"
)

var QueryDraft db.QueryMut = db.QueryMut{
	Q: `query {
            all(func: uid({{.id}})) {
                ProjectDraft.project_status {
                    ProjectColumn.project {
                        uid
                    }
                }
                Post.createdBy { User.username }
            }
        }`,
}

func updateProjectDraftHook(ctx context.Context, obj interface{}, next graphql.Resolver) (interface{}, error) {
	// Pre-processing:
	// - Auth (Author, or Project rights)

	// Get User context
	ctx, uctx, err := auth.GetUserContext(ctx)
	if err != nil {
		return nil, LogErr("Access denied", err)
	}

	// Get input
	var input model.UpdateProjectDraftInput
	ExtractInput(ctx, &input)
	if len(input.Filter.ID) == 0 {
		return nil, fmt.Errorf("Query requires id filters.")
	}

	for _, id := range input.Filter.ID {
		draft := model.ProjectDraft{}
		err := db.GetDB().Gamma1(QueryDraft, map[string]string{"id": id}, &draft)
		if err != nil {
			return nil, err
		}

		// Authorize author
		if uctx.Username == draft.CreatedBy.Username {
			continue
		}

		// Check project auth
		if err = auth.CheckProjectAuth(uctx, draft.ProjectStatus.Project.ID); err != nil {
			return nil, err
		}
	}

	// Forward Query
	return next(ctx)
}
