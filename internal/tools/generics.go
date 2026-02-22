package tools

import (
	"encoding/json"
	"fmt"
)

// Use generics to feed a slice of unknow Type T from a list of map.
func ExtractSlice[T any](a any, data *[]T) error {
	elements, ok := InterfaceSlice(a)
	if !ok {
		return fmt.Errorf("Input is not a slice")
	}

	for _, e := range elements {
		// temp := new(T)
		// StructMap(e, temp)
		raw, err := json.Marshal(e)
		if err != nil {
			return err
		}
		temp, _ := unmarshalAny[T](raw)
		*data = append(*data, *temp)
	}

	return nil
}

func unmarshalAny[T any](bytes []byte) (*T, error) {
	out := new(T)
	if err := json.Unmarshal(bytes, out); err != nil {
		return nil, err
	}
	return out, nil
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

//
// COMMON
//

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
