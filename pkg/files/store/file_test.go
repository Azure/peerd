// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package store

import (
	"crypto/rand"
	"io"
	"os"
	"strings"
	"testing"

	readermocks "github.com/azure/peerd/pkg/discovery/content/reader/mocks"
	"github.com/azure/peerd/pkg/discovery/routing/mocks"
	"github.com/azure/peerd/pkg/files"
)

func TestReadAtWithChunkOffset(t *testing.T) {
	data := []byte("hello world")

	files.CacheBlockSize = 1 // 1 byte

	s, err := NewFilesStore(ctxWithMetrics, mocks.NewMockRouter(make(map[string][]string)), testFileCachePath)
	if err != nil {
		t.Fatal(err)
	}

	fWithChunkOffset := &file{
		Name:        "test",
		reader:      readermocks.NewMockReader(data),
		store:       s.(*store),
		chunkOffset: 4,
	}
	size, err := fWithChunkOffset.Fstat()
	if err != nil {
		t.Fatal(err)
	} else if size != int64(11) {
		t.Errorf("expected size %d, got %d", 11, size)
	}

	// Read the first byte, should get an error.
	buf := make([]byte, 1)
	_, err = fWithChunkOffset.ReadAt(buf, 0)
	if err == nil {
		t.Fatalf("expected %v, got nil", errOnlySingleChunkAvailable)
	} else if err != errOnlySingleChunkAvailable {
		t.Fatalf("expected %v, got %v", errOnlySingleChunkAvailable, err)
	}

	_, err = os.ReadFile(testFileCachePath + "/test/0")
	if !strings.Contains(err.Error(), "no such file or directory") {
		t.Fatalf("expected chunk file to not exist, got %v", err)
	}

	// Read the allowed chunk.
	n, err := fWithChunkOffset.ReadAt(buf, 4)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("expected to read %d bytes, got %d", 1, n)
	}
	if string(buf[0]) != "o" {
		t.Errorf("expected to read %q, got %q", "o", string(buf[0]))
	}

	chunkFile, err := os.ReadFile(testFileCachePath + "/test/4")
	if err != nil {
		t.Fatal(err)
	}
	if string(chunkFile) != "o" {
		t.Errorf("expected chunk file to contain %q, got %q", "o", string(chunkFile))
	}
}

func TestReadAt(t *testing.T) {
	data := []byte("hello world")

	files.CacheBlockSize = 1 // 1 byte

	s, err := NewFilesStore(ctxWithMetrics, mocks.NewMockRouter(make(map[string][]string)), testFileCachePath)
	if err != nil {
		t.Fatal(err)
	}

	f := &file{
		Name:   "test",
		reader: readermocks.NewMockReader(data),
		store:  s.(*store),
	}
	size, err := f.Fstat()
	if err != nil {
		t.Fatal(err)
	} else if size != int64(11) {
		t.Errorf("expected size %d, got %d", 11, size)
	}

	// Read the first byte.
	buf := make([]byte, 1)
	n, err := f.ReadAt(buf, 0)
	if err != nil {
		t.Fatal(err)
	} else if n != 1 {
		t.Errorf("expected to read %d byte, got %d", 1, n)
	}

	chunkFile, err := os.ReadFile(testFileCachePath + "/test/0")
	if err != nil {
		t.Fatal(err)
	} else if string(chunkFile) != "h" {
		t.Errorf("expected chunk file to contain %q, got %q", "h", string(chunkFile))
	}

	// Read in the middle.
	buf = make([]byte, 4)
	n, err = f.ReadAt(buf, 3)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("expected to read %d bytes, got %d", 1, n)
	}
	if string(buf[0]) != "l" {
		t.Errorf("expected to read %q, got %q", "l", string(buf[0]))
	}
}

func TestSeek(t *testing.T) {
	data := []byte("hello world")

	s, err := NewFilesStore(ctxWithMetrics, mocks.NewMockRouter(make(map[string][]string)), testFileCachePath)
	if err != nil {
		t.Fatal(err)
	}

	f := &file{
		Name:   "test",
		reader: readermocks.NewMockReader(data),
		store:  s.(*store),
	}
	size, err := f.Fstat()
	if err != nil {
		t.Fatal(err)
	} else if size != int64(11) {
		t.Errorf("expected size %d, got %d", 11, size)
	}

	// Seek to the beginning.
	c, err := f.Seek(0, io.SeekStart)
	if err != nil {
		t.Fatal(err)
	} else if c != 0 {
		t.Errorf("expected cursor %d, got %d", 0, c)
	}

	// Seek to the middle.
	c, err = f.Seek(size/2, io.SeekStart)
	if err != nil {
		t.Fatal(err)
	} else if c != size/2 {
		t.Errorf("expected cursor %d, got %d", size/2, c)
	}

	// Seek to the middle.
	c, err = f.Seek(0, io.SeekCurrent)
	if err != nil {
		t.Fatal(err)
	} else if c != size/2 {
		t.Errorf("expected cursor %d, got %d", size/2, c)
	}

	// Seek to the end.
	c, err = f.Seek(size, io.SeekStart)
	if err != nil {
		t.Fatal(err)
	} else if c != size {
		t.Errorf("expected cursor %d, got %d", size, c)
	}

	// Seek to the end.
	c, err = f.Seek(0, io.SeekEnd)
	if err != nil {
		t.Fatal(err)
	} else if c != size {
		t.Errorf("expected cursor %d, got %d", size, c)
	}
}

func TestFstat(t *testing.T) {
	data, err := randomBytesN(100)
	if err != nil {
		t.Fatal(err)
	}

	s, err := NewFilesStore(ctxWithMetrics, mocks.NewMockRouter(make(map[string][]string)), testFileCachePath)
	if err != nil {
		t.Fatal(err)
	}

	f := &file{
		Name:   "test",
		reader: readermocks.NewMockReader(data),
		store:  s.(*store),
	}

	size, err := f.Fstat()
	if err != nil {
		t.Fatal(err)
	} else if size != int64(100) {
		t.Errorf("expected size %d, got %d", 100, size)
	}

	f = &file{
		Name:        "test2",
		reader:      readermocks.NewMockReader(data),
		store:       s.(*store),
		chunkOffset: 14,
	}

	size, err = f.Fstat()
	if err != nil {
		t.Fatal(err)
	} else if size != int64(100) {
		t.Errorf("expected size %d, got %d", 100, size)
	}
}

func randomBytesN(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}
