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

//
// Output / visibility directives
//
// These directives gate read access on the response side. They are wired in
// graph/resolver.go::Init() and operate on already-resolved fields.
//
//   @hidden  — the field is never exposed (used for password, token, …)
//   @private — only the owning user can read it
//   @x_ro    — read-only marker on input fields (rejected at write time)
//

// hidden refuses to expose a field that has been marked @hidden in the SDL.
func hidden(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
	rc := graphql.GetResolverContext(ctx)
	fieldName := rc.Field.Name
	return nil, fmt.Errorf("'%s' field is hidden", fieldName)
}

// private only releases the field when the requesting user owns the parent
// object. The owning username is taken either from the upstream context
// (set by setContextWith) or from a *model.User parent.
func private(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
	ctx, uctx, err := auth.GetUserContext(ctx)
	if err != nil {
		return nil, LogErr("Access denied", err)
	}

	rc := graphql.GetResolverContext(ctx)
	fieldName := rc.Field.Name

	// @DEBUG: not workng; ContextWith do not propagage value here, why, gqlgen !
	//         Probably because directive for returned value are not in the same context as of before ?
	// @AFTER_DEBUG: if uid is given, or other @id...
	if username, ok := ctx.Value("username").(string); ok && username == uctx.Username {
		return next(ctx)
	}

	switch v := obj.(type) {
	case *model.User:
		// @debug: username field required in graph
		if v.Username == uctx.Username {
			return next(ctx)
		}
	default:
		return nil, fmt.Errorf("Private directive not implemented for this field: %s", fieldName)
	}

	return nil, fmt.Errorf("'%s' field is private", fieldName)
}

// readOnly rejects any input write to a field tagged @x_ro / @x_patch_ro.
func readOnly(ctx context.Context, obj any, next graphql.Resolver) (any, error) {
	rc := graphql.GetResolverContext(ctx)
	pc := graphql.GetPathContext(ctx)
	queryName := rc.Field.Name
	fieldName := *pc.Field
	return nil, LogErr("Forbiden", fmt.Errorf("Read only field on %s:%s", queryName, fieldName))
}
