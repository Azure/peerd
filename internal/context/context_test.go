// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package context

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

var (
	hostAndPath = "https://avtakkartest.blob.core.windows.net/d18c7a64c5158179-ff8cb2f639ff44879c12c94361a746d0-782b855128//docker/registry/v2/blobs/sha256/d1/d18c7a64c5158179bdee531a663c5b487de57ff17cff3af29a51c7e70b491d9d/data"
	query       = "?se=2023-09-20T01%3A14%3A49Z&sig=m4Cr%2BYTZHZQlN5LznY7nrTQ4LCIx2OqnDDM3Dpedbhs%3D&sp=r&spr=https&sr=b&sv=2018-03-28&regid=01031d61e1024861afee5d512651eb9f"
	u           = hostAndPath + query
)

func TestLogger(t *testing.T) {
	req, err := http.NewRequest("GET", "http://127.0.0.1:5000/blobs/"+u, nil)
	if err != nil {
		t.Fatal(err)
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req

	l := Logger(c)
	if l.Info().Enabled() {
		t.Fatal("expected logger to be disabled")
	}

	testL := zerolog.New(os.Stdout).With().Timestamp().Logger()
	c.Set(LoggerCtxKey, &testL)

	l = Logger(c)
	if !l.Info().Enabled() {
		t.Fatal("expected logger to be enabled")
	}
}

func TestSetOutboundHeaders(t *testing.T) {
	// Create a new request with a URL that has a query string.
	req, err := http.NewRequest("GET", "http://127.0.0.1:5000/blobs/"+u, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a new context with the request.
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req
	FillCorrelationId(ctx)

	SetOutboundHeaders(req, ctx)

	if req.Header.Get(P2PHeaderKey) != "true" {
		t.Errorf("expected: %v, got: %v", "true", req.Header.Get(P2PHeaderKey))
	}

	if req.Header.Get(CorrelationHeaderKey) == "" {
		t.Errorf("expected: %v, got: %v", "not empty", req.Header.Get(CorrelationHeaderKey))
	}

	if req.Header.Get(NodeHeaderKey) != NodeName {
		t.Errorf("expected: %v, got: %v", NodeName, req.Header.Get(NodeHeaderKey))
	}
}

func TestMerge(t *testing.T) {

	ch1 := make(chan string, 10)
	ch2 := make(chan string)
	ch3 := make(chan string, 100)
	ch4 := make(chan string, 1000)

	mergedChan := Merge(ch1, ch2, ch3, ch4)

	// Write to the channels.
	go func() {
		for i := 0; i < 100; i++ {
			ch1 <- fmt.Sprintf("ch1-%d", i)
		}
		close(ch1)
	}()

	go func() {
		for i := 0; i < 100; i++ {
			ch2 <- fmt.Sprintf("ch2-%d", i)
		}
		close(ch2)
	}()

	go func() {
		for i := 0; i < 100; i++ {
			ch3 <- fmt.Sprintf("ch3-%d", i)
		}
		close(ch3)
	}()

	go func() {
		for i := 0; i < 100; i++ {
			ch4 <- fmt.Sprintf("ch4-%d", i)
		}
		close(ch4)
	}()

	// Read from the merged channel.
	total := 0
	for val := range mergedChan {
		if strings.HasPrefix(val, "ch1-") ||
			strings.HasPrefix(val, "ch2-") ||
			strings.HasPrefix(val, "ch3-") ||
			strings.HasPrefix(val, "ch4-") {
			total++
		} else {
			t.Errorf("unexpected value: %v", val)
		}
	}

	if total != 400 {
		t.Errorf("expected: %v, got: %v", 400, total)
	}
}

func TestBlobUrl(t *testing.T) {
	// Create a new request with a URL that has a query string.
	req, err := http.NewRequest("GET", "http://127.0.0.1:5000/blobs/"+u, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a new context with the request.
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req
	ctx.Params = []gin.Param{
		{Key: "url", Value: hostAndPath},
	}

	// Call BlobUrl and verify the result.
	got := BlobUrl(ctx)
	if got != u {
		t.Errorf("expected: %v, got: %v", u, got)
	}
}

func TestFillCorrelationId(t *testing.T) {
	// Create a new request without any correlation ID headers.
	req, err := http.NewRequest("GET", "http://127.0.0.1:5000/blobs/fdsfsdsd", nil)
	if err != nil {
		t.Fatal(err)
	}

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	FillCorrelationId(ctx)
	cid, ok := ctx.Get(CorrelationIdCtxKey)
	if !ok || cid == "" {
		t.Fatal("expected correlation ID to be set")
	}

	sample := uuid.New().String()

	ctx, _ = gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req
	ctx.Request.Header.Set(CorrelationHeaderKey, sample)
	FillCorrelationId(ctx)
	cid, ok = ctx.Get(CorrelationIdCtxKey)
	if !ok || cid == "" {
		t.Fatal("expected correlation ID to be set")
	} else if cid != sample {
		t.Errorf("expected: %v, got: %v", sample, cid)
	}
}

func TestIsRequestFromPeer(t *testing.T) {
	// Create a new request without any correlation ID headers.
	req, err := http.NewRequest("GET", "http://127.0.0.1:5000/blobs/fdsfsdsd", nil)
	if err != nil {
		t.Fatal(err)
	}

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	if IsRequestFromAPeer(ctx) {
		t.Fatal("expected request to not be from a peer")
	}

	ctx.Request.Header.Set(P2PHeaderKey, "true")
	if !IsRequestFromAPeer(ctx) {
		t.Fatal("expected request to be from a peer")
	}
}

func TestRangeStartIndex(t *testing.T) {
	for _, tc := range []struct {
		name          string
		r             string
		want          int64
		expectedError string
	}{
		{
			name:          "no range header",
			r:             "",
			want:          0,
			expectedError: "no range header",
		},
		{
			name:          "invalid range format",
			r:             "bytes=0",
			want:          0,
			expectedError: "invalid range format",
		},
		{
			name:          "invalid range format",
			r:             "bytes=0-",
			want:          0,
			expectedError: "invalid range format",
		},
		{
			name:          "invalid range format",
			r:             "bytes=0-100-200",
			want:          0,
			expectedError: "invalid range format",
		},
		{
			name:          "valid range format",
			r:             "bytes=91-100",
			want:          91,
			expectedError: "",
		},
		{
			name:          "invalid range format",
			r:             "count=91-100",
			want:          0,
			expectedError: "invalid range format",
		},
		{
			name:          "invalid range format",
			r:             "bytes=9.1-100",
			want:          0,
			expectedError: "strconv.Atoi: parsing \"9.1\": invalid syntax",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := RangeStartIndex(tc.r)
			if err != nil {
				if err.Error() != tc.expectedError {
					t.Errorf("expected: %v, got: %v", tc.expectedError, err.Error())
				}
			} else if got != tc.want {
				t.Errorf("expected: %v, got: %v", tc.want, got)
			}
		})
	}
}
