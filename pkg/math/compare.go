// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License, Version 2.0.
package math

// Max64 returns the larger of x or y.
func Max64(x, y int64) int64 {
	if x > y {
		return x
	}
	return y
}

// Min64 returns the smaller of x or y.
func Min64(x, y int64) int64 {
	if x < y {
		return x
	}
	return y
}

// Min returns the smaller of x or y.
func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
