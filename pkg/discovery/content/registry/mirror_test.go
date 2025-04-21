// Copyright (c) Microsoft Corporation.
// Portions (c) 2023 Xenit AB and 2024 The Spegel Authors.
// Licensed under the MIT License.
package registry

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	pcontext "github.com/azure/peerd/pkg/context"
	"github.com/azure/peerd/pkg/discovery/routing/mocks"
	"github.com/azure/peerd/pkg/metrics"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

type TestResponseRecorder struct {
	*httptest.ResponseRecorder
	closeChannel chan bool
}

func (r *TestResponseRecorder) CloseNotify() <-chan bool {
	return r.closeChannel
}

//nolint:unused // ignore
func (r *TestResponseRecorder) closeClient() {
	r.closeChannel <- true
}

func CreateTestResponseRecorder() *TestResponseRecorder {
	return &TestResponseRecorder{
		httptest.NewRecorder(),
		make(chan bool, 1),
	}
}

func TestMirrorHandler(t *testing.T) {
	badSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("foo", "bar")
		if r.Method == http.MethodGet {
			//nolint:errcheck // ignore
			w.Write([]byte("hello world"))
		}
	}))
	defer badSvr.Close()

	goodSvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("foo", "bar")
		if r.Method == http.MethodGet {
			//nolint:errcheck // ignore
			w.Write([]byte("hello world"))
		}
	}))
	defer goodSvr.Close()

	resolver := map[string][]string{
		"no-working-peers":  {badSvr.URL, "foo", badSvr.URL},
		"first-peer":        {goodSvr.URL, badSvr.URL, badSvr.URL},
		"first-peer-error":  {"foo", goodSvr.URL},
		"last-peer-working": {badSvr.URL, badSvr.URL, goodSvr.URL},
	}
	router := mocks.NewMockRouter(resolver)
	m := &Mirror{
		metricsRecorder: metrics.NewPromMetrics(prometheus.DefaultRegisterer, "test", "test"),
		router:          router,
		resolveRetries:  ResolveRetries,
		resolveTimeout:  ResolveTimeout,
		n:               router.Net(),
	}

	tests := []struct {
		name            string
		key             string
		expectedStatus  int
		expectedBody    string
		expectedHeaders map[string][]string
	}{
		{
			name:            "request should timeout when no peers exists",
			key:             "no-peers",
			expectedStatus:  http.StatusNotFound,
			expectedBody:    "",
			expectedHeaders: nil,
		},
		{
			name:            "request should not timeout and give 500 if all peers fail",
			key:             "no-working-peers",
			expectedStatus:  http.StatusInternalServerError,
			expectedBody:    "",
			expectedHeaders: nil,
		},
		{
			name:            "request should work when first peer responds",
			key:             "first-peer",
			expectedStatus:  http.StatusOK,
			expectedBody:    "hello world",
			expectedHeaders: map[string][]string{"foo": {"bar"}},
		},
		{
			name:            "second peer should respond when first gives error",
			key:             "first-peer-error",
			expectedStatus:  http.StatusOK,
			expectedBody:    "hello world",
			expectedHeaders: map[string][]string{"foo": {"bar"}},
		},
		{
			name:            "last peer should respond when two first fail",
			key:             "last-peer-working",
			expectedStatus:  http.StatusOK,
			expectedBody:    "hello world",
			expectedHeaders: map[string][]string{"foo": {"bar"}},
		},
	}
	for _, tt := range tests {
		for _, method := range []string{http.MethodGet, http.MethodHead} {
			t.Run(tt.name, func(t *testing.T) {
				rw := CreateTestResponseRecorder()
				c, _ := gin.CreateTestContext(rw)
				target := fmt.Sprintf("http://example.com/%s", tt.key)
				c.Request = httptest.NewRequest(method, target, nil)
				c.Set(pcontext.DigestCtxKey, tt.key)
				m.Handle(pcontext.FromContext(c))

				resp := rw.Result()
				defer func() {
					require.NoError(t, resp.Body.Close())
				}()
				b, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				require.Equal(t, tt.expectedStatus, resp.StatusCode)

				if method == http.MethodGet {
					require.Equal(t, tt.expectedBody, string(b))
				}
				if method == http.MethodHead {
					require.Equal(t, "", string(b))
				}

				if tt.expectedHeaders == nil {
					require.Len(t, resp.Header, 0)
				}
				for k, v := range tt.expectedHeaders {
					require.Equal(t, v, resp.Header.Values(k))
				}
			})
		}
	}
}
