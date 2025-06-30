// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package main

import (
	"context"
	"strings"
	"testing"

	"github.com/azure/peerd/pkg/metrics"
	"github.com/rs/zerolog"
)

var (
	testCtx context.Context
)

func init() {
	var err error
	testCtx, err = metrics.WithContext(context.Background(), "test", "peerd")
	if err != nil {
		panic(err)
	}
}

func TestRun(t *testing.T) {
	// Use the shared test context
	ctx := testCtx

	testCases := []struct {
		name        string
		args        *Arguments
		expectError bool
		errorMsg    string
	}{
		{
			name: "version flag",
			args: &Arguments{
				Version:  true,
				LogLevel: "info",
			},
			expectError: false,
		},
		{
			name: "server command with valid args",
			args: &Arguments{
				Server: &ServerCmd{
					HttpAddr:        "127.0.0.1:5000",
					HttpsAddr:       "127.0.0.1:5000",
					RouterAddr:      "127.0.0.1:5000",
					PromAddr:        "127.0.0.1:5000",
					PrefetchWorkers: 10,
				},
				LogLevel: "info",
			},
			expectError: true, // Will fail due to k8s dependencies in real implementation
		},
		{
			name: "no subcommand",
			args: &Arguments{
				LogLevel: "info",
			},
			expectError: true,
			errorMsg:    "unknown subcommand",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := run(ctx, tc.args)

			if tc.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tc.errorMsg != "" && !strings.Contains(err.Error(), tc.errorMsg) {
					t.Errorf("Expected error to contain %q, got %q", tc.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestServerCommand_InvalidHttpsAddr(t *testing.T) {
	ctx := testCtx

	args := &ServerCmd{
		HttpAddr:        "127.0.0.1:8080",
		HttpsAddr:       "invalid-address",
		RouterAddr:      "127.0.0.1:8082",
		PromAddr:        "127.0.0.1:8083",
		PrefetchWorkers: 10,
	}

	err := serverCommand(ctx, args)
	if err == nil {
		t.Error("Expected error for invalid https address")
	}

	if !strings.Contains(err.Error(), "missing port") {
		t.Errorf("Expected 'missing port' error, got %v", err)
	}
}

func TestLoggingConfiguration(t *testing.T) {
	testCases := []struct {
		name     string
		logLevel string
		valid    bool
	}{
		{"debug level", "debug", true},
		{"info level", "info", true},
		{"warn level", "warn", true},
		{"error level", "error", true},
		{"fatal level", "fatal", true},
		{"panic level", "panic", true},
		{"invalid level", "invalid", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := zerolog.ParseLevel(tc.logLevel)

			if tc.valid && err != nil {
				t.Errorf("Expected valid log level %s, got error: %v", tc.logLevel, err)
			}

			if !tc.valid && err == nil {
				t.Errorf("Expected invalid log level %s to return error", tc.logLevel)
			}
		})
	}
}
