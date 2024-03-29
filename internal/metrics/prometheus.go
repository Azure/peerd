// Package metrics provides a metrics collector that stores metrics in Prometheus.
package metrics

import (
	"github.com/azure/peerd/internal/context"
	"github.com/prometheus/client_golang/prometheus"
)

// promMetrics is a metrics collector that stores metrics in Prometheus.
type promMetrics struct {
	requestDuration       *prometheus.HistogramVec
	peerDiscoveryDuration *prometheus.HistogramVec
	peerResponseSpeed     *prometheus.HistogramVec
	upstreamResponseSpeed *prometheus.HistogramVec
}

var _ Metrics = &promMetrics{}

// RecordPeerDiscovery records the duration of peer discovery for a given IP address.
func (m *promMetrics) RecordPeerDiscovery(ip string, duration float64) {
	m.peerDiscoveryDuration.WithLabelValues(context.NodeName, ip).Observe(duration)
}

// RecordPeerResponse records the response time and count of a peer's operation.
// It calculates the speed (count/duration) and updates the Prometheus metric.
func (m *promMetrics) RecordPeerResponse(ip string, key string, op string, duration float64, count int64) {
	bps := float64(count) / duration
	m.peerResponseSpeed.WithLabelValues(context.NodeName, ip, op).Observe(bps / float64(1024*1024))
}

// RecordRequest records the duration of a request for a specific method and handler.
// It updates the Prometheus metric for request duration.
func (m *promMetrics) RecordRequest(method string, handler string, duration float64) {
	m.requestDuration.WithLabelValues(context.NodeName, method, handler).Observe(duration)
}

// RecordUpstreamResponse records the duration and count of an upstream response.
// It calculates the speed of the response and updates the corresponding Prometheus metric.
func (m *promMetrics) RecordUpstreamResponse(hostname string, key string, op string, duration float64, count int64) {
	bps := float64(count) / duration
	m.upstreamResponseSpeed.WithLabelValues(context.NodeName, hostname, op).Observe(bps / float64(1024*1024))
}

// NewPromMetrics creates a new instance of promMetrics.
func NewPromMetrics(reg prometheus.Registerer) *promMetrics {

	requestDurationHist := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "peerd_request_duration_seconds",
		Help:    "Duration of requests in seconds.",
		Buckets: prometheus.LinearBuckets(0.005, 0.025, 200),
	}, []string{"self", "method", "handler"})
	reg.MustRegister(requestDurationHist)

	peerDiscoveryDurationHist := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "peerd_peer_discovery_duration_seconds",
		Help:    "Duration of peer discovery in seconds.",
		Buckets: prometheus.LinearBuckets(0.001, 0.002, 200),
	}, []string{"self", "ip"})
	reg.MustRegister(peerDiscoveryDurationHist)

	peerResponseDurationHist := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "peerd_peer_response_speed_mib_per_second",
		Help:    "Speed of peer response in Mib per second.",
		Buckets: prometheus.LinearBuckets(1, 15, 200),
	}, []string{"self", "ip", "op"})
	reg.MustRegister(peerResponseDurationHist)

	upstreamResponseDurationHist := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "peerd_upstream_response_speed_mib_per_seconds",
		Help:    "Speed of upstream response in Mib per second.",
		Buckets: prometheus.LinearBuckets(1, 15, 200),
	}, []string{"self", "hostname", "op"})
	reg.MustRegister(upstreamResponseDurationHist)

	return &promMetrics{
		requestDuration:       requestDurationHist,
		peerDiscoveryDuration: peerDiscoveryDurationHist,
		peerResponseSpeed:     peerResponseDurationHist,
		upstreamResponseSpeed: upstreamResponseDurationHist,
	}
}
