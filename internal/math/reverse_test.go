// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package math

import (
	"sort"
	"testing"
)

func TestReverseSort(t *testing.T) {
	data := []float64{1, 2, 3, 4, 5}

	sort.Sort(ReverseFloat64Slice(data))

	for i, v := range data {

		if v != float64(5-i) {
			t.Errorf("expected: %v, got: %v", float64(5-i), v)
		}
	}
}

func TestLen(t *testing.T) {
	data := []float64{1, 2, 3, 4, 5}

	r := ReverseFloat64Slice(data)

	if r.Len() != 5 {
		t.Errorf("expected: %v, got: %v", 5, r.Len())
	}
}

func TestLess(t *testing.T) {
	data := []float64{1, 2, 3, 4, 5}

	r := ReverseFloat64Slice(data)
	res := r.Less(0, 1)

	if res != false {
		t.Errorf("expected: %v, got: %v", false, res)
	}

	res = r.Less(1, 0)

	if res != true {
		t.Errorf("expected: %v, got: %v", true, res)
	}
}

func TestSwap(t *testing.T) {
	data := []float64{1, 2, 3, 4, 5}

	r := ReverseFloat64Slice(data)
	r.Swap(0, 1)

	if r[0] != 2 || r[1] != 1 {
		t.Errorf("expected: %v, got: %v", []float64{2, 1}, r[:2])
	}
}
