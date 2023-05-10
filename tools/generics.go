package tools

import (
	"encoding/json"
	"fmt"
	// "golang.org/x/exp/constraints"
)

// Use generics to feed a slice of unknow Type T from a list of map.
func ExtractSlice[T any](a interface{}, data *[]T) error {
	elements, ok := InterfaceSlice(a)
	if !ok {
		return fmt.Errorf("Input is not a slice")
	}

	for _, e := range elements {
		//temp := new(T)
		//StructMap(e, temp)
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

// Merge - receives slices of type T and merges them into a single slice of type T.
func Merge[T any](slices ...[]T) (mergedSlice []T) {
	for _, slice := range slices {
		for _, el := range slice {
			mergedSlice = append(mergedSlice, el)
		}
	}
	return mergedSlice
}

// Find - given a slice of type T, executes the passed in predicate function for each element in the slice.
// If the predicate returns true - a pointer to the element is returned. If no element is found, nil is returned.
// The function is passed the current element, the current index and the slice itself as function arguments.
func Find[T any](slice []T, predicate func(value T, index int, slice []T) bool) *T {
	for i, el := range slice {
		if ok := predicate(el, i, slice); ok {
			return &el
		}
	}
	return nil
}

// FindIndexOf - given a slice of type T and a value of type T, return ths first index of an element equal to value.
// If no element is found, -1 is returned.
func FindIndexOf[T comparable](slice []T, value T) int {
	for i, el := range slice {
		if el == value {
			return i
		}
	}
	return -1
}
