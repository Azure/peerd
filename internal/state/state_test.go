// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package state

import (
	"context"
	"testing"
	"time"

	ocitests "github.com/azure/peerd/internal/oci/store/tests"
	"github.com/azure/peerd/internal/routing/tests"
	"github.com/azure/peerd/pkg/containerd"
	"github.com/stretchr/testify/require"
)

// TestContainerdStoreAds is a unit test function that tests the basic functionality of the Advertise function.
// It creates a list of container image references, initializes a mock containerd store and a mock router,
// and then calls the Advertise function with the created context, router, containerd store, and an empty file channel.
// After that, it verifies that the router correctly looks up the peers for each reference.
func TestContainerdStoreAds(t *testing.T) {
	refsStr := []string{
		"docker.io/library/ubuntu:latest@sha256:b060fffe8e1561c9c3e6dea6db487b900100fc26830b9ea2ec966c151ab4c020",
		"ghcr.io/xenitab/spegel:v0.0.9@sha256:fa32bd3bcd49a45a62cfc1b0fed6a0b63bf8af95db5bad7ec22865aee0a4b795",
		"docker.io/library/alpine@sha256:25fad2a32ad1f6f510e528448ae1ec69a28ef81916a004d3629874104f8a7f70",
	}

	refs := []containerd.Reference{}
	for _, refStr := range refsStr {
		img, err := containerd.ParseReference(refStr, "")
		require.NoError(t, err)
		refs = append(refs, img)
	}

	containerdStore := ocitests.NewMockContainerdStore(refs)
	router := tests.NewMockRouter(map[string][]string{})

	ctx, cancel := context.WithCancel(context.TODO())
	go func() {
		time.Sleep(2 * time.Second)
		cancel()
	}()

	Advertise(ctx, router, containerdStore, make(<-chan string)) // TODO avtakkar: add tests for file chan

	for _, ref := range refs {
		peers, ok := router.LookupKey(ref.Digest().String())
		require.True(t, ok)
		require.Len(t, peers, 1)

		if ref.Tag() != "" {
			peers, ok = router.LookupKey(ref.String())
			require.True(t, ok)
			require.Len(t, peers, 1)
		}
	}
}
