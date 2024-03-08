// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License, Version 2.0.
package math

import "fmt"

// Segments represents a range of segments.
type Segments struct {
	offset int64
	step   int
	size   int64
}

// Segment represents a single segment.
type Segment struct {
	Index  int64
	Offset int64
	Count  int
}

// NewSegments creates a new Segments object.
func NewSegments(offset int64, step int, count int64, size int64) (Segments, error) {
	if (step & (step - 1)) > 0 {
		return Segments{}, fmt.Errorf("step must be power of 2, got %d", step)
	}
	return Segments{offset, step, Min64(offset+count, size)}, nil
}

// AlignDown will align down the x by align. For example:
// AlignDown(1, 2) = 0
// AlignDown(29, 14) = 28
func AlignDown(x int64, align int64) int64 {
	return x / align * align
}

// All provides a channel of all segments.
func (r Segments) All() chan Segment {
	ch := make(chan Segment)
	go func() {
		for i := AlignDown(r.offset, int64(r.step)); i < r.size; i += int64(r.step) {
			absOffset := Max64(i, r.offset)
			seg := Segment{Index: i, Offset: absOffset - i}
			seg.Count = int(Min64(i+int64(r.step), r.size) - absOffset)
			if seg.Count > 0 {
				ch <- seg
			}
		}
		close(ch)
	}()
	return ch
}
