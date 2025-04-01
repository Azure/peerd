// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package containerd

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	pcontext "github.com/azure/peerd/pkg/context"
	"github.com/azure/peerd/pkg/oci/distribution"
	"github.com/gin-gonic/gin"
)

func TestNewRegistry(t *testing.T) {
	// Create a new registry
	r := NewRegistry(NewMockContainerdStore(nil))

	if r == nil {
		t.Fatal("expected registry")
	}
}

func TestHandleManifestTooLarge(t *testing.T) {
	img, err := ParseReference("library/alpine:3.18.0", "sha256:bb863d6b95453b6b10dfaa1a52cb53f453d9a97ee775808ebaf6533bb4c9bb30")
	if err != nil {
		t.Fatal(err)
	}
	refs := []Reference{img}
	ms := &MockContainerdStore{
		validRefs:        nil,
		sizeTooLargeRefs: refs,
	}
	r := NewRegistry(ms)
	mr := httptest.NewRecorder()
	mc, _ := gin.CreateTestContext(mr)

	req, err := http.NewRequest("GET", "http://127.0.0.1:5000/v2/library/alpine/manifests/3.18.0", nil)
	if err != nil {
		t.Fatal(err)
	}
	mc.Request = req

	pmc := pcontext.Context{Context: mc}

	// Manifest too large.
	r.handleManifest(pmc, "sha256:bb863d6b95453b6b10dfaa1a52cb53f453d9a97ee775808ebaf6533bb4c9bb30")
	if mr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", mr.Code)
	}
}

func TestHandleManifestNotFound(t *testing.T) {
	ms := &MockContainerdStore{
		validRefs:        nil,
		sizeTooLargeRefs: nil,
	}
	r := NewRegistry(ms)
	mr := httptest.NewRecorder()
	mc, _ := gin.CreateTestContext(mr)

	req, err := http.NewRequest("GET", "http://127.0.0.1:5000/v2/library/alpine/manifests/3.18.0", nil)
	if err != nil {
		t.Fatal(err)
	}
	mc.Request = req

	pmc := pcontext.Context{Context: mc}

	// Manifest not found.
	r.handleManifest(pmc, "sha256:bb863d6b95453b6b10dfaa1a52cb53f453d9a97ee775808ebaf6533bb4c9bb31")
	if mr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", mr.Code)
	}
}

func TestHandleManifestInvalidBytes(t *testing.T) {
	img, err := ParseReference("library/alpine:3.18.0", "sha256:bb863d6b95453b6b10dfaa1a52cb53f453d9a97ee775808ebaf6533bb4c9bb30")
	if err != nil {
		t.Fatal(err)
	}
	refs := []Reference{img}
	ms := &MockContainerdStore{
		invalidBytesRefs: refs,
	}
	r := NewRegistry(ms)
	mr := httptest.NewRecorder()
	mc, _ := gin.CreateTestContext(mr)

	req, err := http.NewRequest("GET", "http://127.0.0.1:5000/v2/library/alpine/manifests/3.18.0", nil)
	if err != nil {
		t.Fatal(err)
	}
	mc.Request = req

	pmc := pcontext.Context{Context: mc}

	// Error while reading bytes.
	r.handleManifest(pmc, "sha256:bb863d6b95453b6b10dfaa1a52cb53f453d9a97ee775808ebaf6533bb4c9bb30")
	if mr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", mr.Code)
	}
}

