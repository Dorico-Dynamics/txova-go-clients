package base

import (
	"sync"
	"time"
)

// CircuitState represents the state of the circuit breaker.
type CircuitState int

const (
	// CircuitClosed is the normal state where requests are allowed.
	CircuitClosed CircuitState = iota
	// CircuitOpen is the state where requests are blocked.
	CircuitOpen
	// CircuitHalfOpen is the state where probe requests are allowed.
	CircuitHalfOpen
)

// String returns the string representation of the circuit state.
func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreaker implements the circuit breaker pattern.
// It prevents cascading failures by stopping requests to unhealthy services.
type CircuitBreaker struct {
	config CircuitBreakerConfig

	mu                   sync.RWMutex
	state                CircuitState
	consecutiveFailures  int
	consecutiveSuccesses int
	lastFailureTime      time.Time
}

// NewCircuitBreaker creates a new CircuitBreaker with the given configuration.
func NewCircuitBreaker(config *CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		config: *config,
		state:  CircuitClosed,
	}
}

// Allow checks if a request is allowed to proceed.
// Returns true if the request should proceed, false if it should be blocked.
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true

	case CircuitOpen:
		// Check if timeout has elapsed.
		if time.Since(cb.lastFailureTime) >= cb.config.Timeout {
			// Transition to half-open.
			cb.state = CircuitHalfOpen
			cb.consecutiveSuccesses = 0
			return true
		}
		return false

	case CircuitHalfOpen:
		// Allow probe requests in half-open state.
		return true

	default:
		return true
	}
}

// RecordSuccess records a successful request.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		// Reset failure count on success.
		cb.consecutiveFailures = 0

	case CircuitHalfOpen:
		cb.consecutiveSuccesses++
		// If enough successes, close the circuit.
		if cb.consecutiveSuccesses >= cb.config.SuccessThreshold {
			cb.state = CircuitClosed
			cb.consecutiveFailures = 0
			cb.consecutiveSuccesses = 0
		}

	case CircuitOpen:
		// Should not happen, but handle gracefully.
		cb.state = CircuitHalfOpen
		cb.consecutiveSuccesses = 1
	}
}

// RecordFailure records a failed request.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.lastFailureTime = time.Now()

	switch cb.state {
	case CircuitClosed:
		cb.consecutiveFailures++
		// If too many failures, open the circuit.
		if cb.consecutiveFailures >= cb.config.FailureThreshold {
			cb.state = CircuitOpen
		}

	case CircuitHalfOpen:
		// Failure in half-open state reopens the circuit.
		cb.state = CircuitOpen
		cb.consecutiveSuccesses = 0

	case CircuitOpen:
		// Already open, just update timestamp.
	}
}

// State returns the current state of the circuit breaker.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Name returns the name of the circuit breaker.
func (cb *CircuitBreaker) Name() string {
	return cb.config.Name
}

// Reset resets the circuit breaker to its initial state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = CircuitClosed
	cb.consecutiveFailures = 0
	cb.consecutiveSuccesses = 0
	cb.lastFailureTime = time.Time{}
}

// Stats returns statistics about the circuit breaker.
type CircuitBreakerStats struct {
	State                CircuitState
	ConsecutiveFailures  int
	ConsecutiveSuccesses int
	LastFailureTime      time.Time
}

// Stats returns the current statistics for the circuit breaker.
func (cb *CircuitBreaker) Stats() CircuitBreakerStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return CircuitBreakerStats{
		State:                cb.state,
		ConsecutiveFailures:  cb.consecutiveFailures,
		ConsecutiveSuccesses: cb.consecutiveSuccesses,
		LastFailureTime:      cb.lastFailureTime,
	}
}
