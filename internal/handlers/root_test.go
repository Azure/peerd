package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

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
