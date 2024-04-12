// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package cache

import (
	"io"
	"os"
	"path"
	"sync"
	"sync/atomic"

	"github.com/rs/zerolog"
)

var fdCnt int32

// item is a cached item.
type item struct {
	key  string
	file *os.File
	lock *sync.RWMutex
}

// drop deletes the underlying file.
func (i *item) drop(l zerolog.Logger) {
	i.lock.Lock()
	defer i.lock.Unlock()

	count := atomic.AddInt32(&fdCnt, -1)
	l.Debug().Str("name", i.file.Name()).Int32("count", count).Msg("cache item drop")

	if err := i.file.Close(); err != nil {
		l.Error().Err(err).Str("name", i.file.Name()).Msg("failed to close file")
	}

	if err := os.Remove(i.file.Name()); err != nil {
		l.Error().Err(err).Str("name", i.file.Name()).Msg("failed to remove file")
	}

	i.file = nil
}

// bytes returns the file bytes.
func (i *item) bytes(l zerolog.Logger) []byte {
	b, err := readFromStart(i.file)
	if err != nil {
		l.Error().Err(err).Str("name", i.file.Name()).Msg("failed to read file")
		return nil
	}

	return b
}

// fill files the file with the given data.
func (i *item) fill(log zerolog.Logger, fetch func() ([]byte, error)) (int, error) {
	buffer, err := fetch()
	if err != nil {
		if err := os.Remove(i.file.Name()); err != nil {
			log.Error().Err(err).Str("name", i.file.Name()).Msg("attempted to remove file because the size read did not match the file size")
		}
		return 0, err
	}

	l, err := writeAll(i.file, buffer)
	if err != nil {
		if err := os.Remove(i.file.Name()); err != nil {
			log.Error().Err(err).Str("name", i.file.Name()).Msg("attempted to remove file because the size written did not match the file size")
		}
		return 0, err
	}

	return l, nil
}

// readFromStart reads the entire file from the beginning.
func readFromStart(file *os.File) ([]byte, error) {
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	fileSize := info.Size()
	fileContent := make([]byte, fileSize)
	offset := int64(0)

	for offset < fileSize && err == nil {
		var l int
		l, err = file.ReadAt(fileContent[offset:], offset)
		offset += int64(l)
	}
	if err == io.EOF {
		err = nil
	}

	if err != nil {
		return nil, err
	}

	if offset != fileSize {
		return nil, io.ErrUnexpectedEOF
	}

	return fileContent[:offset], nil
}

// writeAll writes the file.
func writeAll(file *os.File, buff []byte) (int, error) {
	offset := 0
	err := file.Truncate(0)
	if err != nil {
		return 0, err
	}

	return file.Write(buff[offset:])
}

// newItem creates a new cache item that is ready to be filled.
func newItem(key string, l zerolog.Logger) (*item, error) {
	cacheItem := &item{key: key, lock: new(sync.RWMutex)}
	if err := os.MkdirAll(path.Dir(key), 0755); err != nil {
		return nil, err
	}

	fdCounter := atomic.AddInt32(&fdCnt, 1)
	l.Debug().Str("key", key).Int32("count", fdCounter).Msg("create new cached item")

	var err error
	if cacheItem.file, err = os.OpenFile(key, os.O_CREATE|os.O_RDWR, 0644); err != nil {
		return nil, err
	}

	return cacheItem, nil
}
