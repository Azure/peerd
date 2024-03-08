// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License, Version 2.0.
package cache

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"testing"
)

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
	suf := newRandomStringN(10)
	Path += suf
}

// teardown removes the cache directory.
func teardown() error {
	if err := os.RemoveAll(Path); err != nil {
		return fmt.Errorf("failed to remove cache dir: %v --- %v", Path, err)
	}

	return nil
}

// newRandomString creates a new random string.
func newRandomString() string {
	const blockSize = 1024 * 1024
	r, err := rand.Int(rand.Reader, big.NewInt(4))
	if err != nil {
		panic(err)
	}
	length := r.Int64() * blockSize

	r, err = rand.Int(rand.Reader, big.NewInt(blockSize))
	if err != nil {
		panic(err)
	}
	length += r.Int64()

	return newRandomStringN(int(length))
}

// newRandomStringN creates a new random string of length n.
func newRandomStringN(n int) string {
	randBytes := make([]byte, n/2)
	_, _ = rand.Read(randBytes)

	return fmt.Sprintf("%x", randBytes)
}
