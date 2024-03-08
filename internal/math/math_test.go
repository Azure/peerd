// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License, Version 2.0.
package math

import "testing"

func TestRandomizedGroups(t *testing.T) {
	s := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	n := 2

	got := RandomizedGroups(s, n)

	if len(got) != 4 {
		t.Errorf("expected: %v, got: %v", 4, len(got))
	}

	for _, group := range got {
		if len(group) != 2 {
			t.Errorf("expected: %v, got: %v", 2, len(group))
		}

		for _, item := range group {
			found := false
			for i, element := range s {
				if item == element {
					found = true
					s[i] = "-1"
					break
				}
			}
			if !found {
				t.Errorf("element %v not found in original slice", item)
			}
		}
	}
}

func TestPercentilesFloat64Reverse(t *testing.T) {
	xs := []float64{5 * 1024 * 1024, 2 * 1024 * 1024, 1 * 1024 * 1024, 3 * 1024 * 1024, 4 * 1024 * 1024}
	ps := []float64{0.5, 0.9, 1.0}

	got := PercentilesFloat64Reverse(xs, ps...)
	if len(got) != 3 {
		t.Errorf("expected length: %v, got: %v", 3, len(got))
	}

	if got[0] != 3 {
		t.Errorf("expected p50: %v, got: %v ... %v", 3, got[0], got)
	}

	if got[1] != 2 {
		t.Errorf("expected p100: %v, got: %v ... %v", 2, got[1], got)
	}

	if got[2] != 1 {
		t.Errorf("expected p100: %v, got: %v ... %v", 1, got[2], got)
	}
}
