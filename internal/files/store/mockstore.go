// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License, Version 2.0.
package store

import (
	"context"

	"github.com/azure/peerd/internal/files/cache"
	"github.com/azure/peerd/internal/routing"
)

type MockStore struct {
	*store
}

var _ FilesStore = &MockStore{}

func (m *MockStore) Cache() cache.Cache {
	return m.store.cache
}

func NewMockStore(ctx context.Context, r routing.Router) (*MockStore, error) {
	s, err := NewFilesStore(ctx, r)
	if err != nil {
		return nil, err
	}
	return &MockStore{s.(*store)}, nil
}