func TestHandleManifestHead(t *testing.T) {
	img, err := ParseReference("library/alpine:3.18.0", "sha256:bb863d6b95453b6b10dfaa1a52cb53f453d9a97ee775808ebaf6533bb4c9bb30")
	if err != nil {
		t.Fatal(err)
	}
	refs := []Reference{img}

	ms := NewMockContainerdStore(refs)

	r := NewRegistry(ms)

	mr := httptest.NewRecorder()
	mc, _ := gin.CreateTestContext(mr)

	req, err := http.NewRequest("HEAD", "http://127.0.0.1:5000/v2/library/alpine/manifests/3.18.0", nil)
	if err != nil {
		t.Fatal(err)
	}

	mc.Request = req

	pmc := pcontext.Context{Context: mc}

	r.handleManifest(pmc, "sha256:bb863d6b95453b6b10dfaa1a52cb53f453d9a97ee775808ebaf6533bb4c9bb30")

	if mr.Code != 200 {
		t.Fatalf("expected 200, got %d", mr.Code)
	}

	if mr.Body.String() != "" {
		t.Fatalf("expected no body, got %s", mr.Body.String())
	}

	if mr.Header().Get(contentTypeHeader) != "application/vnd.oci.image.manifest.v1+json" {
		t.Fatalf("expected application/vnd.oci.image.manifest.v1+json, got %s", mr.Header().Get(contentTypeHeader))
	}

	if mr.Header().Get(contentLengthHeader) != "258" {
		t.Fatalf("expected 258, got %s", mr.Header().Get(contentLengthHeader))
	}
}

func TestHandleManifestWriteFailure(t *testing.T) {
	img, err := ParseReference("library/alpine:3.18.0", "sha256:bb863d6b95453b6b10dfaa1a52cb53f453d9a97ee775808ebaf6533bb4c9bb30")
	if err != nil {
		t.Fatal(err)
	}
	refs := []Reference{img}

	ms := NewMockContainerdStore(refs)

	r := NewRegistry(ms)

	mr := httptest.NewRecorder()
	failingWriter := &failingResponseWriter{
		ResponseWriter: mr,
		failWrite:      true,
	}
	mc, _ := gin.CreateTestContext(failingWriter)

	req, err := http.NewRequest("GET", "http://127.0.0.1:5000/v2/library/alpine/manifests/3.18.0", nil)
	if err != nil {
		t.Fatal(err)
	}

	mc.Request = req

	pmc := pcontext.Context{Context: mc}

	r.handleManifest(pmc, "sha256:bb863d6b95453b6b10dfaa1a52cb53f453d9a97ee775808ebaf6533bb4c9bb30")

	if mr.Body.String() != "" {
		t.Fatalf("expected empty body, got %s", mr.Body.String())
	}
}

func TestHandleManifest(t *testing.T) {
	img, err := ParseReference("library/alpine:3.18.0", "sha256:bb863d6b95453b6b10dfaa1a52cb53f453d9a97ee775808ebaf6533bb4c9bb30")
	if err != nil {
		t.Fatal(err)
	}
	refs := []Reference{img}

	ms := NewMockContainerdStore(refs)

	r := NewRegistry(ms)

	mr := httptest.NewRecorder()
	mc, _ := gin.CreateTestContext(mr)

	req, err := http.NewRequest("GET", "http://127.0.0.1:5000/v2/library/alpine/manifests/3.18.0", nil)
	if err != nil {
		t.Fatal(err)
	}

	mc.Request = req

	pmc := pcontext.Context{Context: mc}

	r.handleManifest(pmc, "sha256:bb863d6b95453b6b10dfaa1a52cb53f453d9a97ee775808ebaf6533bb4c9bb30")

	if mr.Code != 200 {
		t.Fatalf("expected 200, got %d", mr.Code)
	}

	if mr.Body.String() != testManifestBlob {
		t.Fatalf("expected %s, got %s", testManifestBlob, mr.Body.String())
	}

	if mr.Header().Get(contentTypeHeader) != "application/vnd.oci.image.manifest.v1+json" {
		t.Fatalf("expected application/vnd.oci.image.manifest.v1+json, got %s", mr.Header().Get(contentTypeHeader))
	}

	if mr.Header().Get(contentLengthHeader) != "258" {
		t.Fatalf("expected 258, got %s", mr.Header().Get(contentLengthHeader))
	}
}

