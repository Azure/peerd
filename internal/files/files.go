// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License, Version 2.0.
package files

import (
	"fmt"
	"io"

	"github.com/azure/peerd/internal/remote"
	"github.com/azure/peerd/pkg/math"
)

const (
	FileChunkKeySep = "_"
)

// CacheBlockSize is the size of a single cached block.
var CacheBlockSize int = 1 * 1024 * 1024 // 1 Mib

// FileChunkKey returns the p2p lookup key for the given chunk of a file.
func FileChunkKey(name string, offset, cacheBlockSize int64) string {
	return name + FileChunkKeySep + fmt.Sprint(math.AlignDown(offset, cacheBlockSize))
}

// Fetchfile gets the content of a file from the given offset using a remote reader.
func FetchFile(r remote.Reader, name string, offset int64, count int) ([]byte, error) {
	d := make([]byte, count)
	l := r.Log().With().Str("name", name).Int64("offset", offset).Int("count", count).Logger()
	l.Debug().Msg("fetch file start")

	_, err := r.PreadRemote(d, offset)
	if err != nil && err != io.EOF {
		l.Error().Err(err).Msg("fetch file error")
		return nil, err
	}

	l.Debug().Msg("fetch file stop")
	return d, nil
}
