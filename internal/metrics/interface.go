// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package metrics

import "github.com/prometheus/client_golang/prometheus"

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

// Global is the global metrics collector.
var Global Metrics = NewPromMetrics(prometheus.DefaultRegisterer)
