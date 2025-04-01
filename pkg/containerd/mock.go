// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package containerd

import (
	"context"
	"fmt"
	"io"

	"github.com/opencontainers/go-digest"
)

const testManifestBlob = `{"schemaVersion": 2, "mediaType": "application/vnd.oci.image.manifest.v1+json", "config": {"digest": "sha256:bb863d6b95453b6b10dfaa1a52cb53f453d9a97ee775808ebaf6533bb4c9bb30", "mediaType": "application/vnd.oci.image.config.v1+json", "size": 0}, "layers": []}`

type MockContainerdStore struct {
	validRefs        []Reference
	sizeTooLargeRefs []Reference
	invalidBytesRefs []Reference
}

var _ Store = &MockContainerdStore{}

func NewMockContainerdStore(validRefs []Reference) *MockContainerdStore {
	return &MockContainerdStore{
		validRefs: validRefs,
	}
}

func (m *MockContainerdStore) Verify(ctx context.Context) error {
	return nil
}

func (m *MockContainerdStore) Subscribe(ctx context.Context) (<-chan Reference, <-chan error) {
	return nil, nil
}

func (m *MockContainerdStore) List(ctx context.Context) ([]Reference, error) {
	return m.validRefs, nil
}

func (m *MockContainerdStore) All(ctx context.Context, ref Reference) ([]string, error) {
	return []string{ref.Digest().String()}, nil
}

func (m *MockContainerdStore) Resolve(ctx context.Context, ref string) (digest.Digest, error) {
	for _, r := range m.validRefs {
		if r.Name() == ref {
			return r.Digest(), nil
		}
	}

	return "", nil
}

func (m *MockContainerdStore) Size(ctx context.Context, dgst digest.Digest) (int64, error) {
	for _, r := range m.validRefs {
		if r.Digest() == dgst {
			return int64(len([]byte(testManifestBlob))), nil
		}
	}

	for _, r := range m.sizeTooLargeRefs {
		if r.Digest() == dgst {
			return maxManifestSize + 1, nil
		}
	}

	for _, r := range m.invalidBytesRefs {
		if r.Digest() == dgst {
			return 100, nil
		}
	}

	return -1, fmt.Errorf("digest %s not found", dgst)
}

func (m *MockContainerdStore) Write(ctx context.Context, dst io.Writer, dgst digest.Digest) error {
	for _, r := range m.validRefs {
		if r.Digest() == dgst {
			_, err := dst.Write([]byte(testManifestBlob))
			return err
		}
	}

	return fmt.Errorf("digest %s not found", dgst)
}

func (m *MockContainerdStore) Bytes(ctx context.Context, dgst digest.Digest) ([]byte, string, error) {
	for _, r := range m.validRefs {
		if r.Digest() == dgst {
			return []byte(testManifestBlob), "application/vnd.oci.image.manifest.v1+json", nil
		}
	}

	for _, r := range m.invalidBytesRefs {
		if r.Digest() == dgst {
			return nil, "", fmt.Errorf("unknown error while reading bytes")
		}
	}

	return nil, "", fmt.Errorf("digest %s not found", dgst)
}
