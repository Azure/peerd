package k8s

import (
	"testing"
)

func TestEmptyConfigOutsidePod(t *testing.T) {
	_, err := NewKubernetesInterface("")
	if err == nil {
		t.Error("Expected non-nil error, got nil")
	}
}
