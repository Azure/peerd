// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	p2pcontext "github.com/azure/peerd/internal/context"
	"github.com/azure/peerd/internal/files"
	"github.com/azure/peerd/internal/files/store"
	"github.com/azure/peerd/pkg/discovery/routing/tests"
	"github.com/azure/peerd/pkg/metrics"
	"github.com/gin-gonic/gin"
)

var (
	hostAndPath       = "https://avtakkartest.blob.core.windows.net/d18c7a64c5158179-ff8cb2f639ff44879c12c94361a746d0-782b855128//docker/registry/v2/blobs/sha256/d1/d18c7a64c5158179bdee531a663c5b487de57ff17cff3af29a51c7e70b491d9d/data"
	query             = "?se=2023-09-20T01%3A14%3A49Z&sig=m4Cr%2BYTZHZQlN5LznY7nrTQ4LCIx2OqnDDM3Dpedbhs%3D&sp=r&spr=https&sr=b&sv=2018-03-28&regid=01031d61e1024861afee5d512651eb9f"
	u                 = hostAndPath + query
	ctxWithMetrics, _ = metrics.WithContext(context.Background(), "test", "peerd")
)

func TestPartialContentResponseInP2PMode(t *testing.T) {
	files.CacheBlockSize = 10
	// Create a new request with a URL that has a query string.
	req, err := http.NewRequest("GET", "http://127.0.0.1:5000/blobs/"+u, nil)
	if err != nil {
		t.Fatal(err)
	}
	expRange := fmt.Sprintf("bytes=%v-%v", 12, 100)
	req.Header.Set("Range", expRange)
	req.Header.Set(p2pcontext.P2PHeaderKey, "true")

	expD := "sha256:d18c7a64c5158179bdee531a663c5b487de57ff17cff3af29a51c7e70b491d9d"

	// Create a new context with the request.
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = req
	ctx.Params = []gin.Param{
		{Key: "url", Value: hostAndPath},
	}

	store.PrefetchWorkers = 0 // turn off prefetching
	s, err := store.NewMockStore(ctxWithMetrics, tests.NewMockRouter(make(map[string][]string)))
	if err != nil {
		t.Fatal(err)
	}

	h := New(ctxWithMetrics, s)

	// Write the chunk file.
	content := newRandomStringN(10)

	s.Cache().PutSize(expD, 200)
	// nolint:errcheck
	s.Cache().GetOrCreate(expD, 10, 10, func() ([]byte, error) {
		return []byte(content), nil
	})

	h.Handle(ctx)
	resp := recorder.Result()

	if resp.StatusCode != http.StatusPartialContent {
		t.Errorf("expected %v, got %v", http.StatusOK, ctx.Writer.Status())
	}

	ret, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(ret) != content[2:] {
		t.Errorf("expected %v, got %v", content[2:], ret)
	}
}

func TestNotFoundInP2PMode(t *testing.T) {
	// Create a new request with a URL that has a query string.
	req, err := http.NewRequest("GET", "http://127.0.0.1:5000/blobs/"+u, nil)
	if err != nil {
		t.Fatal(err)
	}
	expRange := fmt.Sprintf("bytes=%v-%v", files.CacheBlockSize, files.CacheBlockSize+172)
	req.Header.Set("Range", expRange)
	req.Header.Set(p2pcontext.P2PHeaderKey, "true")

	// Create a new context with the request.
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req
	ctx.Params = []gin.Param{
		{Key: "url", Value: hostAndPath},
	}

	store.PrefetchWorkers = 0 // turn off prefetching
	s, err := store.NewFilesStore(ctxWithMetrics, tests.NewMockRouter(make(map[string][]string)))
	if err != nil {
		t.Fatal(err)
	}

	h := New(ctxWithMetrics, s)

	h.Handle(ctx)
	if ctx.Writer.Status() != http.StatusNotFound {
		t.Errorf("expected %v, got %v", http.StatusNotFound, ctx.Writer.Status())
	}
}

func TestFill(t *testing.T) {
	// Create a new request with a URL that has a query string.
	req, err := http.NewRequest("GET", "http://127.0.0.1:5000/blobs/"+u, nil)
	if err != nil {
		t.Fatal(err)
	}
	expRange := fmt.Sprintf("bytes=%v-%v", files.CacheBlockSize, files.CacheBlockSize+172)
	req.Header.Set("Range", expRange)
	req.Header.Set(p2pcontext.P2PHeaderKey, "true")

	expD := "sha256:d18c7a64c5158179bdee531a663c5b487de57ff17cff3af29a51c7e70b491d9d"
	expK := fmt.Sprintf("%v_%v", expD, files.CacheBlockSize)

	// Create a new context with the request.
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req
	ctx.Params = []gin.Param{
		{Key: "url", Value: hostAndPath},
	}

	store.PrefetchWorkers = 0 // turn off prefetching
	s, err := store.NewFilesStore(ctxWithMetrics, tests.NewMockRouter(make(map[string][]string)))
	if err != nil {
		t.Fatal(err)
	}

	h := New(ctxWithMetrics, s)

	err = h.fill(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if ctx.GetString(p2pcontext.FileChunkCtxKey) != expK {
		t.Errorf("expected %v, got %v", expK, ctx.GetString(p2pcontext.FileChunkCtxKey))
	}

	if ctx.GetString(p2pcontext.BlobRangeCtxKey) != expRange {
		t.Errorf("expected %v, got %v", expRange, ctx.GetString(p2pcontext.BlobRangeCtxKey))
	}

	if ctx.GetString(p2pcontext.BlobUrlCtxKey) != hostAndPath+query {
		t.Errorf("expected %v, got %v", hostAndPath+query, ctx.GetString(p2pcontext.BlobUrlCtxKey))
	}
}
