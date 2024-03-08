// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License, Version 2.0.
package math

import "testing"

func TestMax64(t *testing.T) {
	for _, tc := range []struct {
		x, y int64
		want int64
	}{
		{1, 2, 2},
		{2, 1, 2},
		{1, 1, 1},
		{-1, 1, 1},
		{1000, 0, 1000},
	} {
		got := Max64(tc.x, tc.y)
		if got != tc.want {
			t.Errorf("expected: %v, got: %v", tc.want, got)
		}
	}
}

func TestMin64(t *testing.T) {
	for _, tc := range []struct {
		x, y int64
		want int64
	}{
		{1, 2, 1},
		{2, 1, 1},
		{1, 1, 1},
		{-1, 1, -1},
		{1000, 0, 0},
	} {
		got := Min64(tc.x, tc.y)
		if got != tc.want {
			t.Errorf("expected: %v, got: %v", tc.want, got)
		}
	}
}

func TestMin(t *testing.T) {
	for _, tc := range []struct {
		x, y int
		want int
	}{
		{1, 2, 1},
		{2, 1, 1},
		{1, 1, 1},
		{-1, 1, -1},
		{1000, 0, 0},
	} {
		got := Min(tc.x, tc.y)
		if got != tc.want {
			t.Errorf("expected: %v, got: %v", tc.want, got)
		}
	}
}
