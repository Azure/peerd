// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package remote

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	p2pcontext "github.com/azure/peerd/internal/context"
	"github.com/azure/peerd/internal/routing/tests"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

var (
	hostAndPath = "https://avtakkartest.blob.core.windows.net/d18c7a64c5158179-ff8cb2f639ff44879c12c94361a746d0-782b855128//docker/registry/v2/blobs/sha256/d1/d18c7a64c5158179bdee531a663c5b487de57ff17cff3af29a51c7e70b491d9d/data"
	query       = "?se=2023-09-20T01%3A14%3A49Z&sig=m4Cr%2BYTZHZQlN5LznY7nrTQ4LCIx2OqnDDM3Dpedbhs%3D&sp=r&spr=https&sr=b&sv=2018-03-28&regid=01031d61e1024861afee5d512651eb9f"
	u           = hostAndPath + query
)

func TestPreadRemoteUpstream(t *testing.T) {
	// Setup
	m := map[string][]string{}
	key := "somekey"
	expected := "expected-result"
	peersTried := 0
	svr3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peersTried++
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer svr3.Close()
	svr2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peersTried++
		w.WriteHeader(http.StatusNotFound)
	}))
	defer svr2.Close()
	svr1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peersTried++
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer svr1.Close()
	val := []string{svr1.URL, svr2.URL, svr3.URL}
	m[key] = val

	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if "?"+r.URL.RawQuery == query {
			w.Header().Set("Content-Type", "application/octet-stream")
			// nolint:errcheck
			w.Write([]byte(expected))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer svr.Close()
	p := svr.URL + "/some-path"
	u := "http://127.0.0.1:5000/blobs/" + p + query
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		t.Fatal(err)
	}

	router := tests.NewMockRouter(m)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req
	c.Params = []gin.Param{
		{Key: "url", Value: p},
	}
	c.Set(p2pcontext.BlobUrlCtxKey, p2pcontext.BlobUrl(c))
	c.Set(p2pcontext.BlobRangeCtxKey, "bytes=0-10")
	c.Set(p2pcontext.FileChunkCtxKey, key)

	r := NewReader(c, router, 3, 500*time.Millisecond).(*reader)
	b := make([]byte, 10)

	// Test
	got, err := r.PreadRemote(b, 0)

	// Assert
	if err != nil {
		t.Fatal(err)
	} else if got != 10 {
		t.Fatalf("expected %v, got %v", 10, got)
	} else if string(b) != expected[:10] {
		t.Fatalf("expected %v, got %v", expected[:10], string(b))
	} else if peersTried != 3 {
		t.Fatalf("expected %v, got %v", 3, peersTried)
	}
}

func TestFstatRemote(t *testing.T) {
	m := map[string][]string{}

	expected := "expected-result"
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if "?"+r.URL.RawQuery == query {
			w.Header().Set("Content-Type", "application/octet-stream")
			// nolint:errcheck
			w.Write([]byte(expected))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer svr.Close()
	p := svr.URL + "/some-path"
	u := "http://127.0.0.1:5000/blobs/" + p + query
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		t.Fatal(err)
	}

	router := tests.NewMockRouter(m)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req
	c.Params = []gin.Param{
		{Key: "url", Value: p},
	}
	c.Set(p2pcontext.BlobUrlCtxKey, p2pcontext.BlobUrl(c))
	c.Set(p2pcontext.BlobRangeCtxKey, "bytes=0-0")

	r := NewReader(c, router, 3, 500*time.Millisecond).(*reader)

	got, err := r.FstatRemote()
	if err != nil {
		t.Fatal(err)
	} else if got != int64(len(expected)) {
		t.Fatalf("expected %v, got %v", len(expected), got)
	}
}

