// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License, Version 2.0.
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
