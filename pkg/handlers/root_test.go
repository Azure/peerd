// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/azure/peerd/pkg/discovery/routing/mocks"
	"github.com/azure/peerd/pkg/files/store"
	"github.com/azure/peerd/pkg/metrics"
	"github.com/gin-gonic/gin"
)

var (
	ctxWithMetrics, _ = metrics.WithContext(context.Background(), "test", "peerd")
)

func TestNewEngine(t *testing.T) {
	engine := newEngine(ctxWithMetrics)
	if engine == nil {
		t.Fatal("Expected non-nil engine, got nil")
	}

	if engine.Handlers == nil {
		t.Fatal("Expected non-nil handlers, got nil")
	}

	if len(engine.Handlers) != 2 {
		t.Errorf("Expected 2 middleware, got %d", len(engine.Handlers))
	}
}

func TestHandler(t *testing.T) {
	mr := mocks.NewMockRouter(map[string][]string{})
	mfs, err := store.NewMockStore(ctxWithMetrics, mr, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	h, err := Handler(ctxWithMetrics, mr, mfs)
	if err != nil {
		t.Fatal(err)
	}

	if h == nil {
		t.Fatal("Expected non-nil handler, got nil")
	}
}

func TestBlobRoutesRegistered(t *testing.T) {
	mr := mocks.NewMockRouter(map[string][]string{})
	mfs, err := store.NewMockStore(ctxWithMetrics, mr, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	handler, err := Handler(ctxWithMetrics, mr, mfs)
	if err != nil {
		t.Fatal(err)
	}

	// Test cases for blob routes
	testCases := []struct {
		name   string
		method string
		path   string
	}{
		{
			name:   "GET blob route",
			method: "GET",
			path:   "/blobs/test-url",
		},
		{
			name:   "HEAD blob route",
			method: "HEAD",
			path:   "/blobs/test-url",
		},
		{
			name:   "GET blob route with nested path",
			method: "GET",
			path:   "/blobs/https://example.com/path/to/blob",
		},
		{
			name:   "HEAD blob route with nested path",
			method: "HEAD",
			path:   "/blobs/https://example.com/path/to/blob",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(tc.method, tc.path, nil)
			if err != nil {
				t.Fatal(err)
			}

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			// The route should be registered and handled (not return 404)
			// Since we're testing route registration, we don't expect 404
			if recorder.Code == http.StatusNotFound {
				t.Errorf("Route %s %s not registered - got 404", tc.method, tc.path)
			}
		})
	}
}

func TestRegisterRoutes(t *testing.T) {
	engine := newEngine(ctxWithMetrics)

	// Use a test handler that sets a specific response
	testHandler := gin.HandlerFunc(func(c *gin.Context) {
		c.String(http.StatusOK, "test-handler-called")
	})

	registerRoutes(engine, testHandler)

	testCases := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "GET blob route calls handler",
			method:         "GET",
			path:           "/blobs/test-blob-url",
			expectedStatus: http.StatusOK,
			expectedBody:   "test-handler-called",
		},
		{
			name:           "HEAD blob route calls handler",
			method:         "HEAD",
			path:           "/blobs/test-blob-url",
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(tc.method, tc.path, nil)
			if err != nil {
				t.Fatal(err)
			}

			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, req)

			if recorder.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, recorder.Code)
			}

			if tc.method == "GET" && recorder.Body.String() != tc.expectedBody {
				t.Errorf("Expected body %q, got %q", tc.expectedBody, recorder.Body.String())
			}
		})
	}
}
