package tools

import (
	"encoding/json"
)

// StructMap converts a value to another type via JSON round-trip.
func StructMap[Out any](in any) Out {
	var out Out
	raw, _ := json.Marshal(in)
	json.Unmarshal(raw, &out)
	return out
}

// ExtractSlice converts any slice-typed value to a typed []T via JSON round-trip.
func ExtractSlice[T any](a any, data *[]T) error {
	raw, err := json.Marshal(a)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, data)
}

// DerefSlice converts a slice of pointers []*T into a slice of values []T.
func DerefSlice[T any](ptrs []*T) []T {
	result := make([]T, len(ptrs))
	for i, p := range ptrs {
		result[i] = *p
	}
	return result
}

// InterfaceToSlice safely converts an any (expected to be []any)
// into a typed []T slice. Elements that don't match type T are skipped.
// Returns nil if in is nil or not a slice.
func InterfaceToSlice[T any](in any) []T {
	items, ok := in.([]any)
	if !ok {
		return nil
	}
	out := make([]T, 0, len(items))
	for _, v := range items {
		if typed, ok := v.(T); ok {
			out = append(out, typed)
		}
	}
	return out
}

// Find search an element in a slice based on a predicate. It returns element and true if element was found.
func Find[T any](collection []T, predicate func(item T) bool) (T, bool) {
	for _, item := range collection {
		if predicate(item) {
			return item, true
		}
	}

	var result T
	return result, false
}
