// Initial Copyright (c) 2023 Xenit AB and 2024 The Spegel Authors.
// Portions Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package mocks

import (
	"bytes"
	"context"
	"fmt"

	"github.com/containerd/containerd/content"
	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// MockContentStore is a mock implementation of containerd's content store.
type MockContentStore struct {
	Data map[string]string
}

var _ content.Store = &MockContentStore{}

// Info returns the content.Info for the given digest, if it exists in the mocked data keyed by digest.
func (m *MockContentStore) Info(ctx context.Context, dgst digest.Digest) (content.Info, error) {
	if d, ok := m.Data[dgst.String()]; ok {
		return content.Info{
			Digest: dgst,
			Size:   int64(len(d)),
		}, nil
	}
	return content.Info{}, fmt.Errorf("digest not found: %s", dgst.String())
}

func (*MockContentStore) Walk(ctx context.Context, fn content.WalkFunc, filters ...string) error {
	panic("not implemented")
}

func (*MockContentStore) Delete(ctx context.Context, dgst digest.Digest) error {
	panic("not implemented")
}

// ReaderAt returns a content.ReaderAt for the given descriptor, if it exists in the mocked data keyed by digest.
func (m *MockContentStore) ReaderAt(ctx context.Context, desc v1.Descriptor) (content.ReaderAt, error) {
	s, ok := m.Data[desc.Digest.String()]
	if !ok {
		return nil, fmt.Errorf("digest not found: %s", desc.Digest.String())
	}
	return &readerAt{*bytes.NewReader([]byte(s))}, nil
}

func (*MockContentStore) Status(ctx context.Context, ref string) (content.Status, error) {
	panic("not implemented")
}

func (*MockContentStore) Update(ctx context.Context, info content.Info, fieldpaths ...string) (content.Info, error) {
	panic("not implemented")
}

func (*MockContentStore) ListStatuses(ctx context.Context, filters ...string) ([]content.Status, error) {
	panic("not implemented")
}

func (*MockContentStore) Writer(ctx context.Context, opts ...content.WriterOpt) (content.Writer, error) {
	panic("not implemented")
}

func (*MockContentStore) Abort(ctx context.Context, ref string) error {
	panic("not implemented")
}

type readerAt struct {
	bytes.Reader
}

func (r *readerAt) Close() error {
	return nil
}
