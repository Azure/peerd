// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License, Version 2.0.
package store

import (
	"fmt"
	"io"

	"sync"

	"github.com/azure/peerd/internal/files"
	"github.com/azure/peerd/internal/remote"
	"github.com/azure/peerd/pkg/math"
)

var errOnlySingleChunkAvailable = fmt.Errorf("only single chunk available")

// file describes a file that can be read from this content store.
// It implements the File interface. It is similar to os.File.
type file struct {
	Name string

	cur  int64
	size int64

	statLock sync.Mutex

	chunkOffset int64

	reader remote.Reader
	store  *store
}

var _ File = &file{}

// prefetch tries to prefetch the specified parts of the file in chunks of cacheBlockSize.
// It can silently fail.
func (f *file) prefetch(offset int64, count int64) {
	go func() {
		fileSize, err := f.Fstat()
		if err != nil {
			return
		}

		segs, err := math.NewSegments(offset, files.CacheBlockSize, count, fileSize)
		if err != nil {
			f.reader.Log().Error().Err(err).Msg("prefetch error: failed to create segments")
			return
		}

		for seg := range segs.All() {
			f.store.prefetchChan <- prefetchableSegment{
				name:   f.Name,
				reader: f.reader,
				offset: seg.Index,
				count:  seg.Count,
			}
		}
	}()
}

// Seek sets the current file offset.
func (f *file) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekCurrent:
		f.cur += offset
	case io.SeekStart:
		f.cur = offset
	case io.SeekEnd:
		f.cur = f.size
	}

	return f.cur, nil
}

// Fstat returns the size of the file.
func (f *file) Fstat() (int64, error) {
	var hit bool

	f.size, hit = f.store.cache.Size(f.Name)
	if !hit {
		f.reader.Log().Debug().Str("name", f.Name).Int64("size", f.size).Msg("fstat getlen cache miss_1")
		f.statLock.Lock()
		f.size, hit = f.store.cache.Size(f.Name)
		if !hit {
			f.reader.Log().Debug().Str("name", f.Name).Int64("size", f.size).Msg("fstat getlen cache miss_2")
			var err error
			f.size, err = f.reader.FstatRemote()
			if err != nil {
				f.reader.Log().Error().Err(err).Msg("fstat error")
				return 0, err
			}
			f.store.cache.PutSize(f.Name, f.size)
			f.reader.Log().Debug().Str("name", f.Name).Int64("size", f.size).Msg("fstat putlen")
		}
		f.statLock.Unlock()
	}

	return f.size, nil
}

// Read reads up to len(p) bytes into p. It returns the number of bytes read (0 <= n <= len(p)) and any error encountered.
func (f *file) Read(p []byte) (n int, err error) {
	ret, err := f.ReadAt(p, f.cur)
	if err == nil {
		f.cur += int64(ret)
	}
	return ret, err
}

// ReadAt reads len(p) bytes from the File starting at byte offset off. It returns the number of bytes read and the error, if any.
func (f *file) ReadAt(buff []byte, offset int64) (int, error) {
	fileSize, err := f.Fstat()
	if err != nil {
		return 0, err
	}

	alignedOffset := math.AlignDown(offset, int64(files.CacheBlockSize))

	if f.chunkOffset != 0 && alignedOffset != f.chunkOffset {
		f.reader.Log().Error().Err(errOnlySingleChunkAvailable).Int64("chunk", f.chunkOffset).Int64("alignedOffset", alignedOffset).Int64("requestedOffset", offset).Msg("file can only read chunk")
		return -1, errOnlySingleChunkAvailable
	}

	count := int(math.Min64(int64(files.CacheBlockSize), fileSize-alignedOffset))

	data, err := f.store.cache.GetOrCreate(f.Name, alignedOffset, count, func() ([]byte, error) {
		return files.FetchFile(f.reader, f.Name, alignedOffset, count)
	})
	if err != nil {
		f.reader.Log().Error().Err(err).Msg("readat error")
		return 0, fmt.Errorf("failed to ReadAt, path: %v, offset: %v, error: %v", f.Name, offset, err.Error())
	}

	pos := int(offset - alignedOffset)
	ret := math.Min(len(buff), len(data)-pos)
	ret = copy(buff[:ret], data[pos:pos+ret])

	if offset+int64(len(buff)) > fileSize {
		err = io.EOF
	}

	return ret, err
}
