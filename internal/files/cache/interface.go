// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package cache

// Cache describes the cache of files.
type Cache interface {
	// Size gets the size of the file.
	Size(path string) (int64, bool)

	// PutSize sets size of the file.
	PutSize(path string, length int64) bool

	// Exists checks if the given chunk of the file is already cached.
	Exists(name string, offset int64) bool

	// GetOrCreate gets the cached value if available, otherwise downloads the file.
	GetOrCreate(name string, offset int64, count int, fetch func() ([]byte, error)) ([]byte, error)
}

var (
	// FilesCacheMaxCost is the capacity of the files cache in any unit.
	FilesCacheMaxCost int64 = 4 * 1024 * 1024 * 1024 // 4 Gib

	// MemoryCacheMaxCost is the capacity of the memory cache in any unit.
	MemoryCacheMaxCost int64 = 1 * 1024 * 1024 * 1024 // 1 Gib

	// Path is the path to the cache directory.
	Path string = "/tmp/distribution/p2p/cache"
)
