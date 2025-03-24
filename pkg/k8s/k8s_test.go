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
	if err := os.Setenv("NAMESPACE", "custom-ns"); err != nil {
		t.Fatalf("Failed to set NAMESPACE: %v", err)
	}
	namespace = getPodNamespace()
	if namespace != "custom-ns" {
		t.Errorf("Expected namespace to be 'custom-ns', got %s", namespace)
	}

	// Unset NAMESPACE.
	if err := os.Unsetenv("NAMESPACE"); err != nil {
		t.Fatalf("Failed to unset NAMESPACE: %v", err)
	}
}
