// Package metrics provides a metrics collector that stores metrics in Prometheus.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// promMetrics is a metrics collector that stores metrics in Prometheus.
type promMetrics struct {
	requestDuration          *prometheus.HistogramVec
	peerDiscoveryDuration    *prometheus.HistogramVec
	peerResponseDuration     *prometheus.HistogramVec
	upstreamResponseDuration *prometheus.HistogramVec
}

var _ Metrics = &promMetrics{}

// RecordPeerDiscovery records the duration of peer discovery for a given IP address.
func (m *promMetrics) RecordPeerDiscovery(ip string, duration float64) {
	m.peerDiscoveryDuration.WithLabelValues(ip).Observe(duration)
}

// RecordPeerResponse records the response time and count of a peer's operation.
// It calculates the speed (count/duration) and updates the Prometheus metric.
func (m *promMetrics) RecordPeerResponse(ip string, key string, op string, duration float64, count int64) {
	speed := float64(count) / duration
	m.peerResponseDuration.WithLabelValues(ip, op).Observe(speed)
}

// RecordRequest records the duration of a request for a specific method and handler.
// It updates the Prometheus metric for request duration.
func (m *promMetrics) RecordRequest(method string, handler string, duration float64) {
	m.requestDuration.WithLabelValues(method, handler).Observe(duration)
}

// RecordUpstreamResponse records the duration and count of an upstream response.
// It calculates the speed of the response and updates the corresponding Prometheus metric.
func (m *promMetrics) RecordUpstreamResponse(hostname string, key string, op string, duration float64, count int64) {
	speed := float64(count) / duration
	m.upstreamResponseDuration.WithLabelValues(hostname, op).Observe(speed)
}

// NewPromMetrics creates a new instance of promMetrics.
func NewPromMetrics() *promMetrics {

	requestDurationHist := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "peerd_request_duration_seconds",
		Help: "Duration of requests in seconds.",
	}, []string{"method", "handler"})
	prometheus.MustRegister(requestDurationHist)

	peerDiscoveryDurationHist := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "peerd_peer_discovery_duration_seconds",
		Help: "Duration of peer discovery in seconds.",
	}, []string{"ip"})
	prometheus.MustRegister(peerDiscoveryDurationHist)

	peerResponseDurationHist := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "peerd_peer_response_speed_bytes_per_second",
		Help: "Speed of peer response in bytes per second.",
	}, []string{"ip", "op"})
	prometheus.MustRegister(peerResponseDurationHist)

	upstreamResponseDurationHist := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "peerd_upstream_response_speed_bytes_per_seconds",
		Help: "Speed of upstream response in bytes per second.",
	}, []string{"hostname", "op"})
	prometheus.MustRegister(upstreamResponseDurationHist)

	return &promMetrics{
		requestDuration:          requestDurationHist,
		peerDiscoveryDuration:    peerDiscoveryDurationHist,
		peerResponseDuration:     peerResponseDurationHist,
		upstreamResponseDuration: upstreamResponseDurationHist,
	}
}
