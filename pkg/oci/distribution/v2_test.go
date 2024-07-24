// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package distribution

import (
	"testing"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"
)

func TestParsePathComponents(t *testing.T) {
	tests := []struct {
		name            string
		registry        string
		path            string
		expectedRef     string
		expectedDgst    digest.Digest
		expectedRefType ReferenceType
	}{
		{
			name:            "valid manifest tag",
			registry:        "example.com",
			path:            "/v2/foo/bar/manifests/hello-world",
			expectedRef:     "example.com/foo/bar:hello-world",
			expectedDgst:    "",
			expectedRefType: ReferenceTypeManifest,
		},
		{
			name:            "valid blob digest",
			registry:        "docker.io",
			path:            "/v2/library/nginx/blobs/sha256:295c7be079025306c4f1d65997fcf7adb411c88f139ad1d34b537164aa060369",
			expectedRef:     "",
			expectedDgst:    digest.Digest("sha256:295c7be079025306c4f1d65997fcf7adb411c88f139ad1d34b537164aa060369"),
			expectedRefType: ReferenceTypeBlob,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, dgst, refType, err := ParsePathComponents(tt.registry, tt.path)
			require.NoError(t, err)
			require.Equal(t, tt.expectedRef, ref)
			require.Equal(t, tt.expectedDgst, dgst)
			require.Equal(t, tt.expectedRefType, refType)
		})
	}
}

func TestParsePathComponentsInvalidPath(t *testing.T) {
	_, _, _, err := ParsePathComponents("example.com", "/v2/xenitab/spegel/v0.0.1")
	require.EqualError(t, err, "distribution path could not be parsed")
}

func TestParsePathComponentsMissingRegistry(t *testing.T) {
	_, _, _, err := ParsePathComponents("", "/v2/xenitab/spegel/manifests/v0.0.1")
	require.EqualError(t, err, "registry parameter needs to be set for tag references")
}
