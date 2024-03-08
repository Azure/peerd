// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License, Version 2.0.
package mocks

import (
	"context"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/events"
)

type MockEventService struct {
	EnvelopeChan chan *events.Envelope
	ErrorsChan   chan error
}

// Forward implements containerd.EventService.
func (*MockEventService) Forward(ctx context.Context, envelope *events.Envelope) error {
	panic("unimplemented")
}

// Publish implements containerd.EventService.
func (*MockEventService) Publish(ctx context.Context, topic string, event events.Event) error {
	panic("unimplemented")
}

// Subscribe implements containerd.EventService.
func (m *MockEventService) Subscribe(ctx context.Context, filters ...string) (ch <-chan *events.Envelope, errs <-chan error) {
	return m.EnvelopeChan, m.ErrorsChan
}

var _ containerd.EventService = &MockEventService{}
