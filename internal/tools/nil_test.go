package tools_test

import (
	"testing"
)

// TestNilSliceLen verifies that calling len() on a nil slice returns 0
func TestNilSliceLen(t *testing.T) {
	var nilSlice []int = nil

	if nilSlice != nil {
		t.Error("expected slice to be nil")
	}

	if len(nilSlice) != 0 {
		t.Errorf("expected len(nil slice) to be 0, got %d", len(nilSlice))
	}

	// KEY INSIGHT: nilSlice is NOT a pointer!
	// Type of nilSlice: []int (a slice)
	// Value: nil (but it's a nil SLICE, not a nil pointer to a slice)
	//
	// Compare with pointer to slice:
	var ptrToSlice *[]int = nil
	// Type: *[]int (pointer to slice)
	// This is different! len(ptrToSlice) would NOT compile
	//
	// In Go, slices are already reference types (they contain a pointer internally)
	// So "nil slice" means the slice's internal pointer is nil, but the slice itself is a value
	_ = ptrToSlice // keep variable for documentation purposes

	// Verify we can safely append to nil slice
	nilSlice = append(nilSlice, 1, 2, 3)
	if len(nilSlice) != 3 {
		t.Errorf("expected len after append to be 3, got %d", len(nilSlice))
	}
}

// TestNilMapLen verifies that calling len() on a nil map returns 0
func TestNilMapLen(t *testing.T) {
	var nilMap map[string]int = nil

	if nilMap != nil {
		t.Error("expected map to be nil")
	}

	if len(nilMap) != 0 {
		t.Errorf("expected len(nil map) to be 0, got %d", len(nilMap))
	}

	// Reading from nil map returns zero value
	val := nilMap["key"]
	if val != 0 {
		t.Errorf("expected reading from nil map to return 0, got %d", val)
	}

	// Note: Writing to nil map would panic - this is expected behavior
}

// TestNilPointer verifies behavior of nil pointers
func TestNilPointer(t *testing.T) {
	var s string = ""
	sp := &s
	sp = nil

	if sp != nil {
		t.Error("expected pointer to be nil")
	}

	// len() cannot be called on pointers at all - it's a compile-time error
	// This would not compile: len(sp)
	// len() only works on: string, array, slice, map, channel

	// However, you CAN dereference a non-nil pointer and call len on the value
	sp2 := &s
	if len(*sp2) != 0 {
		t.Error("expected dereferenced empty string to have length 0")
	}

	// Dereferencing a nil pointer WILL panic at runtime
	// This is demonstrated in a separate test with panic recovery
}

// TestNilPointerDereferencePanics verifies that dereferencing nil pointer panics
func TestNilPointerDereferencePanics(t *testing.T) {
	var sp *string = nil

	// Attempting to dereference nil pointer should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when dereferencing nil pointer")
		}
	}()

	// This will panic
	_ = *sp
}

// TestSliceVsPointerToSlice demonstrates the critical difference
func TestSliceVsPointerToSlice(t *testing.T) {
	// Case 1: nil slice (this is a VALUE, not a pointer!)
	var nilSlice []int = nil
	// Type: []int
	// You CAN call len() on this
	if len(nilSlice) != 0 {
		t.Error("len() works on nil slice")
	}

	// Case 2: pointer to slice
	var ptrToSlice *[]int = nil
	// Type: *[]int
	// You CANNOT call len() on this - compile error
	// len(ptrToSlice) // <- would not compile!

	// But you CAN dereference a non-nil pointer to slice
	slice := []int{1, 2, 3}
	ptrToSlice = &slice
	if len(*ptrToSlice) != 3 {
		t.Error("can call len() on dereferenced pointer to slice")
	}

	// KEY INSIGHT: In Go, slices are ALREADY reference types!
	// A slice is actually a struct containing:
	// - pointer to underlying array
	// - length
	// - capacity
	// So []int is like a "smart pointer" - it's a value type that contains a pointer
	// That's why nil slice works with len() but nil *[]int doesn't
}

// TestNilSliceVsEmptySlice demonstrates the difference between nil and empty slices
func TestNilSliceVsEmptySlice(t *testing.T) {
	var nilSlice []int
	emptySlice := []int{}

	// Both have length 0
	if len(nilSlice) != 0 || len(emptySlice) != 0 {
		t.Error("both nil and empty slices should have length 0")
	}

	// But they are different
	if nilSlice != nil {
		t.Error("var declaration without initialization should be nil")
	}

	if emptySlice == nil {
		t.Error("literal initialization should not be nil")
	}
}

// TestNilMapVsEmptyMap demonstrates the difference between nil and empty maps
func TestNilMapVsEmptyMap(t *testing.T) {
	var nilMap map[string]int
	emptyMap := map[string]int{}

	// Both have length 0
	if len(nilMap) != 0 || len(emptyMap) != 0 {
		t.Error("both nil and empty maps should have length 0")
	}

	// But they are different
	if nilMap != nil {
		t.Error("var declaration without initialization should be nil")
	}

	if emptyMap == nil {
		t.Error("literal initialization should not be nil")
	}

	// Can write to empty map but not nil map
	emptyMap["key"] = 42
	if emptyMap["key"] != 42 {
		t.Error("should be able to write to empty map")
	}
}
