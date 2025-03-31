package mocks

import "testing"

func TestID(t *testing.T) {
	h := MockHost{}
	id := h.ID()
	if id != "localhost-peer-for-unit-testing" {
		t.Errorf("expected 'localhost-peer-for-unit-testing', got '%s'", id)
	}
}
