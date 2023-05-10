package tools

import (
// "golang.org/x/exp/constraints"
)

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
