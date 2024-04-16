// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package metrics

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestPromMetrics_RecordPeerDiscovery(t *testing.T) {
	reg := prometheus.NewPedanticRegistry()
	m := NewPromMetrics(reg, "test", "peerd")

	ip := "192.168.0.1"
	duration := 0.001
	m.RecordPeerDiscovery(ip, duration)

	// Verify that the prometheus metric was updated correctly
	expected := `
		# HELP peerd_peer_discovery_duration_seconds Duration of peer discovery
		# TYPE peerd_peer_discovery_duration_seconds histogram
		peerd_peer_discovery_duration_seconds_sum 0.001
		peerd_peer_discovery_duration_seconds_count 1
	`

	if err := testutil.GatherAndCompare(reg, strings.NewReader(expected), "peerd_peer_discovery_duration_seconds_sum", "peerd_peer_discovery_duration_seconds_count"); err != nil {
		t.Errorf("unexpected metric result:\n%s", err)
	}

	ip = "10.0.01"
	duration = 1.0
	m.RecordPeerDiscovery(ip, duration)

	// Verify that the prometheus metric was updated correctly
	expected = `
		# HELP peerd_peer_discovery_duration_seconds Duration of peer discovery
		# TYPE peerd_peer_discovery_duration_seconds histogram
		peerd_peer_discovery_duration_seconds_sum 1.001
		peerd_peer_discovery_duration_seconds_count 2
	`

	if err := testutil.GatherAndCompare(reg, strings.NewReader(expected), "peerd_peer_discovery_duration_seconds_sum", "peerd_peer_discovery_duration_seconds_count"); err != nil {
		t.Errorf("unexpected metric result:\n%s", err)
	}
}

func TestPromMetrics_RecordPeerResponse(t *testing.T) {
	reg := prometheus.NewPedanticRegistry()
	m := NewPromMetrics(reg, "test", "peerd")

	ip := "192.168.0.1"
	key := "key"
	op := "operation"
	duration := 1.0
	count := int64(1024 * 1024)
	m.RecordPeerResponse(ip, key, op, duration, count)

	// Verify that the prometheus metric was updated correctly
	expected := `
		# HELP peerd_peer_response_speed Speed of peer response
		# TYPE peerd_peer_response_speed histogram
		peerd_peer_response_speed_sum 1
		peerd_peer_response_speed_count 1
	`

	if err := testutil.GatherAndCompare(reg, strings.NewReader(expected), "peerd_peer_response_speed_sum", "peerd_peer_response_speed_count"); err != nil {
		t.Errorf("unexpected metric result:\n%s", err)
	}

	ip = ""
	key = "key"
	op = "operation"
	duration = 2.0
	count = int64(4 * 1024 * 1024)
	m.RecordPeerResponse(ip, key, op, duration, count)

	// Verify that the prometheus metric was updated correctly
	expected = `
		# HELP peerd_peer_response_speed Speed of peer response
		# TYPE peerd_peer_response_speed histogram
		peerd_peer_response_speed_sum 3
		peerd_peer_response_speed_count 2
	`

	if err := testutil.GatherAndCompare(reg, strings.NewReader(expected), "peerd_peer_response_speed_sum", "peerd_peer_response_speed_count"); err != nil {
		t.Errorf("unexpected metric result:\n%s", err)
	}
}

func TestPromMetrics_RecordRequest(t *testing.T) {
	reg := prometheus.NewPedanticRegistry()
	m := NewPromMetrics(reg, "test", "peerd")

	method := "GET"
	handler := "files"
	duration := 0.5
	m.RecordRequest(method, handler, duration)

	// Verify that the prometheus metric was updated correctly
	expected := `
		# HELP peerd_request_duration_seconds Duration of request
		# TYPE peerd_request_duration_seconds histogram
		peerd_request_duration_seconds_sum 0.5
		peerd_request_duration_seconds_count 1
	`

	if err := testutil.GatherAndCompare(reg, strings.NewReader(expected), "peerd_request_duration_seconds_count", "peerd_request_duration_seconds_sum"); err != nil {
		t.Errorf("unexpected metric result:\n%s", err)
	}

	method = "HEAD"
	handler = "files"
	duration = 0.1
	m.RecordRequest(method, handler, duration)

	// Verify that the prometheus metric was updated correctly
	expected = `
		# HELP peerd_request_duration_seconds Duration of request
		# TYPE peerd_request_duration_seconds histogram
		peerd_request_duration_seconds_sum 0.6
		peerd_request_duration_seconds_count 2
	`

	if err := testutil.GatherAndCompare(reg, strings.NewReader(expected), "peerd_request_duration_seconds_count", "peerd_request_duration_seconds_sum"); err != nil {
		t.Errorf("unexpected metric result:\n%s", err)
	}
}

func TestPromMetrics_RecordUpstreamResponse(t *testing.T) {
	reg := prometheus.NewPedanticRegistry()
	m := NewPromMetrics(reg, "test", "peerd")

	hostname := "localhost"
	key := "key"
	op := "operation"
	duration := 1.0
	count := int64(1024 * 1024)

	m.RecordUpstreamResponse(hostname, key, op, duration, count)

	// Verify that the prometheus metric was updated correctly
	expected := `
		# HELP peerd_upstream_response_speed Speed of upstream response
		# TYPE peerd_upstream_response_speed histogram
		peerd_upstream_response_speed_sum 1
		peerd_upstream_response_speed_count 1
	`

	if err := testutil.GatherAndCompare(reg, strings.NewReader(expected), "peerd_upstream_response_speed_sum", "peerd_upstream_response_speed_count"); err != nil {
		t.Errorf("unexpected metric result:\n%s", err)
	}

	hostname = "localhost"
	key = "key"
	op = "operation"
	duration = 2.0
	count = int64(4 * 1024 * 1024)

	m.RecordUpstreamResponse(hostname, key, op, duration, count)

	// Verify that the prometheus metric was updated correctly
	expected = `
		# HELP peerd_upstream_response_speed Speed of upstream response
		# TYPE peerd_upstream_response_speed histogram
		peerd_upstream_response_speed_sum 3
		peerd_upstream_response_speed_count 2
	`

	if err := testutil.GatherAndCompare(reg, strings.NewReader(expected), "peerd_upstream_response_speed_sum", "peerd_upstream_response_speed_count"); err != nil {
		t.Errorf("unexpected metric result:\n%s", err)
	}
}
