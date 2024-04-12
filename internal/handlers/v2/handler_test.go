package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	p2pcontext "github.com/azure/peerd/internal/context"
	"github.com/azure/peerd/pkg/containerd"
	"github.com/azure/peerd/pkg/discovery/routing/tests"
	"github.com/azure/peerd/pkg/metrics"
	"github.com/azure/peerd/pkg/oci/distribution"
	"github.com/gin-gonic/gin"
)

var (
	ctxWithMetrics, _ = metrics.WithContext(context.Background(), "test", "peerd")
)

func TestNew(t *testing.T) {
	mr := tests.NewMockRouter(nil)
	ms := containerd.NewMockContainerdStore(nil)

	h, err := New(ctxWithMetrics, mr, ms)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if h == nil {
		t.Fatalf("unexpected nil handler")
	}
}

func TestFillDefault(t *testing.T) {
	mr := tests.NewMockRouter(nil)
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

	err = h.fill(mc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	gotNs := mc.GetString(p2pcontext.NamespaceCtxKey)
	if gotNs != "docker.io" {
		t.Fatalf("expected docker.io, got %s", gotNs)
	}

	if mc.GetString(p2pcontext.ReferenceCtxKey) != "docker.io/library/alpine:3.18.0" {
		t.Fatalf("expected library/alpine, got %s", mc.GetString(p2pcontext.ReferenceCtxKey))
	}

	if mc.GetString(p2pcontext.DigestCtxKey) != "" {
		t.Fatalf("expected empty string, got %s", mc.GetString(p2pcontext.DigestCtxKey))
	}

	gotRefType, ok := mc.Get(p2pcontext.RefTypeCtxKey)
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

	err = h.fill(mc2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mc2.GetString(p2pcontext.NamespaceCtxKey) != "k8s.io" {
		t.Fatalf("expected k8s.io, got %s", mc2.GetString(p2pcontext.NamespaceCtxKey))
	}

	if mc2.GetString(p2pcontext.ReferenceCtxKey) != "" {
		t.Fatalf("expected empty string, got %s", mc2.GetString(p2pcontext.ReferenceCtxKey))
	}

	if mc2.GetString(p2pcontext.DigestCtxKey) != "sha256:bb863d6b95453b6b10dfaa1a52cb53f453d9a97ee775808ebaf6533bb4c9bb30" {
		t.Fatalf("expected sha256:bb863d6b95453b6b10dfaa1a52cb53f453d9a97ee775808ebaf6533bb4c9bb30, got %s", mc2.GetString(p2pcontext.DigestCtxKey))
	}

	gotRefType, ok = mc2.Get(p2pcontext.RefTypeCtxKey)
	if !ok {
		t.Fatalf("expected reference type, got nil")
	}

	if gotRefType.(distribution.ReferenceType) != distribution.ReferenceTypeManifest {
		t.Fatalf("expected Manifest, got %v", gotRefType)
	}
}
