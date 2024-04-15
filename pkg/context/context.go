// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package context

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

const (
	// KubeConfigPath is the path of the kubeconfig file.
	KubeConfigPath = "/opt/peerd/kubeconfig"
)

// Context keys.
const (
	CorrelationIdCtxKey = "correlation_id"
	DigestCtxKey        = "digest"
	FileChunkCtxKey     = "file_chunk"
	BlobUrlCtxKey       = "blob_url"
	BlobRangeCtxKey     = "blob_range"
	NamespaceCtxKey     = "namespace"
	ReferenceCtxKey     = "reference"
	RefTypeCtxKey       = "ref_type"
	LoggerCtxKey        = "logger"
)

// Request headers.
const (
	P2PHeaderKey         = "X-MS-Peerd-RequestFromPeer"
	CorrelationHeaderKey = "X-MS-Peerd-CorrelationId"
	NodeHeaderKey        = "X-MS-Peerd-Node"
)

// Log messages.
const (
	PeerResolutionStartLog     = "peer resolution start"
	PeerResolutionStopLog      = "peer resolution stop"
	PeerNotFoundLog            = "peer not found"
	PeerResolutionExhaustedLog = "peer resolution exhausted"
	PeerRequestErrorLog        = "peer request error"
)

var (
	NodeName, _ = os.Hostname()
)

// Context is the request context that can be passed around to various components to provide request specific information.
type Context struct {
	*gin.Context
}

// FromContext creates a new context from the given gin context.
func FromContext(c *gin.Context) Context {
	return Context{Context: c}
}

// Copy creates a copy of the context that can be safely used outside the request's scope.
func (c Context) Copy() Context {
	cc := c.Context.Copy()
	return Context{Context: cc}
}

// IsRequestFromAPeer indicates if the current request is from a peer.
func IsRequestFromAPeer(c Context) bool {
	return c.Request.Header.Get(P2PHeaderKey) == "true"
}

// FillCorrelationId fills the correlation ID in the context.
func FillCorrelationId(c Context) {
	correlationId := c.Request.Header.Get(CorrelationHeaderKey)
	if correlationId == "" {
		correlationId = uuid.New().String()
	}
	c.Set(CorrelationIdCtxKey, correlationId)
}

// SetOutboundHeaders sets the mandatory headers for all outbound requests.
func SetOutboundHeaders(r *http.Request, c Context) {
	r.Header.Set(P2PHeaderKey, "true")
	r.Header.Set(CorrelationHeaderKey, c.GetString(CorrelationIdCtxKey))
	r.Header.Set(NodeHeaderKey, NodeName)
}

// Logger gets the logger with request specific fields.
func Logger(c Context) zerolog.Logger {
	var l zerolog.Logger
	obj, ok := c.Get(LoggerCtxKey)
	if !ok {
		fmt.Println("WARN: logger not found in context")
		l = zerolog.Nop()
	} else {
		ctxLog := obj.(*zerolog.Logger)
		l = *ctxLog
	}

	return l.With().Str("correlationid", c.GetString(CorrelationIdCtxKey)).Str("url", c.Request.URL.String()).Str("range", c.Request.Header.Get("Range")).Bool("requestfrompeer", IsRequestFromAPeer(c)).Str("clientip", c.ClientIP()).Str("clientname", c.Request.Header.Get(NodeHeaderKey)).Logger()
}

// BlobUrl extracts the blob URL from the incoming request URL.
func BlobUrl(c Context) string {
	return strings.TrimPrefix(c.Param("url"), "/") + "?" + c.Request.URL.RawQuery
}

// RangeStartIndex returns the start index of a byte range specified in the given range header value.
// It expects the range value to be in the format "bytes=startIndex-endIndex".
func RangeStartIndex(rangeValue string) (int64, error) {
	if rangeValue == "" {
		return 0, errors.New("no range header")
	}

	// split the range value by "="
	parts := strings.Split(rangeValue, "=")
	if len(parts) != 2 || parts[0] != "bytes" {
		return 0, errors.New("invalid range format")
	}

	// split the byte range by "-"
	ranges := strings.Split(parts[1], "-")
	if len(ranges) != 2 {
		return 0, errors.New("invalid range format")
	}

	// convert the start index to an integer
	startIndex, err := strconv.Atoi(ranges[0])
	if err != nil {
		return 0, err
	}

	return int64(startIndex), nil
}
