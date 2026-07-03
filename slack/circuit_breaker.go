package slack

import (
	"sync"
	"time"
)

type circuitBreaker struct {
	mu              sync.RWMutex
	failureCount    int
	lastFailureTime time.Time
	successCount    int
	threshold       int
	timeout         time.Duration
	openUntil       time.Time
	open            bool
}

func newCircuitBreaker(threshold int, timeout time.Duration) *circuitBreaker {
	return &circuitBreaker{
		threshold: threshold,
		timeout:   timeout,
	}
}

func (cb *circuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailureTime = time.Now()

	if cb.failureCount >= cb.threshold {
		cb.openUntil = time.Now().Add(cb.timeout)
		cb.open = true
	}
}

func (cb *circuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.successCount++
	if cb.successCount > 0 {
		cb.failureCount = maxInt(0, cb.failureCount-1)
	}

	if cb.failureCount == 0 {
		cb.open = false
	}
}

func (cb *circuitBreaker) isOpen() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if !cb.open {
		return false
	}

	if time.Now().After(cb.openUntil) {
		cb.open = false
		cb.failureCount = 0
		cb.successCount = 0
		return false
	}

	return true
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
