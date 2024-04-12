// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package remote

import (
	"net/http"

	"github.com/rs/zerolog"
)

// Reader provides a read-only interface to a remote file.
type Reader interface {
	// PreadRemote is like pread but to a remote file.
	PreadRemote(buf []byte, offset int64) (int, error)

	// FstatRemote stats a remote file.
	FstatRemote() (int64, error)

	// Log returns the logger with context for this reader.
	Log() *zerolog.Logger
}

// Error describes an error that occurred during a remote operation.
type Error struct {
	*http.Response
	error
}