func TestFstatRemotePartialContent(t *testing.T) {
	m := map[string][]string{}

	expected := "expected-result"
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if "?"+r.URL.RawQuery == query {
			w.Header().Set("Content-Type", "application/octet-stream")
			// nolint:errcheck
			w.WriteHeader(http.StatusPartialContent)
			w.Header().Set("Content-Range", "bytes 0-10/10")
			// nolint:errcheck
			w.Write([]byte(expected))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer svr.Close()
	p := svr.URL + "/some-path"
	u := "http://127.0.0.1:5000/blobs/" + p + query
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		t.Fatal(err)
	}

	router := tests.NewMockRouter(m)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req
	c.Params = []gin.Param{
		{Key: "url", Value: p},
	}
	c.Set(p2pcontext.BlobUrlCtxKey, p2pcontext.BlobUrl(c))
	c.Set(p2pcontext.BlobRangeCtxKey, "bytes=0-0")

	r := NewReader(c, router, 3, 500*time.Millisecond).(*reader)

	got, err := r.FstatRemote()
	if err != nil {
		t.Fatal(err)
	} else if got != int64(len(expected)) {
		t.Fatalf("expected %v, got %v", len(expected), got)
	}
}

func TestP2pRetries(t *testing.T) {
	l := zerolog.Nop()
	m := map[string][]string{}
	key := "somekey"
	expected := "expected-result"
	svr3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		// nolint:errcheck
		w.Write([]byte(expected))
	}))
	defer svr3.Close()
	svr2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer svr2.Close()
	svr1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer svr1.Close()
	val := []string{svr1.URL, svr2.URL, svr3.URL}
	m[key] = val

	req, err := http.NewRequest("GET", "http://127.0.0.1:5000/blobs/"+u, nil)
	if err != nil {
		t.Fatal(err)
	}

	router := tests.NewMockRouter(m)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req
	r := NewReader(c, router, 3, 500*time.Millisecond).(*reader)
	b := make([]byte, 10)

	got, err := r.doP2p(l, key, 0, 10, operationPreadRemote, b)
	if err != nil {
		t.Fatal(err)
	}

	if got != 10 {
		t.Fatalf("expected %v, got %v", 10, got)
	} else if string(b) != expected[:10] {
		t.Fatalf("expected %v, got %v", expected[:10], string(b))
	}
}

func TestP2pSuccess(t *testing.T) {
	l := zerolog.Nop()
	m := map[string][]string{}
	key := "somekey"
	expected := "expected-result"
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		// nolint:errcheck
		w.Write([]byte(expected))
	}))
	defer svr.Close()
	val := []string{svr.URL}
	m[key] = val

	req, err := http.NewRequest("GET", "http://127.0.0.1:5000/blobs/"+u, nil)
	if err != nil {
		t.Fatal(err)
	}

	router := tests.NewMockRouter(m)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req
	r := NewReader(c, router, 3, 500*time.Millisecond).(*reader)
	b := make([]byte, 10)

	got, err := r.doP2p(l, key, 0, 10, operationPreadRemote, b)
	if err != nil {
		t.Fatal(err)
	}

	if got != 10 {
		t.Fatalf("expected %v, got %v", 10, got)
	} else if string(b) != expected[:10] {
		t.Fatalf("expected %v, got %v", expected[:10], string(b))
	}
}

func TestP2pPeerNotFound(t *testing.T) {
	l := zerolog.Nop()
	m := map[string][]string{}

	req, err := http.NewRequest("GET", "http://127.0.0.1:5000/blobs/"+u, nil)
	if err != nil {
		t.Fatal(err)
	}

	router := tests.NewMockRouter(m)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req

	r := NewReader(c, router, 3, 500*time.Millisecond).(*reader)

	b := make([]byte, 10)
	_, err = r.doP2p(l, "key", 0, 10, operationPreadRemote, b)
	if err == nil {
		t.Fatal("expected error")
	}

	if err != errPeerNotFound {
		t.Fatalf("expected %v, got %v", errPeerNotFound, err)
	}
}

func TestP2pNoInfiniteLoops(t *testing.T) {
	l := zerolog.Nop()
	m := map[string][]string{}
	key := "some-key"
	val := []string{"http://localhost"}
	m[key] = val

	req, err := http.NewRequest("GET", "http://127.0.0.1:5000/blobs/"+u, nil)
	if err != nil {
		t.Fatal(err)
	}

	router := tests.NewMockRouter(m)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req
	c.Request.Header.Add(p2pcontext.P2PHeaderKey, "true")

	r := NewReader(c, router, 3, 500*time.Millisecond).(*reader)

	b := make([]byte, 10)
	_, err = r.doP2p(l, key, 0, 10, operationPreadRemote, b)
	if err == nil {
		t.Fatal("expected error")
	}

	if err != errPeerNotFound {
		t.Fatalf("expected %v, got %v", errPeerNotFound, err)
	}
}
