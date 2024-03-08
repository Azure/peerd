package math

import (
	"crypto/rand"
	"math/big"
	"sort"
)

// PercentilesFloat64Reverse calculates the percentile of a slice of floats in reverse order.
// NOTE: The unit of each value of xs is 'bits' and the result is 'Mb'.
func PercentilesFloat64Reverse(xs []float64, ps ...float64) []float64 {
	if len(xs) == 0 {
		return nil
	}

	// Sort in descending order
	sort.Sort(ReverseFloat64Slice(xs))
	results := []float64{}

	for _, p := range ps {
		if p < 0 {
			p = 0
		}
		if p > 1 {
			p = 1
		}

		i := int(float64(len(xs)-1) * p)
		results = append(results, xs[i]/1024/1024)
	}

	return results
}

// RandomizedGroups groups the given collection randomly into groups of size n
func RandomizedGroups(s []string, n int) [][]string {
	groups := make([][]string, 0)
	numGroups := len(s) / n
	if len(s)%n != 0 {
		numGroups++
	}

	// Shuffle the slice
	for i := range s {
		j, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			panic(err)
		}
		s[i], s[j.Int64()] = s[j.Int64()], s[i]
	}

	// Create groups
	for i := 0; i < numGroups; i++ {
		group := make([]string, 0)
		for j := 0; j < n && i*n+j < len(s); j++ {
			group = append(group, s[i*n+j])
		}
		groups = append(groups, group)
	}

	return groups
}
