package circuitbreaker_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/hallode/golib/circuitbreaker"
)

func TestNonTransientError_DoesNotTripBreaker(t *testing.T) {
	cb := circuitbreaker.NewCircuitBreaker(circuitbreaker.Config{
		Name:             "test-non-transient",
		MaxRequests:      3,
		Interval:         0,
		Timeout:          0,
		FailureThreshold: 10,
	})

	for range 20 {
		_, err := circuitbreaker.Execute(cb, context.Background(), func() (struct{}, error) {
			return struct{}{}, circuitbreaker.NewNonTransientError(errors.New("validation failed"))
		})
		if err == nil {
			t.Fatal("expected non-transient error to propagate")
		}
	}

	_, err := circuitbreaker.Execute(cb, context.Background(), func() (string, error) {
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("breaker should stay closed after non-transient failures, got %v", err)
	}
}

func TestTransientError_OpensBreaker(t *testing.T) {
	cb := circuitbreaker.NewCircuitBreaker(circuitbreaker.Config{
		Name:             "test-transient",
		MaxRequests:      1,
		Interval:         0,
		Timeout:          0,
		FailureThreshold: 3,
	})

	for range 3 {
		_, _ = circuitbreaker.Execute(cb, context.Background(), func() (struct{}, error) {
			return struct{}{}, errors.New("connection refused")
		})
	}

	_, err := circuitbreaker.Execute(cb, context.Background(), func() (string, error) {
		return "ok", nil
	})
	if err == nil || !strings.Contains(err.Error(), "temporarily unavailable") {
		t.Fatalf("expected open breaker error, got %v", err)
	}
}

func TestExecute_ReturnsResult(t *testing.T) {
	cb := circuitbreaker.NewDefaultCircuitBreaker("default-test")

	got, err := circuitbreaker.Execute(cb, context.Background(), func() (int, error) {
		return 42, nil
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got != 42 {
		t.Fatalf("Execute() = %d, want 42", got)
	}
}
