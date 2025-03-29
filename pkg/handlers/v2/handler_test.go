// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package v2

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/azure/peerd/pkg/containerd"
	pcontext "github.com/azure/peerd/pkg/context"
	"github.com/azure/peerd/pkg/discovery/routing/mocks"
	"github.com/azure/peerd/pkg/metrics"
	"github.com/azure/peerd/pkg/oci/distribution"
	"github.com/gin-gonic/gin"
)

var (
	ctxWithMetrics, _ = metrics.WithContext(context.Background(), "test", "peerd")
)

func TestNew(t *testing.T) {
	mr := mocks.NewMockRouter(nil)
	ms := containerd.NewMockContainerdStore(nil)

	h, err := New(ctxWithMetrics, mr, ms)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if h == nil {
		t.Fatalf("unexpected nil handler")
	}
}

// TestHandlePeerNotFound tests the Handle method of the handler when the peer is not found.
// The request is simulated to be a GET manifest request from the local containerd client.
// The handler is expected to discover a peer and having found none after the timeout, return a 404.
func TestHandlePeerNotFound(t *testing.T) {
	mr := mocks.NewMockRouter(nil)
	ms := containerd.NewMockContainerdStore(nil)

	h, err := New(ctxWithMetrics, mr, ms)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	recorder := httptest.NewRecorder()
	mc, _ := gin.CreateTestContext(recorder)
	req, err := http.NewRequest("GET", "http://127.0.0.1:5000/v2/library/alpine/manifests/sha256:bb863d6b95453b6b10dfaa1a52cb53f453d9a97ee775808ebaf6533bb4c9bb30?ns=k8s.io", nil)
	if err != nil {
		t.Fatal(err)
	}
	mc.Request = req

	pc := pcontext.FromContext(mc)
	pcontext.FillCorrelationId(pc)
	h.Handle(pc)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status NotFound, got %d", recorder.Code)
	}
}

// TestHandleContentNotFound tests the Handle method of the handler when the content is not found.
// The request is simulated to be a GET manifest request from a peer peerd pod.
// The handler should look for the content in the local containerd store and having found none, return a 404.
func TestHandleContentNotFound(t *testing.T) {
	mr := mocks.NewMockRouter(nil)
	ms := containerd.NewMockContainerdStore(nil)

	h, err := New(ctxWithMetrics, mr, ms)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	recorder := httptest.NewRecorder()
	mc, _ := gin.CreateTestContext(recorder)
	req, err := http.NewRequest("GET", "http://127.0.0.1:5000/v2/library/alpine/manifests/sha256:bb863d6b95453b6b10dfaa1a52cb53f453d9a97ee775808ebaf6533bb4c9bb30?ns=k8s.io", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set(pcontext.P2PHeaderKey, "true")
	mc.Request = req

	pc := pcontext.FromContext(mc)
	pcontext.FillCorrelationId(pc)
	h.Handle(pc)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status NotFound, got %d", recorder.Code)
	}
}

// TestHandleContentFound tests the Handle method of the handler when the content is found.
// The request is simulated to be a GET manifest request from a peer peerd pod.
// The handler should look for the content in the local containerd store and having found it, return a 200 with the context.
func TestHandleContentOk(t *testing.T) {
	mr := mocks.NewMockRouter(nil)
	ref, err := containerd.ParseReference("k8s.io/library/alpine/sha256:bb863d6b95453b6b10dfaa1a52cb53f453d9a97ee775808ebaf6533bb4c9bb30", "sha256:bb863d6b95453b6b10dfaa1a52cb53f453d9a97ee775808ebaf6533bb4c9bb30")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ms := containerd.NewMockContainerdStore([]containerd.Reference{ref})

	h, err := New(ctxWithMetrics, mr, ms)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	recorder := httptest.NewRecorder()
	mc, _ := gin.CreateTestContext(recorder)
	req, err := http.NewRequest("GET", "http://127.0.0.1:5000/v2/library/alpine/manifests/sha256:bb863d6b95453b6b10dfaa1a52cb53f453d9a97ee775808ebaf6533bb4c9bb30?ns=k8s.io", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set(pcontext.P2PHeaderKey, "true")
	mc.Request = req

	pc := pcontext.FromContext(mc)
	pcontext.FillCorrelationId(pc)
	h.Handle(pc)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status OK, got %d", recorder.Code)
	}

	// Ensure body size is greater than 0
	if recorder.Body.Len() == 0 {
		t.Fatalf("expected non-empty body, got empty")
	}

	// Ensure content type is set to application/vnd.oci.image.manifest.v1+json
	if recorder.Header().Get("Content-Type") != "application/vnd.oci.image.manifest.v1+json" {
		t.Fatalf("expected content type application/vnd.oci.image.manifest.v1+json, got %s", recorder.Header().Get("Content-Type"))
	}

	// Ensure content length is set
	if recorder.Header().Get("Content-Length") == "" {
		t.Fatalf("expected content length, got empty")
	}
}

