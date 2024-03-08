package files

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/azure/peerd/internal/remote"
	"github.com/rs/zerolog"
)

func TestFileChunkKey(t *testing.T) {
	d := "abc"
	cacheBlockSize := int64(1024)

	key := FileChunkKey(d, 123, cacheBlockSize)
	if key != "abc_0" {
		t.Errorf("expected key %s, got %s", "abc_0", key)
	}

	key = FileChunkKey(d, int64(123)+cacheBlockSize, cacheBlockSize)
	exp := fmt.Sprintf("abc_%v", cacheBlockSize)
	if key != exp {
		t.Errorf("expected key %s, got %s", exp, key)
	}
}

func TestFetchFile(t *testing.T) {
	d := map[string][]byte{
		"0": []byte("abc"),
		"3": []byte("def"),
	}

	r := &mockReader{data: d}

	if b, err := FetchFile(r, "test", 0, 3); err != nil {
		t.Errorf("expected no error, got %v", err)
	} else if string(b) != "abc" {
		t.Errorf("expected %s, got %s", "abc", string(b))
	}

	if b, err := FetchFile(r, "test", 0, 4); err != nil {
		t.Errorf("expected no error, got %v", err)
	} else if string(b[:3]) != "abc" {
		t.Errorf("expected %s, got %s", "abc", string(b))
	}

	if b, err := FetchFile(r, "test", 3, 3); err != nil {
		t.Errorf("expected no error, got %v", err)
	} else if string(b) != "def" {
		t.Errorf("expected %s, got %s", "def", string(b))
	}

	if b, err := FetchFile(r, "test", 31, 4); err == nil {
		t.Errorf("expected error, got %s", string(b))
	}
}

type mockReader struct {
	data map[string][]byte
}

// FstatRemote implements remote.Reader.
func (*mockReader) FstatRemote() (int64, error) {
	panic("unimplemented")
}

// Log implements remote.Reader.
func (*mockReader) Log() *zerolog.Logger {
	l := zerolog.Nop()
	return &l
}

// PreadRemote implements remote.Reader.
func (m *mockReader) PreadRemote(buf []byte, offset int64) (int, error) {
	if d, ok := m.data[strconv.FormatInt(offset, 10)]; ok {
		return copy(buf, d), nil
	} else {
		return 0, os.ErrNotExist
	}
}

var _ remote.Reader = &mockReader{}
