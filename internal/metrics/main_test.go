// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package metrics

import (
	"crypto/rand"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	p, err := setup()
	if err != nil {
		fmt.Printf("failed to setup test: %v", err)
		os.Exit(42)
	}
	code := m.Run()
	err = teardown(p)
	if code == 0 && err != nil {
		code = 42
	}
	os.Exit(code)
}

func setup() (string, error) {
	suf := newRandomStringN(10)
	Path = "./" + suf

	_, err := os.Create(Path)
	if err != nil {
		return "", err
	}

	ReportInterval = 1 * time.Second
	AggregationInterval = 20 * time.Millisecond
	RetentionPeriod = 2 * time.Second

	return Path, nil
}

// teardown removes the cache directory.
func teardown(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("failed to remove test file: %v --- %v", path, err)
	}

	return nil
}

// newRandomStringN creates a new random string of length n.
func newRandomStringN(n int) string {
	randBytes := make([]byte, n/2)
	_, _ = rand.Read(randBytes)

	return fmt.Sprintf("%x", randBytes)
}
