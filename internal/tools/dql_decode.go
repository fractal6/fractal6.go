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

package tools

import (
	"encoding/json"
	"regexp"
	"strings"
)

// DecodeDqlSlice converts a slice of raw DQL maps into typed values using DecodeDql.
func DecodeDqlSlice[T any](results []map[string]any) ([]T, error) {
	out := make([]T, 0, len(results))
	for _, m := range results {
		v, err := DecodeDql[T](m)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// DecodeDql decodes a DQL result into a typed value.
// raw can be map[string]any (single record) or []map[string]any (slice).
// Applies CleanDqlMap preprocessing before JSON decoding.
//
// Usage:
//
//	obj, err := DecodeDql[model.Tension](r.All[0])     // single record
//	data, err := DecodeDql[[]model.Node](r.All)         // slice
func DecodeDql[T any](raw any) (T, error) {
	var result T
	var cleaned any
	switch t := raw.(type) {
	case map[string]any:
		cleaned = CleanDqlMap(t)
	case []map[string]any:
		out := make([]map[string]any, len(t))
		for i, m := range t {
			out[i] = CleanDqlMap(m)
		}
		cleaned = out
	default:
		cleaned = raw
	}
	b, err := json.Marshal(cleaned)
	if err != nil {
		return result, err
	}
	err = json.Unmarshal(b, &result)
	return result, err
}

// CleanDqlMap applies deep DQL name cleaning to a raw DQL result map.
// Strips composite prefixes ("Node.name" → "name"), renames "uid" → "id",
// and recursively cleans nested maps and arrays.
func CleanDqlMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		ks := strings.Split(k, ".")
		nk := ks[len(ks)-1]

		if nk == "uid" {
			nk = "id"
		}

		var nv any
		switch t := v.(type) {
		case map[string]any:
			nv = CleanDqlMap(CleanAliasedMap(t))
		case []any:
			for i, x := range t {
				if m, ok := x.(map[string]any); ok {
					t[i] = CleanDqlMap(CleanAliasedMap(m))
				}
			}
			nv = t
		default:
			nv = t
		}
		out[nk] = nv
	}

	return out
}

var endDigits = regexp.MustCompile(`[0-9]+$`)

// CleanAliasedMap copy the input map by renaming all the keys
// recursively by removing trailing integers.
// @DEBUG: how to better handle aliasing (check gqlgen)
func CleanAliasedMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		var nk string
		if IsDigit(k[len(k)-1]) {
			nk = endDigits.ReplaceAllString(k, "")
		} else {
			nk = k
		}

		var nv any
		switch t := v.(type) {
		case map[string]any:
			nv = CleanAliasedMap(t)
		case []any:
			for i, x := range t {
				if m, ok := x.(map[string]any); ok {
					t[i] = CleanAliasedMap(m)
				}
			}
			nv = t
		default:
			nv = t
		}
		out[nk] = nv
	}

	return out
}

// First extracts the first element from a slice, or zero value if empty.
// Composes with functions returning ([]T, error) like Meta and Gamma.
func First[T any](items []T, err error) (T, error) {
	var zero T
	if err != nil {
		return zero, err
	}
	if len(items) == 0 {
		return zero, nil
	}
	return items[0], nil
}
