// Package circuitbreaker wraps sony/gobreaker with a generic Execute[T] and
// business-error exclusion: wrap validation/business errors in
// NewNonTransientError so they do not trip the breaker. When the breaker is open
// Execute returns ErrCircuitOpen. Optional state-change logging via
// Config.EnableLogging requires golib/log to be initialized first.
package circuitbreaker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hallode/golib/v2/log"

	"github.com/sony/gobreaker"
)

// ErrCircuitOpen is returned by Execute when the breaker is open (or half-open
// and rejecting requests) and the guarded function was not called. Match it with
// errors.Is(err, ErrCircuitOpen).
var ErrCircuitOpen = errors.New("service temporarily unavailable: circuit breaker is open")

// NonTransientError marks business/validation errors that must not trip the circuit breaker.
type NonTransientError struct {
	Err error
}

func (e *NonTransientError) Error() string { return e.Err.Error() }
func (e *NonTransientError) Unwrap() error { return e.Err }

func NewNonTransientError(err error) error {
	return &NonTransientError{Err: err}
}

type Config struct {
	Name string
	// MaxRequests is the number of probe requests allowed while half-open; the
	// breaker closes once they all succeed. It governs half-open recovery.
	MaxRequests      uint32
	Interval         time.Duration
	Timeout          time.Duration
	FailureThreshold uint32

	// EnableLogging emits state-change and open-breaker messages via golib/log.
	// Default is false; requires log.New when enabled.
	EnableLogging bool
}

type CircuitBreaker struct {
	cb            *gobreaker.CircuitBreaker
	enableLogging bool
}

func NewCircuitBreaker(config Config) *CircuitBreaker {
	settings := gobreaker.Settings{
		Name:        config.Name,
		MaxRequests: config.MaxRequests,
		Interval:    config.Interval,
		Timeout:     config.Timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= config.FailureThreshold
		},
		IsSuccessful: func(err error) bool {
			if err == nil {
				return true
			}
			var nonTransient *NonTransientError
			return errors.As(err, &nonTransient)
		},
	}

	if config.EnableLogging {
		settings.OnStateChange = func(name string, from gobreaker.State, to gobreaker.State) {
			log.WithParams(log.Params{"circuit_breaker": name, "from": from.String(), "to": to.String()}).Info("Circuit breaker state changed")
		}
	}

	return &CircuitBreaker{
		cb:            gobreaker.NewCircuitBreaker(settings),
		enableLogging: config.EnableLogging,
	}
}

func NewDefaultCircuitBreaker(name string) *CircuitBreaker {
	return NewCircuitBreaker(Config{
		Name:             name,
		MaxRequests:      3,
		Interval:         60 * time.Second,
		Timeout:          10 * time.Second,
		FailureThreshold: 5,
	})
}

func Execute[T any](cb *CircuitBreaker, ctx context.Context, fn func() (T, error)) (T, error) {
	var zero T

	result, err := cb.cb.Execute(func() (any, error) {
		return fn()
	})

	if err != nil {
		if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
			if cb.enableLogging {
				log.With(ctx).WithParam("error", err).Warnf("Circuit breaker %s is open", cb.cb.Name())
			}
			return zero, fmt.Errorf("circuitbreaker %q: %w", cb.cb.Name(), ErrCircuitOpen)
		}
		return zero, err
	}

	if result == nil {
		return zero, nil
	}

	val, ok := result.(T)
	if !ok {
		return zero, errors.New("invalid response type from circuit breaker")
	}

	return val, nil
}
