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

// DecodeDql decodes a DQL result into a typed value.
// raw can be map[string]any (single record) or []map[string]any (slice).
// Applies CleanCompositeName preprocessing before JSON decoding.
//
// Usage:
//
//	obj, err := DecodeDql[model.Tension](r.All[0])     // single record
//	data, err := DecodeDql[[]model.Node](r.All)         // slice
func DecodeDql[T any](raw any) (T, error) {
	var result T
	cleaned := cleanDqlRaw(raw)
	b, err := json.Marshal(cleaned)
	if err != nil {
		return result, err
	}
	err = json.Unmarshal(b, &result)
	return result, err
}

// CleanDqlMaps applies deep DQL name cleaning to a slice of raw DQL result maps.
// Strips composite prefixes ("Node.name" → "name"), renames "uid" → "id",
// and recursively cleans nested maps and arrays.
// Used by Meta() and Gamma() which return untyped maps to callers.
func CleanDqlMaps(all []map[string]any) []map[string]any {
	out := make([]map[string]any, len(all))
	for i, m := range all {
		out[i] = CleanCompositeName(m, true)
	}
	return out
}

// CleanDqlMap applies deep DQL name cleaning to a single raw DQL result map.
// Used by GetFieldBy*/GetSubFieldBy* which return cleaned maps for multi-field queries.
func CleanDqlMap(m map[string]any) map[string]any {
	return CleanCompositeName(m, true)
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

// CleanCompositeName keeps the last key string when separated by dot
// (eg [a.key.name: 10] -> [name: 10]) and replaces uid field with id
// (dgraph to gqlgen compatibility).
func CleanCompositeName(m map[string]any, deep bool) map[string]any {
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
			if deep {
				nv = CleanCompositeName(CleanAliasedMap(t), true)
			} else {
				nv = CleanAliasedMap(t)
			}
		case []any:
			for i, x := range t {
				if m, ok := x.(map[string]any); ok {
					t[i] = CleanCompositeName(CleanAliasedMap(m), true)
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

// cleanDqlRaw preprocesses DQL response data (single map or slice of maps).
func cleanDqlRaw(v any) any {
	switch t := v.(type) {
	case map[string]any:
		return CleanCompositeName(t, true)
	case []map[string]any:
		out := make([]map[string]any, len(t))
		for i, m := range t {
			out[i] = CleanCompositeName(m, true)
		}
		return out
	default:
		return v
	}
}
