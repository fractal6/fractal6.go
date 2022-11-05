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
	"strings"
	"github.com/99designs/gqlgen/graphql"
)

var FieldTransformFunc map[string]func(context.Context, graphql.Resolver) (interface{}, error)

func init() {

    FieldTransformFunc = map[string]func(context.Context, graphql.Resolver) (interface{}, error){
        "lower": lower,
    }

}


func lower(ctx context.Context, next graphql.Resolver) (interface{}, error) {
    data, err := next(ctx)
    switch d := data.(type) {
    case *string:
        v := strings.ToLower(*d)
        return &v, err
    case string:
        v := strings.ToLower(d)
        return v, err
    }
    field := *graphql.GetPathContext(ctx).Field
    return nil, fmt.Errorf("Type unknwown for field %s", field)
}

