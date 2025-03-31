package mocks

import (
	"context"
	"testing"
	"time"
)

func TestNewMockRouter(t *testing.T) {
	r := NewMockRouter(map[string][]string{})
	if r == nil {
		t.Errorf("expected non-nil router")
	}
}

func TestNet(t *testing.T) {
	r := NewMockRouter(map[string][]string{})
	if r.Net() == nil {
		t.Errorf("expected non-nil net")
	}
}

func TestResolveWithNegativeCacheCallback(t *testing.T) {
	r := NewMockRouter(map[string][]string{"key1": {"value1"}})
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	valueChan, callback, err := r.ResolveWithNegativeCacheCallback(ctx, "key2", false, 1) // count and allowSelf are not used.
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	select {
	case value := <-valueChan:
		t.Errorf("expected no value, got %s", value)
	case <-ctx.Done():
		// This is expected as the context should timeout. Invoke callback to populate negative cache.
		callback()
	}
}

func TestProvider(t *testing.T) {
	r := NewMockRouter(map[string][]string{})
	keys := []string{"key1", "key2"}
	err := r.Provide(context.Background(), keys)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestResolve(t *testing.T) {
	r := NewMockRouter(map[string][]string{"key1": {"value1"}})

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	valueChan, err := r.Resolve(ctx, "key1", false, 1) // count and allowSelf are not used.
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	select {
	case value := <-valueChan:
		if value.HttpHost != "value1" {
			t.Errorf("expected value1, got %s", value)
		}
	case <-ctx.Done():
		t.Errorf("expected value to be received")
	}

	valueChan, err = r.Resolve(ctx, "key2", false, 1)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	select {
	case value := <-valueChan:
		t.Errorf("expected no value, got %s", value.HttpHost)
	case <-ctx.Done():
		// This is expected as the context should timeout.
	}
}

func TestLookupKey(t *testing.T) {
	r := NewMockRouter(map[string][]string{"key1": {"value1"}})
	value, ok := r.LookupKey("key1")
	if !ok {
		t.Errorf("expected key to be found")
	}
	if len(value) != 1 {
		t.Errorf("expected one value, got %d", len(value))
	}
	if value[0] != "value1" {
		t.Errorf("expected value1, got %s", value[0])
	}
}

func TestClose(t *testing.T) {
	r := NewMockRouter(map[string][]string{})
	err := r.Close()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
