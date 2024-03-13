// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package urlparser

import (
	"testing"

	"github.com/opencontainers/go-digest"
)

func TestParser(t *testing.T) {
	p := New()
	if p == nil {
		t.Errorf("expected non-nil parser")
	}

	// Test Azure URLs
	for _, test := range azureTestCases {
		got, err := p.ParseDigest(test.url)
		if test.valid {
			if err != nil {
				t.Errorf("expected no error parsing digest from url %s", test.url)
			} else if got != digest.Digest(test.digest) {
				t.Errorf("expected digest %s, got %s", test.digest, got)
			}
		} else {
			if err == nil {
				t.Errorf("expected error parsing digest from url %s", test.url)
			}
		}
	}
}
