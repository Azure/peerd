package store

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/opencontainers/go-digest"
)

// FilesStore describes a store for files.
type FilesStore interface {
	// Key tries to find the cache key for the requested content or returns empty.
	Key(c *gin.Context) (key string, d digest.Digest, err error)

	// Open opens the requested file and starts prefetching it. It also returns the size of the file.
	Open(c *gin.Context) (File, error)

	// Subscribe returns a channel that will be notified when a blob is added to the store.
	Subscribe() chan string
}

// File is an abstraction for a file that can be read from this store.
// It is similar to os.File.
type File interface {
	// Seek sets the current file offset.
	Seek(offset int64, whence int) (int64, error)

	// Fstat returns the size of the file.
	Fstat() (int64, error)

	// Read reads up to len(p) bytes into p. It returns the number of bytes read (0 <= n <= len(p)) and any error encountered.
	Read(p []byte) (n int, err error)

	// ReadAt reads len(p) bytes from the File starting at byte offset off. It returns the number of bytes read and the error, if any.
	ReadAt(buff []byte, off int64) (int, error)
}

var (
	// PrefetchWorkers is the number of workers that will be used to prefetch files.
	// To disable prefetch, set this to 0.
	PrefetchWorkers = 50

	// ResolveRetries is the number of times to attempt resolving a key before giving up.
	ResolveRetries = 3

	// ResolveTimeout is the timeout for resolving a key.
	ResolveTimeout = 20 * time.Millisecond
)
