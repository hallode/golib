package otel

import (
	"context"
	"testing"
)

// preserveCtxKey is a distinct type for context.Value tests (not bool/string keys).
type preserveCtxKey struct{}

func TestTracerPreservesParentContextValues(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), preserveCtxKey{}, "keep-me")
	next, span := Tracer(ctx)
	defer span.End()

	if got, ok := next.Value(preserveCtxKey{}).(string); !ok || got != "keep-me" {
		t.Fatalf("Tracer lost parent context.Value: ok=%v got=%v", ok, got)
	}
}
