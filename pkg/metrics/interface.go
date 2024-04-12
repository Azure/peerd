// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package metrics

import (
	"context"
	"errors"

	"github.com/prometheus/client_golang/prometheus"
)

// Metrics defines an interface to collect p2p metrics.
type Metrics interface {
	// RecordRequest records the time it takes to process a request.
	RecordRequest(method, handler string, duration float64)

	// RecordPeerDiscovery records the time it takes to discover a peer.
	RecordPeerDiscovery(ip string, duration float64)

	// RecordPeerResponse records the time it takes for a peer to respond for a key.
	RecordPeerResponse(ip, key, op string, duration float64, count int64)

	// RecordUpstreamResponse records the time it takes for an upstream to respond for a key.
	RecordUpstreamResponse(hostname, key, op string, duration float64, count int64)
}

// WithContext returns a new context with an metrics recorder.
func WithContext(ctx context.Context, name, prefix string) (context.Context, error) {
	pm := NewPromMetrics(prometheus.DefaultRegisterer, name, prefix)
	if pm == nil {
		return nil, errors.New("failed to create prometheus metrics")
	}

	return context.WithValue(ctx, ctxKey{}, pm), nil
}

// FromContext returns the metrics recorder from the context.
func FromContext(ctx context.Context) Metrics {
	return ctx.Value(ctxKey{}).(*promMetrics)
}
