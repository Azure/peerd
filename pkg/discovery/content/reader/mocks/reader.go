// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package mocks

import (
	"github.com/azure/peerd/pkg/discovery/content/reader"
	"github.com/rs/zerolog"
)

var l = zerolog.Nop()

type mockReader struct {
	data []byte
}

var _ reader.Reader = &mockReader{}

// FstatRemote implements remote.Reader.
func (m *mockReader) FstatRemote() (int64, error) {
	return int64(len(m.data)), nil
}

// Log implements remote.Reader.
func (*mockReader) Log() *zerolog.Logger {
	return &l
}

// PreadRemote implements remote.Reader.
func (m *mockReader) PreadRemote(buf []byte, offset int64) (int, error) {
	if offset >= int64(len(m.data)) {
		return 0, nil
	}
	return copy(buf, m.data[offset:]), nil
}

// NewMockReader creates a new mock reader for testing purposes.
func NewMockReader(data []byte) reader.Reader {
	return &mockReader{data: data}
}
