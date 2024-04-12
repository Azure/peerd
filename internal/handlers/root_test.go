package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/azure/peerd/internal/files/store"
	"github.com/azure/peerd/pkg/containerd"
	"github.com/azure/peerd/pkg/discovery/routing/tests"
	"github.com/gin-gonic/gin"
)

var simpleOKHandler = gin.HandlerFunc(func(c *gin.Context) {
	c.Status(http.StatusOK)
})

func TestV2RoutesRegistrations(t *testing.T) {
	recorder := httptest.NewRecorder()
	mc, me := gin.CreateTestContext(recorder)
	registerRoutes(me, nil, simpleOKHandler)

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{
			name:           "root",
			method:         http.MethodGet,
			path:           "/v2",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "root head",
			method:         http.MethodHead,
			path:           "/v2",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "manifests",
			method:         http.MethodGet,
			path:           "/v2/azure-cli/manifests/latest?ns=registry.k8s.io",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "manifests nested",
			method:         http.MethodGet,
			path:           "/v2/azure-cli/with/a/nested/component/manifests/latest?ns=registry.k8s.io",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "blobs",
			method:         http.MethodGet,
			path:           "/v2/azure-cli/blobs/sha256:1234?ns=registry.k8s.io",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "blobs nested",
			method:         http.MethodGet,
			path:           "/v2/azure-cli/with/a/nested/component/blobs/sha256:1234?ns=registry.k8s.io",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, tt.path, nil)
			if err != nil {
				t.Fatal(err)
			}

			me.ServeHTTP(mc.Writer, req)

			if recorder.Code != http.StatusOK {
				t.Errorf("%s: expected status code %d, got %d", tt.name, http.StatusOK, recorder.Code)
			}
		})
	}
}

func TestNewEngine(t *testing.T) {
	engine := newEngine(context.Background())
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
	ctx := context.Background()
	mr := tests.NewMockRouter(map[string][]string{})
	ms := containerd.NewMockContainerdStore(nil)
	mfs, err := store.NewMockStore(ctx, mr)
	if err != nil {
		t.Fatal(err)
	}

	h, err := Handler(ctx, mr, ms, mfs)
	if err != nil {
		t.Fatal(err)
	}

	if h == nil {
		t.Fatal("Expected non-nil handler, got nil")
	}
}