func TestHandleBlobNotFound(t *testing.T) {
	ms := &MockContainerdStore{}
	r := NewRegistry(ms)
	mr := httptest.NewRecorder()
	mc, _ := gin.CreateTestContext(mr)
	req, err := http.NewRequest("GET", "http://127.0.0.1:5000/v2/library/alpine/blobs/sha256:blob", nil)
	if err != nil {
		t.Fatal(err)
	}

	mc.Request = req

	pmc := pcontext.Context{Context: mc}

	r.handleBlob(pmc, "sha256:blob")
	if mr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", mr.Code)
	}
	if mr.Body.String() != "" {
		t.Fatalf("expected empty body, got %s", mr.Body.String())
	}
}

func TestHandleBlobHead(t *testing.T) {
	img, err := ParseReference("library/alpine:3.18.0", "sha256:blob")
	if err != nil {
		t.Fatal(err)
	}
	refs := []Reference{img}

	ms := NewMockContainerdStore(refs)

	r := NewRegistry(ms)

	mr := httptest.NewRecorder()
	mc, _ := gin.CreateTestContext(mr)

	req, err := http.NewRequest("HEAD", "http://127.0.0.1:5000/v2/library/alpine/blobs/sha256:blob", nil)
	if err != nil {
		t.Fatal(err)
	}

	mc.Request = req

	pmc := pcontext.Context{Context: mc}

	r.handleBlob(pmc, "sha256:blob")

	if mr.Code != 200 {
		t.Fatalf("expected 200, got %d", mr.Code)
	}

	if mr.Body.String() != "" {
		t.Fatalf("expected empty body, got %s", mr.Body.String())
	}

	if mr.Header().Get(contentLengthHeader) != "258" {
		t.Fatalf("expected 258, got %s", mr.Header().Get(contentLengthHeader))
	}

	if mr.Header().Get(dockerContentDigestHeader) != "sha256:blob" {
		t.Fatalf("expected sha256:blob, got %s", mr.Header().Get(dockerContentDigestHeader))
	}
}

func TestHandleBlobFailedWrite(t *testing.T) {
	img, err := ParseReference("library/alpine:3.18.0", "sha256:blob")
	if err != nil {
		t.Fatal(err)
	}
	refs := []Reference{img}

	ms := NewMockContainerdStore(refs)

	r := NewRegistry(ms)

	mr := httptest.NewRecorder()
	failingWriter := &failingResponseWriter{
		ResponseWriter: mr,
		failWrite:      true,
	}
	mc, _ := gin.CreateTestContext(failingWriter)

	req, err := http.NewRequest("GET", "http://127.0.0.1:5000/v2/library/alpine/blobs/sha256:blob", nil)
	if err != nil {
		t.Fatal(err)
	}

	mc.Request = req

	pmc := pcontext.Context{Context: mc}

	r.handleBlob(pmc, "sha256:blob")

	if mr.Body.String() != "" {
		t.Fatalf("expected empty body, got %s", mr.Body.String())
	}
}

func TestHandleBlob(t *testing.T) {
	img, err := ParseReference("library/alpine:3.18.0", "sha256:blob")
	if err != nil {
		t.Fatal(err)
	}
	refs := []Reference{img}

	ms := NewMockContainerdStore(refs)

	r := NewRegistry(ms)

	mr := httptest.NewRecorder()
	mc, _ := gin.CreateTestContext(mr)

	req, err := http.NewRequest("GET", "http://127.0.0.1:5000/v2/library/alpine/blobs/sha256:blob", nil)
	if err != nil {
		t.Fatal(err)
	}

	mc.Request = req

	pmc := pcontext.Context{Context: mc}

	r.handleBlob(pmc, "sha256:blob")

	if mr.Code != 200 {
		t.Fatalf("expected 200, got %d", mr.Code)
	}

	if mr.Body.String() != testManifestBlob {
		t.Fatalf("expected %s, got %s", testManifestBlob, mr.Body.String())
	}

	if mr.Header().Get(contentLengthHeader) != "258" {
		t.Fatalf("expected 258, got %s", mr.Header().Get(contentLengthHeader))
	}

	if mr.Header().Get(dockerContentDigestHeader) != "sha256:blob" {
		t.Fatalf("expected sha256:blob, got %s", mr.Header().Get(dockerContentDigestHeader))
	}
}

