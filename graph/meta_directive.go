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
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/99designs/gqlgen/graphql"

	"fractale/fractal6.go/db"
	"fractale/fractal6.go/graph/model"
	. "fractale/fractal6.go/internal/tools"
)

// metaQuery resolves a single @meta(f:...) field. Each entry in metaRegistry
// closes over its target Go type at boot time and decodes the DQL response
// directly into it -- one JSON pass, no per-call reflection on the hot path.
type metaQuery func(maps map[string]string) (any, error)

// metaRegistry maps the @meta directive's `f` argument to a typed resolver.
// Adding a new @meta-backed schema field is a one-line addition here.
//
// DQL templates referenced by these entries MUST emit JSON keys that match
// the target type's `json:"..."` tags -- alias `Type.field` predicates in
// the template (e.g. `createdAt: Post.createdAt`) rather than relying on a
// post-hoc cleaning pass.
var metaRegistry = map[string]metaQuery{
	"getNodeHistory":  metaSlice[model.Event]("getNodeHistory"),
	"getNodeActivity": metaSlice[model.Activity]("getNodeActivity"),
	"getUserActivity": metaSlice[model.Activity]("getUserActivity"),
	"getEventCount":   metaScalar[model.EventCount]("getEventCount"),
}

// metaSlice returns a resolver that decodes the DQL `all` block into []*T.
func metaSlice[T any](name string) metaQuery {
	return func(maps map[string]string) (any, error) {
		res, err := db.GetDB().QueryDql(name, maps)
		if err != nil {
			return nil, err
		}
		if res == nil || len(res.Json) == 0 {
			return []*T{}, nil
		}
		var w struct {
			All []*T `json:"all"`
		}
		if err := json.Unmarshal(res.Json, &w); err != nil {
			return nil, err
		}
		if w.All == nil {
			return []*T{}, nil
		}
		return w.All, nil
	}
}

// metaScalar returns a resolver that decodes the first row of the `all` block
// into *T. Returns a zero-valued *T when the block is empty so the gqlgen type
// assertion (`tmp.(*T)`) always succeeds.
func metaScalar[T any](name string) metaQuery {
	return func(maps map[string]string) (any, error) {
		res, err := db.GetDB().QueryDql(name, maps)
		if err != nil {
			return nil, err
		}
		if res == nil || len(res.Json) == 0 {
			return new(T), nil
		}
		var w struct {
			All []*T `json:"all"`
		}
		if err := json.Unmarshal(res.Json, &w); err != nil {
			return nil, err
		}
		if len(w.All) == 0 || w.All[0] == nil {
			return new(T), nil
		}
		return w.All[0], nil
	}
}

// meta is the @meta directive entrypoint. It builds the DQL parameter map
// from the parent object + field arguments, then dispatches to the typed
// resolver registered under `f`.
func meta(ctx context.Context, obj any, next graphql.Resolver, f string, k []string) (any, error) {
	if _, err := next(ctx); err != nil {
		return nil, err
	}

	resolver, ok := metaRegistry[f]
	if !ok {
		return nil, fmt.Errorf("@meta: unknown function %q", f)
	}

	maps, err := buildMetaArgs(ctx, obj, k)
	if err != nil {
		return nil, err
	}

	return resolver(maps)
}

// buildMetaArgs collects DQL template parameters from two sources:
//   - keys named in the directive's `k` list, looked up first in the request
//     context, then by reflection on the parent object;
//   - field arguments declared in the GraphQL schema (e.g. activity(from,to)).
//
// The first key in `k` is the primary key and must resolve to a non-empty
// value; the rest are optional.
func buildMetaArgs(ctx context.Context, obj any, k []string) (map[string]string, error) {
	maps := map[string]string{}

	for i, key := range k {
		v, ok := ctx.Value(key).(string)
		if !ok {
			o := reflect.ValueOf(obj).Elem().FieldByName(ToGoNameFormat(key))
			if !o.IsValid() {
				if i == 0 {
					rc := graphql.GetResolverContext(ctx)
					return nil, fmt.Errorf("'%s' field on '%s' seems not valid or unknown", key, rc.Field.Name)
				}
				continue
			}
			if o.Kind() == reflect.Ptr {
				if o.IsNil() {
					continue
				}
				v = o.Elem().String()
			} else {
				v = o.String()
			}
		}

		if v == "" && i == 0 {
			rc := graphql.GetResolverContext(ctx)
			return nil, fmt.Errorf("'%s' field is needed to query '%s'", key, rc.Field.Name)
		}
		if v != "" {
			maps[key] = v
		}
	}

	if fc := graphql.GetFieldContext(ctx); fc != nil {
		for argName, argVal := range fc.Args {
			switch v := argVal.(type) {
			case string:
				if v != "" {
					maps[argName] = v
				}
			case *string:
				if v != nil && *v != "" {
					maps[argName] = *v
				}
			}
		}
	}

	return maps, nil
}
