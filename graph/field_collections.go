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
	"reflect"
	"strings"

	"fractale/fractal6.go/graph/model"
	"fractale/fractal6.go/tools"
)

//
// Misc field utils
//

func queryTypeFromGraphqlContext(ctx context.Context) (string, string, string, error) {
	var err error
	var ok bool
	var queryType, typeName, queryName string

	rc := graphql.GetResolverContext(ctx)
	qName := tools.SplitCamelCase(rc.Field.Name)
	if len(qName) < 2 {
		return queryType, typeName, "", fmt.Errorf("query type name unknown")
	}
	queryType = qName[0]
	typeName = strings.Join(qName[1:], "")
	queryName = rc.Path().String()
	for _, t := range []string{"query", "get", "add", "update", "delete", "aggregate"} {
		ok = ok || (queryType == t)
	}
	if !ok {
		err = fmt.Errorf("query type name unknown")
	}
	return queryType, typeName, queryName, err
}

// setContext add the {n} field in the context for further inspection in next resolvers.
// Its used in the hook_ resolvers for Update and Delete queries.
func setContextWith(ctx context.Context, obj interface{}, n string) (context.Context, string, error) {
	var val string
	var err error
	var filter model.JsonAtom

	if obj.(model.JsonAtom)[n] != nil {
		// won't work since input directive apply on one argument only :S
		if val, ok := obj.(model.JsonAtom)[n].(string); ok {
			ctx = context.WithValue(ctx, n, val)
			return ctx, val, err

		} else {
			return ctx, val, err
		}
	} else if obj.(model.JsonAtom)["input"] != nil {
		if obj.(model.JsonAtom)["input"].(model.JsonAtom)["filter"] == nil {
			panic("add mutation not supported here.")
		}
		// Update mutation
		filter = obj.(model.JsonAtom)["input"].(model.JsonAtom)["filter"].(model.JsonAtom)
	} else if obj.(model.JsonAtom)["filter"] != nil {
		// Delete mutation
		filter = obj.(model.JsonAtom)["filter"].(model.JsonAtom)
	} else {
		return ctx, val, err
	}

	if filter[n] == nil {
		return ctx, val, err
	}

	switch n {
	case "nameid", "rootnameid", "username":
		v := filter[n].(model.JsonAtom)["eq"]
		if v != nil {
			val = v.(string)
		}
	case "id":
		ids := filter[n].([]interface{})
		if len(ids) != 1 {
			return ctx, val, fmt.Errorf("multiple ID is not allowed for this request.")
		}
		val = ids[0].(string)
	}

	ctx = context.WithValue(ctx, n, val)
	return ctx, val, err
}

func getNestedObj(obj interface{}, field string) interface{} {
	var source model.JsonAtom
	var target interface{}

	source = obj.(model.JsonAtom)
	fields := strings.Split(field, ".")

	for i, f := range fields {
		target = source[f]
		if target == nil {
			return nil
		}
		if i < len(fields)-1 {
			source = target.(model.JsonAtom)
		}
	}

	return target
}

func get(obj model.JsonAtom, field string, deflt interface{}) interface{} {
	v := obj[field]
	if v == nil {
		return deflt
	}

	return v
}

//
// qqlgen code to extract fields
//

func GetPreloads(ctx context.Context) []string {
	return GetNestedPreloads(
		graphql.GetRequestContext(ctx),
		graphql.CollectFieldsCtx(ctx, nil),
		"", true,
	)
}

func GetQueryGraph(ctx context.Context) string {
	return strings.Join(GetPreloads(ctx), " ")
}

func GetNestedPreloads(ctx *graphql.RequestContext, fields []graphql.CollectedField, prefix string, first bool) (preloads []string) {
	for _, f := range fields {
		//prefixColumn := GetPreloadString(prefix, f.Name)
		prefixColumn := f.Name
		preloads = append(preloads, prefixColumn)
		if len(f.Arguments) > 0 {
			preloads = append(preloads, "(")
			for i, a := range f.Arguments {
				if i > 0 {
					preloads = append(preloads, ",")
				}
				preloads = append(preloads, fmt.Sprintf("%s:%s", a.Name, a.Value))
			}
			preloads = append(preloads, ")")
		}
		if len(f.SelectionSet) > 0 {
			preloads = append(preloads, "{")
			preloads = append(preloads, GetNestedPreloads(ctx, graphql.CollectFields(ctx, f.SelectionSet, nil), prefixColumn, false)...)
			preloads = append(preloads, "}")
		}
	}
	return
}

func GetPreloadString(prefix, name string) string {
	var fname string = name
	return fname
}

// PayloadContains return true if the query payload contains the given field,
// by looking the raw json query (trough collected fields)
func PayloadContains(ctx context.Context, field string) bool {
	fields := graphql.CollectFieldsCtx(ctx, nil)[0]
	for _, c := range graphql.CollectFields(graphql.GetRequestContext(ctx), fields.SelectionSet, nil) {
		if c.Name == field {
			return true
		}
	}
	return false
}

// PayloadContains return true if the query payload contains the given field,
// by looking the reflected Go varaible. It is used for input hook when
// the payload is not available in the context.
func PayloadContainsGo(obj interface{}, field string) bool {
	n := reflect.ValueOf(obj).Elem().FieldByName(field).String()
	return n != ""
}
