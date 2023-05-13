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

// IndexOf returns the index at which the first occurrence of a value is found in an array or return -1
// if the value cannot be found.
func IndexOf[T comparable](collection []T, element T) int {
	for i, item := range collection {
		if item == element {
			return i
		}
	}

	return -1
}

// LastIndexOf returns the index at which the last occurrence of a value is found in an array or return -1
// if the value cannot be found.
func LastIndexOf[T comparable](collection []T, element T) int {
	length := len(collection)

	for i := length - 1; i >= 0; i-- {
		if collection[i] == element {
			return i
		}
	}

	return -1
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

// FindIndexOf searches an element in a slice based on a predicate and returns the index and true.
// It returns -1 and false if the element is not found.
func FindIndexOf[T any](collection []T, predicate func(item T) bool) (T, int, bool) {
	for i, item := range collection {
		if predicate(item) {
			return item, i, true
		}
	}

	var result T
	return result, -1, false
}

// FindLastIndexOf searches last element in a slice based on a predicate and returns the index and true.
// It returns -1 and false if the element is not found.
func FindLastIndexOf[T any](collection []T, predicate func(item T) bool) (T, int, bool) {
	length := len(collection)

	for i := length - 1; i >= 0; i-- {
		if predicate(collection[i]) {
			return collection[i], i, true
		}
	}

	var result T
	return result, -1, false
}

// FindOrElse search an element in a slice based on a predicate. It returns the element if found or a given fallback value otherwise.
func FindOrElse[T any](collection []T, fallback T, predicate func(item T) bool) T {
	for _, item := range collection {
		if predicate(item) {
			return item
		}
	}

	return fallback
}
