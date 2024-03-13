// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package tests

import (
	"context"
	"io"

	"github.com/azure/peerd/pkg/containerd"
	"github.com/opencontainers/go-digest"
)

type MockContainerdStore struct {
	refs []containerd.Reference
}

var _ containerd.Store = &MockContainerdStore{}

func NewMockContainerdStore(refs []containerd.Reference) *MockContainerdStore {
	return &MockContainerdStore{
		refs: refs,
	}
}

func (m *MockContainerdStore) Verify(ctx context.Context) error {
	return nil
}

func (m *MockContainerdStore) Subscribe(ctx context.Context) (<-chan containerd.Reference, <-chan error) {
	return nil, nil
}

func (m *MockContainerdStore) List(ctx context.Context) ([]containerd.Reference, error) {
	return m.refs, nil
}

func (m *MockContainerdStore) All(ctx context.Context, ref containerd.Reference) ([]string, error) {
	return []string{ref.Digest().String()}, nil
}

func (m *MockContainerdStore) Resolve(ctx context.Context, ref string) (digest.Digest, error) {
	return "", nil
}

func (m *MockContainerdStore) Size(ctx context.Context, dgst digest.Digest) (int64, error) {
	return 0, nil
}

func (m *MockContainerdStore) Write(ctx context.Context, dst io.Writer, dgst digest.Digest) error {
	return nil
}

func (m *MockContainerdStore) Bytes(ctx context.Context, dgst digest.Digest) ([]byte, string, error) {
	return nil, "", nil
}
