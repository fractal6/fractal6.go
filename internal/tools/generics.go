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
