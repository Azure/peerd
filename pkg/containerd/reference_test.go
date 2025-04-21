// Initial Copyright (c) 2023 Xenit AB and 2024 The Spegel Authors.
// Portions Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package containerd

import (
	"fmt"
	"testing"

	digest "github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"
)

func TestParseReference(t *testing.T) {
	tests := []struct {
		name               string
		image              string
		digestInImage      bool
		expectedRepository string
		expectedTag        string
		expectedDigest     digest.Digest
	}{
		{
			name:               "Latest tag",
			image:              "library/ubuntu:latest",
			digestInImage:      false,
			expectedRepository: "library/ubuntu",
			expectedTag:        "latest",
			expectedDigest:     digest.Digest("sha256:c0669ef34cdc14332c0f1ab0c2c01acb91d96014b172f1a76f3a39e63d1f0bda"),
		},
		{
			name:               "Only tag",
			image:              "library/alpine:3.18.0",
			digestInImage:      false,
			expectedRepository: "library/alpine",
			expectedTag:        "3.18.0",
			expectedDigest:     digest.Digest("sha256:c0669ef34cdc14332c0f1ab0c2c01acb91d96014b172f1a76f3a39e63d1f0bda"),
		},
		{
			name:               "Tag and digest",
			image:              "jetstack/cert-manager-controller:3.18.0@sha256:c0669ef34cdc14332c0f1ab0c2c01acb91d96014b172f1a76f3a39e63d1f0bda",
			digestInImage:      true,
			expectedRepository: "jetstack/cert-manager-controller",
			expectedTag:        "3.18.0",
			expectedDigest:     digest.Digest("sha256:c0669ef34cdc14332c0f1ab0c2c01acb91d96014b172f1a76f3a39e63d1f0bda"),
		},
		{
			name:               "Only digest",
			image:              "fluxcd/helm-controller@sha256:c0669ef34cdc14332c0f1ab0c2c01acb91d96014b172f1a76f3a39e63d1f0bda",
			digestInImage:      true,
			expectedRepository: "fluxcd/helm-controller",
			expectedTag:        "",
			expectedDigest:     digest.Digest("sha256:c0669ef34cdc14332c0f1ab0c2c01acb91d96014b172f1a76f3a39e63d1f0bda"),
		},
	}
	registries := []string{"docker.io", "quay.io", "ghcr.com", "127.0.0.1"}
	for _, registry := range registries {
		for _, tt := range tests {
			t.Run(fmt.Sprintf("%s_%s", tt.name, registry), func(t *testing.T) {
				for _, targetDigest := range []string{tt.expectedDigest.String(), ""} {
					ref, err := ParseReference(fmt.Sprintf("%s/%s", registry, tt.image), digest.Digest(targetDigest))
					if !tt.digestInImage && targetDigest == "" {
						require.EqualError(t, err, "invalid digest: ")
						continue
					}
					require.NoError(t, err)
					require.Equal(t, registry, ref.Host())
					require.Equal(t, tt.expectedRepository, ref.Repository())
					require.Equal(t, tt.expectedTag, ref.Tag())
					require.Equal(t, tt.expectedDigest, ref.Digest())
				}
			})

		}
	}
}

func TestParseImageDigestDoesNotMatch(t *testing.T) {
	_, err := ParseReference("quay.io/jetstack/cert-manager-webhook@sha256:13fd9eaadb4e491ef0e1d82de60cb199f5ad2ea5a3f8e0c19fdf31d91175b9cb", digest.Digest("sha256:ec4306b243d98cce7c3b1f994f2dae660059ef521b2b24588cfdc950bd816d4c"))
	require.EqualError(t, err, "invalid digest, target does not match parsed digest: quay.io/jetstack/cert-manager-webhook@sha256:13fd9eaadb4e491ef0e1d82de60cb199f5ad2ea5a3f8e0c19fdf31d91175b9cb sha256:13fd9eaadb4e491ef0e1d82de60cb199f5ad2ea5a3f8e0c19fdf31d91175b9cb")
}

func TestParseImageNoTagOrDigest(t *testing.T) {
	_, err := ParseReference("ghcr.io/xenitab/spegel", digest.Digest(""))
	require.EqualError(t, err, "invalid digest: ")
}

func TestString(t *testing.T) {
	got, err := ParseReference("jetstack/cert-manager-controller:3.18.0@sha256:c0669ef34cdc14332c0f1ab0c2c01acb91d96014b172f1a76f3a39e63d1f0bda", digest.Digest("sha256:c0669ef34cdc14332c0f1ab0c2c01acb91d96014b172f1a76f3a39e63d1f0bda"))
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, "jetstack/cert-manager-controller:3.18.0", got.String())
	require.Equal(t, "jetstack/cert-manager-controller:3.18.0@sha256:c0669ef34cdc14332c0f1ab0c2c01acb91d96014b172f1a76f3a39e63d1f0bda", got.Name())
}