func TestHandleBadDigest(t *testing.T) {
	ms := NewMockContainerdStore(nil)

	r := NewRegistry(ms)

	mr := httptest.NewRecorder()
	mc, _ := gin.CreateTestContext(mr)

	req, err := http.NewRequest("GET", "http://127.0.0.1:5000/v2/library/alpine/manifests/3.18.0", nil)
	if err != nil {
		t.Fatal(err)
	}

	mc.Request = req
	mc.Set(pcontext.DigestCtxKey, "sha256bb863d6b95453b6b10dfaa1a52cb53f453d9a97ee775808ebaf6533bb4c9bb30")
	mc.Set(pcontext.ReferenceCtxKey, "library/alpine:3.18.0")
	mc.Set(pcontext.RefTypeCtxKey, distribution.ReferenceType(distribution.ReferenceTypeManifest))

	pmc := pcontext.Context{Context: mc}

	r.Handle(pmc)

	if mr.Code != 400 {
		t.Fatalf("expected 400, got %d", mr.Code)
	}
}

func TestHandleMissingRefType(t *testing.T) {
	ms := NewMockContainerdStore(nil)

	r := NewRegistry(ms)

	mr := httptest.NewRecorder()
	mc, _ := gin.CreateTestContext(mr)

	req, err := http.NewRequest("GET", "http://127.0.0.1:5000/v2/library/alpine/manifests/3.18.0", nil)
	if err != nil {
		t.Fatal(err)
	}

	mc.Request = req
	mc.Set(pcontext.DigestCtxKey, "sha256:bb863d6b95453b6b10dfaa1a52cb53f453d9a97ee775808ebaf6533bb4c9bb30")
	mc.Set(pcontext.ReferenceCtxKey, "library/alpine:3.18.0")

	pmc := pcontext.Context{Context: mc}

	r.Handle(pmc)

	if mr.Code != 500 {
		t.Fatalf("expected 500, got %d", mr.Code)
	}
}

func TestHandleM(t *testing.T) {
	img, err := ParseReference("library/alpine:3.18.0", "sha256:bb863d6b95453b6b10dfaa1a52cb53f453d9a97ee775808ebaf6533bb4c9bb30")
	if err != nil {
		t.Fatal(err)
	}
	refs := []Reference{img}

	ms := NewMockContainerdStore(refs)

	r := NewRegistry(ms)

	mr := httptest.NewRecorder()
	mc, _ := gin.CreateTestContext(mr)

	req, err := http.NewRequest("GET", "http://127.0.0.1:5000/v2/library/alpine/manifests/3.18.0", nil)
	if err != nil {
		t.Fatal(err)
	}

	mc.Request = req
	mc.Set(pcontext.DigestCtxKey, "sha256:bb863d6b95453b6b10dfaa1a52cb53f453d9a97ee775808ebaf6533bb4c9bb30")
	mc.Set(pcontext.ReferenceCtxKey, "library/alpine:3.18.0")
	mc.Set(pcontext.RefTypeCtxKey, distribution.ReferenceType(distribution.ReferenceTypeManifest))

	pmc := pcontext.Context{Context: mc}

	r.Handle(pmc)

	if mr.Code != 200 {
		t.Fatalf("expected 200, got %d", mr.Code)
	}

	if mr.Body.String() != testManifestBlob {
		t.Fatalf("expected %s, got %s", testManifestBlob, mr.Body.String())
	}

	if mr.Header().Get(contentTypeHeader) != "application/vnd.oci.image.manifest.v1+json" {
		t.Fatalf("expected application/vnd.oci.image.manifest.v1+json, got %s", mr.Header().Get(contentTypeHeader))
	}

	if mr.Header().Get(contentLengthHeader) != "258" {
		t.Fatalf("expected 258, got %s", mr.Header().Get(contentLengthHeader))
	}
}

