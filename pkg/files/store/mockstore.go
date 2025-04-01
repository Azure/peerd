// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package store

import (
	"context"

	"github.com/azure/peerd/pkg/cache"
	"github.com/azure/peerd/pkg/discovery/routing"
)

type MockStore struct {
	*store
}

var _ FilesStore = &MockStore{}

func (m *MockStore) Cache() cache.Cache {
	return m.cache
}

func NewMockStore(ctx context.Context, r routing.Router, fileCachePath string) (*MockStore, error) {
	s, err := NewFilesStore(ctx, r, fileCachePath)
	if err != nil {
		return nil, err
	}
	return &MockStore{s.(*store)}, nil
}
