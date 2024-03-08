package math

import "testing"

func TestNewSegments(t *testing.T) {
	_, err := NewSegments(0, 3, 100, 100)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAlignDown(t *testing.T) {
	for _, testcase := range []struct {
		x        int64
		align    int64
		expected int64
	}{
		{
			x:        1,
			align:    2,
			expected: 0,
		},
		{
			x:        29,
			align:    14,
			expected: 28,
		},
		{
			x:        0,
			align:    2,
			expected: 0,
		},
		{
			x:        2,
			align:    2,
			expected: 2,
		},
		{
			x:        2147483647,
			align:    2,
			expected: 2147483646,
		},
		{
			x:        2147483647,
			align:    4,
			expected: 2147483644,
		},
		{
			x:        2147483647,
			align:    8,
			expected: 2147483640,
		},
		{
			x:        2147483647,
			align:    16,
			expected: 2147483632,
		},
		{
			x:        2147483647,
			align:    32,
			expected: 2147483616,
		},
	} {
		got := AlignDown(testcase.x, testcase.align)

		if got != testcase.expected {
			t.Errorf("expected: %v, got: %v", testcase.expected, got)
		}
	}
}

func TestAll(t *testing.T) {
	for _, testcase := range []struct {
		offset   int64
		step     int
		count    int64
		size     int64
		expected []Segment
	}{
		{
			offset: 0,
			step:   4,
			count:  10,
			size:   10,
			expected: []Segment{
				{Index: 0, Offset: 0, Count: 4},
				{Index: 4, Offset: 0, Count: 4},
				{Index: 8, Offset: 0, Count: 2},
			},
		},
		{
			offset: 3,
			step:   2,
			count:  9,
			size:   15,
			expected: []Segment{
				{Index: 2, Offset: 1, Count: 1},
				{Index: 4, Offset: 0, Count: 2},
				{Index: 6, Offset: 0, Count: 2},
				{Index: 8, Offset: 0, Count: 2},
				{Index: 10, Offset: 0, Count: 2},
			},
		},
		{
			offset: 0,
			step:   4,
			count:  10,
			size:   2147483647,
			expected: []Segment{
				{Index: 0, Offset: 0, Count: 4},
				{Index: 4, Offset: 0, Count: 4},
				{Index: 8, Offset: 0, Count: 2},
			},
		},
		{
			offset: 3,
			step:   2,
			count:  9,
			size:   2147483647,
			expected: []Segment{
				{Index: 2, Offset: 1, Count: 1},
				{Index: 4, Offset: 0, Count: 2},
				{Index: 6, Offset: 0, Count: 2},
				{Index: 8, Offset: 0, Count: 2},
				{Index: 10, Offset: 0, Count: 2},
			},
		},
	} {
		segs, err := NewSegments(testcase.offset, testcase.step, testcase.count, testcase.size)
		if err != nil {
			t.Error(err)
		}

		i := 0
		for seg := range segs.All() {
			expected := testcase.expected[i]
			if expected != seg {
				t.Errorf("expected: %v, got: %v", expected, seg)
			}
			i++
		}
	}
}
