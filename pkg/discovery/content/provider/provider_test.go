// Initial Copyright (c) 2023 Xenit AB and 2024 The Spegel Authors.
// Portions Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package provider

import (
	"context"
	"testing"
	"time"

	"github.com/azure/peerd/pkg/discovery/routing/mocks"
	"github.com/rs/zerolog"
)

func TestProvide_Success(t *testing.T) {
	ctx := zerolog.New(zerolog.NewTestWriter(t)).WithContext(context.Background())
	pMap := map[string][]string{}
	router := mocks.NewMockRouter(pMap)
	filesChan := make(chan string, 2)

	// Send test blobs
	testBlobs := []string{
		"sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		"sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855_0-1023",
	}

	go func() {
		defer close(filesChan)
		for _, blob := range testBlobs {
			filesChan <- blob
		}
	}()

	// Run Provide in a goroutine with timeout
	done := make(chan struct{})
	go func() {
		defer close(done)
		ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
		defer cancel()
		Provide(ctx, router, filesChan)
	}()

	// Wait for completion or timeout
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Provide did not complete within timeout")
	}

	// Verify pMap has both the expected blobs
	for _, blob := range testBlobs {
		if _, ok := pMap[blob]; !ok {
			t.Errorf("Expected blob %s to be provided, but it was not found", blob)
		}
	}
}

func TestProvide_ContextCancellation(t *testing.T) {
	ctx := zerolog.New(zerolog.NewTestWriter(t)).WithContext(context.Background())
	pMap := map[string][]string{}
	router := mocks.NewMockRouter(pMap)
	filesChan := make(chan string)

	ctx, cancel := context.WithCancel(ctx)

	// Start Provide in a goroutine
	done := make(chan struct{})
	go func() {
		defer close(done)
		Provide(ctx, router, filesChan)
	}()

	// Cancel context immediately
	cancel()

	// Should complete quickly
	select {
	case <-done:
		// Success - function returned due to context cancellation
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Provide did not exit after context cancellation")
	}

	// Verify no provide calls were made
	if len(pMap) != 0 {
		t.Errorf("Expected no blobs to be provided, but got %d", len(pMap))
	}
}
