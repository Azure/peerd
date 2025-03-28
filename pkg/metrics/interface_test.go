package metrics

import (
	"context"
	"testing"
)

func TestWithContext(t *testing.T) {
	ctx := context.Background()
	name := "test"
	prefix := "test_prefix"

	ctx, err := WithContext(ctx, name, prefix)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}

	m := ctx.Value(ctxKey{}).(*promMetrics)
	if m == nil {
		t.Fatal("expected non-nil metrics")
	}
	if m.name != name {
		t.Errorf("expected name %s, got %s", name, m.name)
	}
}

func TestFromContext(t *testing.T) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, ctxKey{}, &promMetrics{})

	m := FromContext(ctx)
	if m == nil {
		t.Fatal("expected non-nil metrics")
	}
}
