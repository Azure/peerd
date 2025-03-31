package mocks

import "testing"

func TestNewMockReader(t *testing.T) {
	data := []byte("test data")
	mr := NewMockReader(data)
	if mr == nil {
		t.Errorf("NewMockReader returned nil")
	}
}

func TestLog(t *testing.T) {
	data := []byte("test data")
	mr := NewMockReader(data)
	if mr.Log() == nil {
		t.Errorf("Log returned nil")
	}
}

func TestFstatRemote(t *testing.T) {
	data := []byte("test data")
	mr := NewMockReader(data)
	got, err := mr.FstatRemote()
	if err != nil {
		t.Errorf("FstatRemote returned error: %v", err)
	}
	if got != int64(len(data)) {
		t.Errorf("FstatRemote returned size %d, want %d", got, len(data))
	}
}

func TestPreadRemote(t *testing.T) {
	data := []byte("test data")
	mr := NewMockReader(data)
	buf := make([]byte, 4)
	n, err := mr.PreadRemote(buf, 0)
	if err != nil {
		t.Errorf("PreadRemote returned error: %v", err)
	}
	if n != 4 {
		t.Errorf("PreadRemote returned %d bytes, want 4", n)
	}
	if string(buf) != "test" {
		t.Errorf("PreadRemote returned %s, want 'test'", string(buf))
	}

	// Test reading from offset
	n, err = mr.PreadRemote(buf, 4)
	if err != nil {
		t.Errorf("PreadRemote returned error: %v", err)
	}
	if n != 4 {
		t.Errorf("PreadRemote returned %d bytes, want 4", n)
	}
	if string(buf) != " dat" {
		t.Errorf("PreadRemote returned %s, want ' dat'", string(buf))
	}

	buf = make([]byte, 1)
	n, err = mr.PreadRemote(buf, 8)
	if err != nil {
		t.Errorf("PreadRemote returned error: %v", err)
	}
	if n != 1 {
		t.Errorf("PreadRemote returned %d bytes, want 1", n)
	}
	if string(buf) != "a" {
		t.Errorf("PreadRemote returned %s, want 'a'", string(buf))
	}
}