func TestFillDefault(t *testing.T) {
	mr := mocks.NewMockRouter(nil)
	ms := containerd.NewMockContainerdStore(nil)

	h, err := New(ctxWithMetrics, mr, ms)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	recorder := httptest.NewRecorder()
	mc, _ := gin.CreateTestContext(recorder)

	req, err := http.NewRequest("GET", "http://127.0.0.1:5000/v2/library/alpine/manifests/3.18.0", nil)
	if err != nil {
		t.Fatal(err)
	}
	mc.Request = req

	pmc := pcontext.FromContext(mc)

	err = h.fill(pmc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	gotNs := mc.GetString(pcontext.NamespaceCtxKey)
	if gotNs != "docker.io" {
		t.Fatalf("expected docker.io, got %s", gotNs)
	}

	if mc.GetString(pcontext.ReferenceCtxKey) != "docker.io/library/alpine:3.18.0" {
		t.Fatalf("expected library/alpine, got %s", mc.GetString(pcontext.ReferenceCtxKey))
	}

	if mc.GetString(pcontext.DigestCtxKey) != "" {
		t.Fatalf("expected empty string, got %s", mc.GetString(pcontext.DigestCtxKey))
	}

	gotRefType, ok := mc.Get(pcontext.RefTypeCtxKey)
	if !ok {
		t.Fatalf("expected reference type, got nil")
	}

	if gotRefType.(distribution.ReferenceType) != distribution.ReferenceTypeManifest {
		t.Fatalf("expected Manifest, got %v", gotRefType)
	}

	mc2, _ := gin.CreateTestContext(recorder)
	req2, err := http.NewRequest("GET", "http://127.0.0.1:5000/v2/library/alpine/manifests/sha256:bb863d6b95453b6b10dfaa1a52cb53f453d9a97ee775808ebaf6533bb4c9bb30?ns=k8s.io", nil)
	if err != nil {
		t.Fatal(err)
	}
	mc2.Request = req2

	pmc2 := pcontext.FromContext(mc2)

	err = h.fill(pmc2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mc2.GetString(pcontext.NamespaceCtxKey) != "k8s.io" {
		t.Fatalf("expected k8s.io, got %s", mc2.GetString(pcontext.NamespaceCtxKey))
	}

	if mc2.GetString(pcontext.ReferenceCtxKey) != "" {
		t.Fatalf("expected empty string, got %s", mc2.GetString(pcontext.ReferenceCtxKey))
	}

	if mc2.GetString(pcontext.DigestCtxKey) != "sha256:bb863d6b95453b6b10dfaa1a52cb53f453d9a97ee775808ebaf6533bb4c9bb30" {
		t.Fatalf("expected sha256:bb863d6b95453b6b10dfaa1a52cb53f453d9a97ee775808ebaf6533bb4c9bb30, got %s", mc2.GetString(pcontext.DigestCtxKey))
	}

	gotRefType, ok = mc2.Get(pcontext.RefTypeCtxKey)
	if !ok {
		t.Fatalf("expected reference type, got nil")
	}

	if gotRefType.(distribution.ReferenceType) != distribution.ReferenceTypeManifest {
		t.Fatalf("expected Manifest, got %v", gotRefType)
	}
}
