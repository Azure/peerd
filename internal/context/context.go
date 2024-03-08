// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License, Version 2.0.
package context

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// P2P network.
const (
	KeyTTL = 30 * time.Minute
)

// Cache constants.
const (
	P2pLookupCacheTtl      = 500 * time.Millisecond
	P2pLookupNotFoundValue = "PEER_NOT_FOUND"
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
	P2PHeaderKey         = "X-MS-Cluster-P2P-RequestFromPeer"
	CorrelationHeaderKey = "X-MS-Cluster-P2P-CorrelationId"
	NodeHeaderKey        = "X-MS-Cluster-P2P-Node"
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
	Namespace   = "peerd-ns"

	// KubeConfigPath is the path of the kubeconfig file, which is used if run in an environment outside a pod.
	KubeConfigPath = "/opt/peerd/kubeconfig"
)

// IsRequestFromAPeer indicates if the current request is from a peer.
func IsRequestFromAPeer(c *gin.Context) bool {
	return c.Request.Header.Get(P2PHeaderKey) == "true"
}

func FillCorrelationId(c *gin.Context) {
	correlationId := c.Request.Header.Get(CorrelationHeaderKey)
	if correlationId == "" {
		correlationId = uuid.New().String()
	}
	c.Set(CorrelationIdCtxKey, correlationId)
}

// Logger gets the logger with request specific fields.
func Logger(c *gin.Context) zerolog.Logger {
	var l zerolog.Logger
	obj, ok := c.Get(LoggerCtxKey)
	if !ok {
		fmt.Println("WARN: logger not found in context")
		l = zerolog.Nop()
	} else {
		ctxLog := obj.(*zerolog.Logger)
		l = *ctxLog
	}

	return l.With().Str("correlationid", c.GetString(CorrelationIdCtxKey)).Str("url", c.Request.URL.String()).Str("range", c.Request.Header.Get("Range")).Bool("p2p", IsRequestFromAPeer(c)).Str("ip", c.ClientIP()).Str("peer", c.Request.Header.Get(NodeHeaderKey)).Logger()
}

// BlobUrl extracts the blob URL from the incoming request URL.
func BlobUrl(c *gin.Context) string {
	return strings.TrimPrefix(c.Param("url"), "/") + "?" + c.Request.URL.RawQuery
}

// SetOutboundHeaders sets the mandatory headers for all outbound requests.
func SetOutboundHeaders(r *http.Request, c *gin.Context) {
	r.Header.Set(P2PHeaderKey, "true")
	r.Header.Set(CorrelationHeaderKey, c.GetString(CorrelationIdCtxKey))
	r.Header.Set(NodeHeaderKey, NodeName)
}

// Merge merges multiple input channels into a single output channel.
// It starts a goroutine for each input channel and sends the values from each input channel to the output channel.
// Once all input channels are closed, it closes the output channel.
// The function returns the output channel.
func Merge[T any](cs ...<-chan T) <-chan T {
	var wg sync.WaitGroup
	out := make(chan T)

	output := func(c <-chan T) {
		for n := range c {
			out <- n
		}
		wg.Done()
	}
	wg.Add(len(cs))
	for _, c := range cs {
		go output(c)
	}

	go func() {
		wg.Wait()
		close(out)
	}()
	return out
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
