package math

import "sort"

// ReverseFloat64Slice is a type that implements the sort.Interface interface
// so that we can sort a slice of float64 in reverse order
type ReverseFloat64Slice []float64

var _ sort.Interface = ReverseFloat64Slice{}

// Len returns the length of the slice
func (r ReverseFloat64Slice) Len() int {
	return len(r)
}

// Less returns true if the element at index i is greater than the element at index j
func (r ReverseFloat64Slice) Less(i, j int) bool {
	return r[i] > r[j]
}

// Swap swaps the elements at indexes i and j
func (r ReverseFloat64Slice) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}
