// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package context

import (
	"net/http"
	"net/http/httptest"
	"os"
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

	pc := FromContext(c)

	l := Logger(pc)
	if l.Info().Enabled() {
		t.Fatal("expected logger to be disabled")
	}

	testL := zerolog.New(os.Stdout).With().Timestamp().Logger()
	c.Set(LoggerCtxKey, &testL)

	l = Logger(pc)
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

	pc := FromContext(ctx)

	FillCorrelationId(pc)

	SetOutboundHeaders(req, pc)

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

	pc := FromContext(ctx)

	// Call BlobUrl and verify the result.
	got := BlobUrl(pc)
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

	pc := FromContext(ctx)

	FillCorrelationId(pc)
	cid, ok := pc.Get(CorrelationIdCtxKey)
	if !ok || cid == "" {
		t.Fatal("expected correlation ID to be set")
	}

	sample := uuid.New().String()

	ctx, _ = gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req
	ctx.Request.Header.Set(CorrelationHeaderKey, sample)

	pc = FromContext(ctx)

	FillCorrelationId(pc)
	cid, ok = pc.Get(CorrelationIdCtxKey)
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

	pc := FromContext(ctx)

	if IsRequestFromAPeer(pc) {
		t.Fatal("expected request to not be from a peer")
	}

	ctx.Request.Header.Set(P2PHeaderKey, "true")
	if !IsRequestFromAPeer(pc) {
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
