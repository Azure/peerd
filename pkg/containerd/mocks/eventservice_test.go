package mocks

import (
	"context"
	"testing"

	"github.com/containerd/containerd/events"
)

func TestSubscribe(t *testing.T) {
	m := &MockEventService{
		EnvelopeChan: make(chan *events.Envelope),
		ErrorsChan:   make(chan error),
	}
	envelopeChan, errChan := m.Subscribe(context.Background(), "test")
	if envelopeChan == nil {
		t.Errorf("Expected non-nil envelopeChan, got nil")
	}
	if errChan == nil {
		t.Errorf("Expected non-nil errChan, got nil")
	}
}

func TestEventStorePanics(t *testing.T) {
	tests := []struct {
		name string
		fn   func()
	}{
		{
			name: "TestForward",
			fn: func() {
				m := &MockEventService{}
				err := m.Forward(context.Background(), nil)
				if err != nil {
					t.Errorf("Forward() error = %v", err)
				}
			},
		},
		{
			name: "TestPublish",
			fn: func() {
				m := &MockEventService{}
				err := m.Publish(context.Background(), "", nil)
				if err != nil {
					t.Errorf("Publish() error = %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("Expected panic, but did not panic")
				}
			}()
			tt.fn()
		})
	}
}
