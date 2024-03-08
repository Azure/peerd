// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License, Version 2.0.
package cache

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	syncmap "github.com/azure/peerd/internal/cache"
	"github.com/azure/peerd/internal/files"
	"github.com/dgraph-io/ristretto"
	"github.com/rs/zerolog"
)

// fileCache implements FileCache.
type fileCache struct {
	fileCache     *ristretto.Cache
	metadataCache *syncmap.SyncMap
	path          string
	lock          sync.RWMutex
	log           zerolog.Logger
}

var _ Cache = &fileCache{}

// Exists checks if the file exists in the cache.
func (c *fileCache) Exists(name string, offset int64) bool {
	key := c.getKey(name, offset)
	val, found := c.fileCache.Get(key)
	if found {
		cacheItem := val.(*item)
		cacheItem.lock.Lock()
		defer cacheItem.lock.Unlock()

		if info, err := cacheItem.file.Stat(); err != nil {
			return false
		} else {
			return info.Size() > 0
		}
	}
	return false
}

// GetOrCreate gets the cached value if available, otherwise fetches it.
func (c *fileCache) GetOrCreate(name string, alignedOffset int64, count int, fetch func() ([]byte, error)) ([]byte, error) {
	key := c.getKey(name, alignedOffset)
	val, found := c.fileCache.Get(key)
	if !found {
		c.lock.Lock()
		if val, found = c.fileCache.Get(key); found && val != nil {
			c.lock.Unlock()
		} else {
			var err error
			val, err = newItem(key, c.log)
			if err != nil {
				c.lock.Unlock()
				return nil, err
			}
			ok := c.fileCache.Set(key, val, 0)
			if !ok {
				c.lock.Unlock()
				return nil, io.ErrUnexpectedEOF
			}

			// wait for value to pass through buffers
			waitForSet()

			c.lock.Unlock()
		}
	}

	cacheItem := val.(*item)

	cacheItem.lock.RLock()
	info, err := cacheItem.file.Stat()

	if err != nil {
		cacheItem.lock.RUnlock()
		return nil, err
	}

	if info.Size() != int64(count) {
		cacheItem.lock.RUnlock()

		cacheItem.lock.Lock()

		// check again after acquiring lock
		info, err = cacheItem.file.Stat()
		if err != nil {
			cacheItem.lock.Unlock()
			return nil, err
		} else if info.Size() != int64(count) {

			n, err := cacheItem.fill(c.log, fetch)
			cacheItem.lock.Unlock()

			if err != nil {
				return nil, err
			} else if int64(n) != int64(count) {
				return nil, fmt.Errorf("fill did not retrieve expected number of bytes, expected: %v, got: %v", count, n)
			}
		} else {
			cacheItem.lock.Unlock()
		}
		cacheItem.lock.RLock()
	}

	result := cacheItem.bytes(c.log)
	cacheItem.lock.RUnlock()

	if len(result) != count {
		return result, fmt.Errorf("bytes did not retrieve expected number of bytes, expected: %v, got: %v", count, len(result))
	}

	return result, nil
}

// Size gets the length of the file.
func (c *fileCache) Size(name string) (int64, bool) {
	key := filepath.Join(name, "metainfo")
	// c.metadataCache.Wait()
	val, found := c.metadataCache.Get(key)
	if !found {
		return 0, false
	}
	return val.(int64), true
}

// PutSize puts the length of the file.
func (c *fileCache) PutSize(name string, len int64) bool {
	key := filepath.Join(name, "metainfo")
	c.metadataCache.Set(key, len)
	c.log.Debug().Str("key", key).Int64("len", len).Msg("put len")
	return true
}

func (c *fileCache) getKey(name string, offset int64) string {
	return filepath.Join(c.path, name, strconv.FormatInt(offset, 10))
}

func waitForSet() {
	time.Sleep(10 * time.Millisecond)
}

// New creates a new cache of files.
func New(ctx context.Context) Cache {
	log := zerolog.Ctx(ctx).With().Str("component", "cache").Logger()

	atomic.StoreInt32(&fdCnt, 0)
	if err := os.MkdirAll(Path, 0755); err != nil {
		// This will call os.Exit(1)
		log.Fatal().Err(err).Str("path", Path).Msg("failed to initialize cache directory")
	}

	cache := &fileCache{
		log:           log,
		path:          Path,
		metadataCache: syncmap.MakeSyncMap(1e7),
	}

	var err error
	if cache.fileCache, err = ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,
		MaxCost:     FilesCacheMaxCost,
		BufferItems: 64,

		OnExit: func(val interface{}) {
			item := val.(*item)
			item.drop(log)
		},

		Cost: func(val interface{}) int64 {
			return int64(files.CacheBlockSize)
		},
	}); err != nil {
		// This will call os.Exit(1)
		log.Fatal().Err(err).Msg("failed to initialize file cache")
	}

	return cache
}
