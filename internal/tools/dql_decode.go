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
	"fmt"
	"regexp"
	"strings"
)

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
		return result, fmt.Errorf("DecodeDql: unsupported input type %T", raw)
	}
	b, err := json.Marshal(cleaned)
	if err != nil {
		return result, err
	}
	err = json.Unmarshal(b, &result)
	return result, err
}

// cleanSlice applies a map-cleaning function recursively to []any elements.
// Non-map elements are copied as-is.
func cleanSlice(s []any, fn func(map[string]any) map[string]any) []any {
	out := make([]any, len(s))
	for i, x := range s {
		if elem, ok := x.(map[string]any); ok {
			out[i] = fn(elem)
		} else {
			out[i] = x
		}
	}
	return out
}

// cleanDqlKey strips the "Type." prefix from a DQL field key and renames "uid" to "id".
//
//	cleanDqlKey("Node.nameid") → "nameid"
//	cleanDqlKey("uid")         → "id"
func cleanDqlKey(key string) string {
	if i := strings.LastIndex(key, "."); i >= 0 {
		key = key[i+1:]
	}
	if key == "uid" {
		key = "id"
	}
	return key
}

// CleanDqlMap applies deep DQL name cleaning to a raw DQL result map.
// Strips composite prefixes ("Node.name" → "name"), renames "uid" → "id",
// and recursively cleans nested maps and arrays.
func CleanDqlMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		nk := cleanDqlKey(k)

		var nv any
		switch t := v.(type) {
		case map[string]any:
			nv = CleanDqlMap(CleanAliasedMap(t))
		case []any:
			nv = cleanSlice(t, func(elem map[string]any) map[string]any {
				return CleanDqlMap(CleanAliasedMap(elem))
			})
		default:
			nv = t
		}
		out[nk] = nv
	}

	return out
}

var endDigits = regexp.MustCompile(`[0-9]+$`)

// CleanAliasedMap copies the input map by renaming all the keys
// recursively by removing trailing integers.
// @DEBUG: how to better handle aliasing (check gqlgen)
func CleanAliasedMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		nk := k
		if len(k) > 0 && IsDigit(k[len(k)-1]) {
			nk = endDigits.ReplaceAllString(k, "")
		}

		var nv any
		switch t := v.(type) {
		case map[string]any:
			nv = CleanAliasedMap(t)
		case []any:
			nv = cleanSlice(t, CleanAliasedMap)
		default:
			nv = t
		}
		out[nk] = nv
	}

	return out
}

// DecodeAt walks `path` from a single-result cleaned DQL map slice.
// Each path step is a "Type.field" token; the "Type." prefix is stripped
// internally. Intermediate edges may be cardinality-1 (map) or many ([]any),
// in which case the walk fans out and the result is a slice. The leaf step
// may be a space-separated list of fields ("uid Tension.title"); when so,
// the parent map is returned in lieu of a scalar.
//
// Returns (nil, nil) for empty results, nil intermediates, or missing keys.
// Returns an error when results contains more than one record (DQL contract:
// these helpers expect a single anchor row).
func DecodeAt(results []map[string]any, path ...string) (any, error) {
	if len(results) > 1 {
		return nil, fmt.Errorf("DecodeAt: got multiple results")
	}
	if len(results) == 0 {
		return nil, nil
	}
	if len(path) == 0 {
		return nil, fmt.Errorf("DecodeAt: empty path")
	}
	return walkPath(results[0], path)
}

func walkPath(node any, path []string) (any, error) {
	if node == nil {
		return nil, nil
	}
	step := path[0]
	isLeaf := len(path) == 1
	multi := isLeaf && len(strings.Fields(step)) > 1

	switch v := node.(type) {
	case map[string]any:
		if isLeaf && multi {
			return v, nil
		}
		next := v[cleanDqlKey(step)]
		if isLeaf {
			return next, nil
		}
		return walkPath(next, path[1:])
	case []any:
		out := make([]any, 0, len(v))
		for _, elem := range v {
			r, err := walkPath(elem, path)
			if err != nil {
				return nil, err
			}
			out = append(out, r)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("DecodeAt: unexpected type %T", v)
	}
}

// Dedupe removes duplicate items from a slice, keeping the first occurrence.
// The key function extracts the deduplication key from each item.
func Dedupe[T any](items []T, key func(T) string) []T {
	out := make([]T, 0, len(items))
	seen := make(map[string]bool)
	for _, item := range items {
		k := key(item)
		if !seen[k] {
			seen[k] = true
			out = append(out, item)
		}
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
