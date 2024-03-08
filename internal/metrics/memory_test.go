// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License, Version 2.0.
package metrics

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestMetricsWritten(t *testing.T) {
	m := NewMemoryMetrics()

	m.RecordPeerDiscovery("10.0.0.1", 1.0)
	m.RecordPeerDiscovery("10.0.0.3", 1.2)
	m.RecordPeerDiscovery("10.0.0.2", 1.0)

	m.RecordPeerResponse("10.0.0.1", "key", "pread", 1.0, 15)
	m.RecordPeerResponse("10.0.0.3", "key-a", "pread", 1.2, 10)
	m.RecordPeerResponse("10.0.0.2", "key-b", "pread", 1.0, 1)

	m.RecordRequest("GET", "key", 1.0)

	m.RecordUpstreamResponse("upstream-a", "key-a", "pread", 1.2, 10)

	time.Sleep(ReportInterval + 300*time.Millisecond)

	contents, err := os.ReadFile(Path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if len(contents) == 0 {
		t.Fatalf("file is empty")
	}

	s := string(contents)

	if !strings.Contains(s, "speed") {
		t.Fatalf("file does not contain speed metric")
	}

	if !strings.Contains(s, "bytes") {
		t.Fatalf("file does not contain bytes metric")
	}

	if !strings.Contains(s, "latency") {
		t.Fatalf("file does not contain latency metric")
	}
}
