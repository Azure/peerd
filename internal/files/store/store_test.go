package store

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	p2pcontext "github.com/azure/peerd/internal/context"
	"github.com/azure/peerd/internal/files"
	"github.com/azure/peerd/internal/routing/tests"
	"github.com/gin-gonic/gin"
	"github.com/opencontainers/go-digest"
)

var (
	hostAndPath = "https://avtakkartest.blob.core.windows.net/d18c7a64c5158179-ff8cb2f639ff44879c12c94361a746d0-782b855128//docker/registry/v2/blobs/sha256/d1/d18c7a64c5158179bdee531a663c5b487de57ff17cff3af29a51c7e70b491d9d/data"
	query       = "?se=2023-09-20T01%3A14%3A49Z&sig=m4Cr%2BYTZHZQlN5LznY7nrTQ4LCIx2OqnDDM3Dpedbhs%3D&sp=r&spr=https&sr=b&sv=2018-03-28&regid=01031d61e1024861afee5d512651eb9f"
	u           = hostAndPath + query
)

func TestOpenP2p(t *testing.T) {
	// Create a new request with a URL that has a query string.
	req, err := http.NewRequest("GET", "http://127.0.0.1:5000/blobs/"+u, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%v-%v", files.CacheBlockSize, files.CacheBlockSize+172))
	req.Header.Set(p2pcontext.P2PHeaderKey, "true")

	expD := "sha256:d18c7a64c5158179bdee531a663c5b487de57ff17cff3af29a51c7e70b491d9d"
	expK := fmt.Sprintf("%v%v%v", expD, files.FileChunkKeySep, files.CacheBlockSize)

	// Create a new context with the request.
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req
	ctx.Params = []gin.Param{
		{Key: "url", Value: hostAndPath},
	}
	ctx.Set(p2pcontext.FileChunkCtxKey, expK)

	PrefetchWorkers = 0 // turn off prefetching
	s, err := NewFilesStore(context.Background(), tests.NewMockRouter(make(map[string][]string)))
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.Open(ctx)
	if err != os.ErrNotExist {
		t.Errorf("expected %v, got %v", os.ErrNotExist, err)
	}
}

func TestOpenNonP2p(t *testing.T) {
	// Create a new request with a URL that has a query string.
	req, err := http.NewRequest("GET", "http://127.0.0.1:5000/blobs/"+u, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%v-%v", files.CacheBlockSize, files.CacheBlockSize+172))

	expD := "sha256:d18c7a64c5158179bdee531a663c5b487de57ff17cff3af29a51c7e70b491d9d"
	expK := fmt.Sprintf("%v%v%v", expD, files.FileChunkKeySep, files.CacheBlockSize)

	// Create a new context with the request.
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req
	ctx.Params = []gin.Param{
		{Key: "url", Value: hostAndPath},
	}
	ctx.Set(p2pcontext.FileChunkCtxKey, expK)

	PrefetchWorkers = 0 // turn off prefetching
	s, err := NewMockStore(context.Background(), tests.NewMockRouter(make(map[string][]string)))
	if err != nil {
		t.Fatal(err)
	}

	s.Cache().PutSize(expD, 200)

	_, err = s.Open(ctx)
	if err != nil {
		t.Fatal(err)
	}
}

func TestKey(t *testing.T) {
	// Create a new request with a URL that has a query string.
	req, err := http.NewRequest("GET", "http://127.0.0.1:5000/blobs/"+u, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%v-%v", files.CacheBlockSize, files.CacheBlockSize+172))

	expD := "sha256:d18c7a64c5158179bdee531a663c5b487de57ff17cff3af29a51c7e70b491d9d"
	expK := fmt.Sprintf("%v_%v", expD, files.CacheBlockSize)

	// Create a new context with the request.
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req
	ctx.Params = []gin.Param{
		{Key: "url", Value: hostAndPath},
	}

	s, err := NewFilesStore(context.Background(), tests.NewMockRouter(make(map[string][]string)))
	if err != nil {
		t.Fatal(err)
	}

	k, d, err := s.Key(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if k != expK {
		t.Errorf("expected key %s, got %s", expK, k)
	}

	if d != digest.Digest(expD) {
		t.Errorf("expected digest %s, got %s", expD, d)
	}
}

func TestSubscribe(t *testing.T) {
	s, err := NewFilesStore(context.Background(), tests.NewMockRouter(make(map[string][]string)))
	if err != nil {
		t.Fatal(err)
	}
	ch := s.Subscribe()
	if ch == nil {
		t.Fatal("expected channel, got nil")
	}
}
