// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package oci

import (
	"net/http"
	"net/http/httptest"
	"testing"

	p2pcontext "github.com/azure/peerd/internal/context"
	"github.com/azure/peerd/internal/oci/distribution"
	"github.com/azure/peerd/internal/oci/store/tests"
	"github.com/azure/peerd/pkg/containerd"
	"github.com/gin-gonic/gin"
)

func TestNewRegistry(t *testing.T) {
	// Create a new registry
	r := NewRegistry(tests.NewMockContainerdStore(nil))

	if r == nil {
		t.Fatal("expected registry")
	}
}

func TestHandleManifest(t *testing.T) {
	img, err := containerd.ParseReference("library/alpine:3.18.0", "sha256:bb863d6b95453b6b10dfaa1a52cb53f453d9a97ee775808ebaf6533bb4c9bb30")
	if err != nil {
		t.Fatal(err)
	}
	refs := []containerd.Reference{img}

	ms := tests.NewMockContainerdStore(refs)

	r := NewRegistry(ms)

	mr := httptest.NewRecorder()
	mc, _ := gin.CreateTestContext(mr)

	req, err := http.NewRequest("GET", "http://127.0.0.1:5000/v2/library/alpine/manifests/3.18.0", nil)
	if err != nil {
		t.Fatal(err)
	}

	mc.Request = req

	r.handleManifest(mc, "sha256:bb863d6b95453b6b10dfaa1a52cb53f453d9a97ee775808ebaf6533bb4c9bb30")

	if mr.Code != 200 {
		t.Fatalf("expected 200, got %d", mr.Code)
	}

	if mr.Body.String() != "test" {
		t.Fatalf("expected test, got %s", mr.Body.String())
	}

	if mr.Header().Get(contentTypeHeader) != "application/vnd.oci.image.manifest.v1+json" {
		t.Fatalf("expected application/vnd.oci.image.manifest.v1+json, got %s", mr.Header().Get(contentTypeHeader))
	}

	if mr.Header().Get(contentLengthHeader) != "4" {
		t.Fatalf("expected 4, got %s", mr.Header().Get(contentLengthHeader))
	}
}

func TestHandleBlob(t *testing.T) {
	img, err := containerd.ParseReference("library/alpine:3.18.0", "sha256:blob")
	if err != nil {
		t.Fatal(err)
	}
	refs := []containerd.Reference{img}

	ms := tests.NewMockContainerdStore(refs)

	r := NewRegistry(ms)

	mr := httptest.NewRecorder()
	mc, _ := gin.CreateTestContext(mr)

	req, err := http.NewRequest("GET", "http://127.0.0.1:5000/v2/library/alpine/blobs/sha256:blob", nil)
	if err != nil {
		t.Fatal(err)
	}

	mc.Request = req

	r.handleBlob(mc, "sha256:blob")

	if mr.Code != 200 {
		t.Fatalf("expected 200, got %d", mr.Code)
	}

	if mr.Body.String() != "test" {
		t.Fatalf("expected test, got %s", mr.Body.String())
	}

	if mr.Header().Get(contentLengthHeader) != "4" {
		t.Fatalf("expected 4, got %s", mr.Header().Get(contentLengthHeader))
	}

	if mr.Header().Get(dockerContentDigestHeader) != "sha256:blob" {
		t.Fatalf("expected sha256:blob, got %s", mr.Header().Get(dockerContentDigestHeader))
	}
}

func TestHandle(t *testing.T) {
	img, err := containerd.ParseReference("library/alpine:3.18.0", "sha256:bb863d6b95453b6b10dfaa1a52cb53f453d9a97ee775808ebaf6533bb4c9bb30")
	if err != nil {
		t.Fatal(err)
	}
	refs := []containerd.Reference{img}

	ms := tests.NewMockContainerdStore(refs)

	r := NewRegistry(ms)

	mr := httptest.NewRecorder()
	mc, _ := gin.CreateTestContext(mr)

	req, err := http.NewRequest("GET", "http://127.0.0.1:5000/v2/library/alpine/manifests/3.18.0", nil)
	if err != nil {
		t.Fatal(err)
	}

	mc.Request = req
	mc.Set(p2pcontext.DigestCtxKey, "sha256:bb863d6b95453b6b10dfaa1a52cb53f453d9a97ee775808ebaf6533bb4c9bb30")
	mc.Set(p2pcontext.ReferenceCtxKey, "library/alpine:3.18.0")
	mc.Set(p2pcontext.RefTypeCtxKey, distribution.ReferenceType(distribution.ReferenceTypeManifest))

	r.Handle(mc)

	if mr.Code != 200 {
		t.Fatalf("expected 200, got %d", mr.Code)
	}

	if mr.Body.String() != "test" {
		t.Fatalf("expected test, got %s", mr.Body.String())
	}

	if mr.Header().Get(contentTypeHeader) != "application/vnd.oci.image.manifest.v1+json" {
		t.Fatalf("expected application/vnd.oci.image.manifest.v1+json, got %s", mr.Header().Get(contentTypeHeader))
	}

	if mr.Header().Get(contentLengthHeader) != "4" {
		t.Fatalf("expected 4, got %s", mr.Header().Get(contentLengthHeader))
	}
}
