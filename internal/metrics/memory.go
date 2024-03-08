// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License, Version 2.0.
package metrics

import (
	"os"
	"syscall"
	"time"

	hmetrics "github.com/hashicorp/go-metrics"
)

var (
	// Path is the default path to write metrics.
	Path = "/var/log/p2pmetrics"

	// ReportInterval is the interval to report metrics.
	ReportInterval = 3 * time.Minute

	// AggregationInterval is the interval to aggregate metrics.
	AggregationInterval = 2 * time.Minute

	// RetentionPeriod is the retention period of metrics.
	RetentionPeriod = 10 * time.Minute
)

// memoryMetrics is a metrics collector that stores metrics in memory.
type memoryMetrics struct {
	sink *hmetrics.InmemSink

	reportingInterval time.Duration
	reportFilePath    string
}

// RecordPeerDiscovery records the time it takes to discover a peer.
func (m *memoryMetrics) RecordPeerDiscovery(ip string, duration float64) {
	m.recordLatency(duration, ip, "discovery")
}

// RecordPeerResponse records the time it takes for a peer to respond for a key.
func (m *memoryMetrics) RecordPeerResponse(ip, key, op string, duration float64, count int64) {
	m.recordLatency(duration, ip, op)
	m.recordBytes(count, ip, op)

	if duration > 0 {
		m.recordSpeed(float64(count)/duration, ip, op)
	}
}

// RecordRequest records the time it takes to process a request.
func (m *memoryMetrics) RecordRequest(method string, handler string, duration float64) {
	m.recordLatency(duration, "server", method+"_"+handler)
}

// RecordUpstreamResponse records the time it takes for an upstream to respond for a key.
func (m *memoryMetrics) RecordUpstreamResponse(hostname, key, op string, duration float64, count int64) {
	m.recordLatency(duration, hostname, op)
	m.recordBytes(count, hostname, op)

	if duration > 0 {
		m.recordSpeed(float64(count)/duration, hostname, op)
	}
}

// recordLatency records the time it takes to perform an operation.
func (m *memoryMetrics) recordLatency(duration float64, host, op string) {
	m.sink.AddSample([]string{"latency", host, op}, float32(duration))
}

// recordSpeed records the speed of a download from a host.
func (m *memoryMetrics) recordSpeed(speed float64, host, op string) {
	m.sink.AddSample([]string{"speed", host, op}, float32(speed))
}

// recordBytes records the number of bytes downloaded from a host.
func (m *memoryMetrics) recordBytes(bytes int64, host, op string) {
	m.sink.AddSample([]string{"bytes", host, op}, float32(bytes))
}

var _ Metrics = &memoryMetrics{}

// reportPeriodically reports the current metrics to a file every 5 minutes.
func (m *memoryMetrics) reportPeriodically() {
	go func() {
		ticker := time.NewTicker(m.reportingInterval)
		defer ticker.Stop()
		for range ticker.C {
			f, err := os.OpenFile(m.reportFilePath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
			if err == nil {
				hmetrics.NewInmemSignal(m.sink, hmetrics.DefaultSignal, f)

				_ = syscall.Kill(os.Getpid(), syscall.SIGUSR1)

				// Wait for flush.
				time.Sleep(20 * time.Millisecond)

				_ = f.Sync()
				f.Close()
			}
		}
	}()
}

// NewMemoryMetrics returns a new memory metrics collector.
func NewMemoryMetrics() Metrics {
	sink := hmetrics.NewInmemSink(AggregationInterval, RetentionPeriod)

	c := hmetrics.DefaultConfig("peerd")
	c.EnableRuntimeMetrics = false

	_, err := hmetrics.NewGlobal(c, sink)
	if err != nil {
		panic(err)
	}

	m := &memoryMetrics{sink, ReportInterval, Path}
	m.reportPeriodically()

	return m
}
