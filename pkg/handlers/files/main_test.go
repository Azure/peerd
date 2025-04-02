// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package files

import (
	"crypto/rand"
	"fmt"
	"os"
	"testing"
)

var testFileCachePath string

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	err := teardown()
	if code == 0 && err != nil {
		code = 42
	}
	os.Exit(code)
}

func setup() {
	cwd, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("failed to get current working directory: %v", err))
	}

	testFileCachePath = cwd + newRandomStringN(10)
}

// teardown removes the cache directory.
func teardown() error {
	if err := os.RemoveAll(testFileCachePath); err != nil {
		return fmt.Errorf("failed to remove cache dir: %v --- %v", testFileCachePath, err)
	}

	return nil
}

// newRandomStringN creates a new random string of length n.
func newRandomStringN(n int) string {
	randBytes := make([]byte, n/2)
	_, _ = rand.Read(randBytes)

	return fmt.Sprintf("%x", randBytes)
}
