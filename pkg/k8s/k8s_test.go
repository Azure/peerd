// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package k8s

import (
	"os"
	"testing"
)

func TestEmptyConfigOutsidePod(t *testing.T) {
	_, err := NewKubernetesInterface("", "test-node")
	if err == nil {
		t.Error("Expected non-nil error, got nil")
	}
}

func TestGetPodNamespace(t *testing.T) {
	namespace := getPodNamespace()
	if namespace == "" {
		t.Error("Expected non-empty namespace, got empty")
	}

	if namespace != peerdDefaultNamespace {
		t.Errorf("Expected namespace to be '%s', got %s", peerdDefaultNamespace, namespace)
	}

	// Set NAMESPACE to a custom value.
	os.Setenv("NAMESPACE", "custom-ns")
	namespace = getPodNamespace()
	if namespace != "custom-ns" {
		t.Errorf("Expected namespace to be 'custom-ns', got %s", namespace)
	}

	// Unset NAMESPACE.
	os.Unsetenv("NAMESPACE")
}