func TestHandleB(t *testing.T) {
	img, err := ParseReference("library/alpine:3.18.0", "sha256:bb863d6b95453b6b10dfaa1a52cb53f453d9a97ee775808ebaf6533bb4c9bb30")
	if err != nil {
		t.Fatal(err)
	}
	refs := []Reference{img}

	ms := NewMockContainerdStore(refs)

	r := NewRegistry(ms)

	mr := httptest.NewRecorder()
	mc, _ := gin.CreateTestContext(mr)

	req, err := http.NewRequest("GET", "http://127.0.0.1:5000/v2/library/alpine/blobs/sha256:bb863d6b95453b6b10dfaa1a52cb53f453d9a97ee775808ebaf6533bb4c9bb30", nil)
	if err != nil {
		t.Fatal(err)
	}

	mc.Request = req
	mc.Set(pcontext.DigestCtxKey, "sha256:bb863d6b95453b6b10dfaa1a52cb53f453d9a97ee775808ebaf6533bb4c9bb30")
	//mc.Set(pcontext.ReferenceCtxKey, "library/alpine")
	mc.Set(pcontext.RefTypeCtxKey, distribution.ReferenceType(distribution.ReferenceTypeBlob))

	pmc := pcontext.Context{Context: mc}

	r.Handle(pmc)

	if mr.Code != 200 {
		t.Fatalf("expected 200, got %d", mr.Code)
	}

	if mr.Body.String() != testManifestBlob {
		t.Fatalf("expected %s, got %s", testManifestBlob, mr.Body.String())
	}

	if mr.Header().Get(contentLengthHeader) != "258" {
		t.Fatalf("expected 258, got %s", mr.Header().Get(contentLengthHeader))
	}
}

func TestHandleByTag(t *testing.T) {
	img, err := ParseReference("library/alpine:3.18.0", "sha256:bb863d6b95453b6b10dfaa1a52cb53f453d9a97ee775808ebaf6533bb4c9bb30")
	if err != nil {
		t.Fatal(err)
	}
	refs := []Reference{img}

	ms := NewMockContainerdStore(refs)

	r := NewRegistry(ms)

	mr := httptest.NewRecorder()
	mc, _ := gin.CreateTestContext(mr)

	req, err := http.NewRequest("GET", "http://127.0.0.1:5000/v2/library/alpine/manifests/3.18.0", nil)
	if err != nil {
		t.Fatal(err)
	}

	mc.Request = req
	//mc.Set(pcontext.DigestCtxKey, "sha256:bb863d6b95453b6b10dfaa1a52cb53f453d9a97ee775808ebaf6533bb4c9bb30")
	mc.Set(pcontext.ReferenceCtxKey, "library/alpine:3.18.0")
	mc.Set(pcontext.RefTypeCtxKey, distribution.ReferenceType(distribution.ReferenceTypeManifest))

	pmc := pcontext.Context{Context: mc}

	r.Handle(pmc)

	if mr.Code != 200 {
		t.Fatalf("expected 200, got %d", mr.Code)
	}

	if mr.Body.String() != testManifestBlob {
		t.Fatalf("expected %s, got %s", testManifestBlob, mr.Body.String())
	}

	if mr.Header().Get(contentTypeHeader) != "application/vnd.oci.image.manifest.v1+json" {
		t.Fatalf("expected application/vnd.oci.image.manifest.v1+json, got %s", mr.Header().Get(contentTypeHeader))
	}

	if mr.Header().Get(contentLengthHeader) != "258" {
		t.Fatalf("expected 258, got %s", mr.Header().Get(contentLengthHeader))
	}
}

type failingResponseWriter struct {
	http.ResponseWriter
	failWrite bool
}

func (w *failingResponseWriter) Write(data []byte) (int, error) {
	if w.failWrite {
		return 0, fmt.Errorf("simulated write failure")
	}
	return w.ResponseWriter.Write(data)
}
