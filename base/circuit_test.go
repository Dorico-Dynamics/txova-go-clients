package base

import (
	"testing"
	"time"
)

func TestCircuitStateString(t *testing.T) {
	tests := []struct {
		state    CircuitState
		expected string
	}{
		{CircuitClosed, "closed"},
		{CircuitOpen, "open"},
		{CircuitHalfOpen, "half-open"},
		{CircuitState(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.state.String(); got != tt.expected {
				t.Errorf("CircuitState.String() = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestNewCircuitBreaker(t *testing.T) {
	cfg := &CircuitBreakerConfig{
		FailureThreshold: 5,
		SuccessThreshold: 2,
		Timeout:          30 * time.Second,
		Name:             "test-service",
	}

	cb := NewCircuitBreaker(cfg)

	if cb.State() != CircuitClosed {
		t.Errorf("expected initial state Closed, got %s", cb.State())
	}
	if cb.Name() != "test-service" {
		t.Errorf("expected name 'test-service', got %s", cb.Name())
	}
}

func TestCircuitBreakerAllowInClosedState(t *testing.T) {
	cb := NewCircuitBreaker(&CircuitBreakerConfig{
		FailureThreshold: 5,
		SuccessThreshold: 2,
		Timeout:          30 * time.Second,
	})

	for i := 0; i < 10; i++ {
		if !cb.Allow() {
			t.Errorf("expected Allow() to return true in Closed state")
		}
	}
}

func TestCircuitBreakerOpensAfterFailures(t *testing.T) {
	cb := NewCircuitBreaker(&CircuitBreakerConfig{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          30 * time.Second,
	})

	cb.RecordFailure()
	if cb.State() != CircuitClosed {
		t.Errorf("expected state Closed after 1 failure, got %s", cb.State())
	}

	cb.RecordFailure()
	if cb.State() != CircuitClosed {
		t.Errorf("expected state Closed after 2 failures, got %s", cb.State())
	}

	cb.RecordFailure()
	if cb.State() != CircuitOpen {
		t.Errorf("expected state Open after 3 failures, got %s", cb.State())
	}

	if cb.Allow() {
		t.Error("expected Allow() to return false in Open state")
	}
}

func TestCircuitBreakerResetsFailuresOnSuccess(t *testing.T) {
	cb := NewCircuitBreaker(&CircuitBreakerConfig{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          30 * time.Second,
	})

	cb.RecordFailure()
	cb.RecordFailure()

	cb.RecordSuccess()

	cb.RecordFailure()
	cb.RecordFailure()

	if cb.State() != CircuitClosed {
		t.Errorf("expected state Closed (failures reset), got %s", cb.State())
	}
}

func TestCircuitBreakerTransitionToHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker(&CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          50 * time.Millisecond,
	})

	cb.RecordFailure()
	cb.RecordFailure()

	if cb.State() != CircuitOpen {
		t.Errorf("expected state Open, got %s", cb.State())
	}

	time.Sleep(60 * time.Millisecond)

	if !cb.Allow() {
		t.Error("expected Allow() to return true after timeout")
	}

	if cb.State() != CircuitHalfOpen {
		t.Errorf("expected state HalfOpen after timeout, got %s", cb.State())
	}
}

func TestCircuitBreakerClosesFromHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker(&CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          50 * time.Millisecond,
	})

	cb.RecordFailure()
	cb.RecordFailure()

	time.Sleep(60 * time.Millisecond)
	cb.Allow()

	cb.RecordSuccess()
	if cb.State() != CircuitHalfOpen {
		t.Errorf("expected state HalfOpen after 1 success, got %s", cb.State())
	}

	cb.RecordSuccess()
	if cb.State() != CircuitClosed {
		t.Errorf("expected state Closed after 2 successes, got %s", cb.State())
	}
}

func TestCircuitBreakerReopensFromHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker(&CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          50 * time.Millisecond,
	})

	cb.RecordFailure()
	cb.RecordFailure()

	time.Sleep(60 * time.Millisecond)
	cb.Allow()

	cb.RecordSuccess()

	cb.RecordFailure()

	if cb.State() != CircuitOpen {
		t.Errorf("expected state Open after failure in HalfOpen, got %s", cb.State())
	}
}

func TestCircuitBreakerReset(t *testing.T) {
	cb := NewCircuitBreaker(&CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          30 * time.Second,
	})

	cb.RecordFailure()
	cb.RecordFailure()

	if cb.State() != CircuitOpen {
		t.Errorf("expected state Open, got %s", cb.State())
	}

	cb.Reset()

	if cb.State() != CircuitClosed {
		t.Errorf("expected state Closed after reset, got %s", cb.State())
	}

	if !cb.Allow() {
		t.Error("expected Allow() to return true after reset")
	}
}

func TestCircuitBreakerStats(t *testing.T) {
	cb := NewCircuitBreaker(&CircuitBreakerConfig{
		FailureThreshold: 5,
		SuccessThreshold: 2,
		Timeout:          30 * time.Second,
	})

	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()

	stats := cb.Stats()

	if stats.State != CircuitClosed {
		t.Errorf("expected stats.State Closed, got %s", stats.State)
	}
	if stats.ConsecutiveFailures != 3 {
		t.Errorf("expected ConsecutiveFailures 3, got %d", stats.ConsecutiveFailures)
	}
	if stats.LastFailureTime.IsZero() {
		t.Error("expected LastFailureTime to be set")
	}
}

func TestCircuitBreakerConcurrency(t *testing.T) {
	cb := NewCircuitBreaker(&CircuitBreakerConfig{
		FailureThreshold: 100,
		SuccessThreshold: 50,
		Timeout:          30 * time.Second,
	})

	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				cb.Allow()
				cb.RecordSuccess()
				cb.RecordFailure()
				cb.State()
				cb.Stats()
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestCircuitBreakerAllowInHalfOpen(t *testing.T) {
	// Default maxConcurrentProbes is 1, so only one probe at a time.
	cb := NewCircuitBreaker(&CircuitBreakerConfig{
		FailureThreshold: 1,
		SuccessThreshold: 2,
		Timeout:          50 * time.Millisecond,
	})

	cb.RecordFailure()

	time.Sleep(60 * time.Millisecond)

	// First Allow() should return true and transition to half-open.
	if !cb.Allow() {
		t.Error("expected first Allow() in HalfOpen to return true")
	}

	// Second Allow() should return false (probe limit reached).
	if cb.Allow() {
		t.Error("expected subsequent Allow() in HalfOpen to return false (probe limit reached)")
	}

	// After recording success, probe counter decrements and another probe is allowed.
	cb.RecordSuccess()

	if !cb.Allow() {
		t.Error("expected Allow() to return true after RecordSuccess() decrements probe counter")
	}
}

func TestCircuitBreakerMaxConcurrentProbes(t *testing.T) {
	// Configure multiple concurrent probes.
	cb := NewCircuitBreaker(&CircuitBreakerConfig{
		FailureThreshold:    1,
		SuccessThreshold:    2,
		Timeout:             50 * time.Millisecond,
		MaxConcurrentProbes: 3,
	})

	cb.RecordFailure()

	time.Sleep(60 * time.Millisecond)

	// Should allow up to 3 concurrent probes.
	if !cb.Allow() {
		t.Error("expected Allow() #1 to return true")
	}
	if !cb.Allow() {
		t.Error("expected Allow() #2 to return true")
	}
	if !cb.Allow() {
		t.Error("expected Allow() #3 to return true")
	}

	// Fourth probe should be rejected.
	if cb.Allow() {
		t.Error("expected Allow() #4 to return false (max probes reached)")
	}

	// Verify stats.
	stats := cb.Stats()
	if stats.InFlightProbes != 3 {
		t.Errorf("expected InFlightProbes=3, got %d", stats.InFlightProbes)
	}
	if stats.MaxConcurrentProbes != 3 {
		t.Errorf("expected MaxConcurrentProbes=3, got %d", stats.MaxConcurrentProbes)
	}

	// Record a success to decrement probe counter.
	cb.RecordSuccess()

	// Now another probe should be allowed.
	if !cb.Allow() {
		t.Error("expected Allow() to return true after probe completed")
	}
}

func TestCircuitBreakerProbeCounterOnFailure(t *testing.T) {
	cb := NewCircuitBreaker(&CircuitBreakerConfig{
		FailureThreshold:    1,
		SuccessThreshold:    2,
		Timeout:             50 * time.Millisecond,
		MaxConcurrentProbes: 2,
	})

	cb.RecordFailure()

	time.Sleep(60 * time.Millisecond)

	// Allow first probe.
	if !cb.Allow() {
		t.Error("expected Allow() to return true")
	}

	// Record failure - should reopen circuit and reset probe counter.
	cb.RecordFailure()

	if cb.State() != CircuitOpen {
		t.Errorf("expected state Open after probe failure, got %s", cb.State())
	}

	stats := cb.Stats()
	if stats.InFlightProbes != 0 {
		t.Errorf("expected InFlightProbes=0 after reopening, got %d", stats.InFlightProbes)
	}
}

func TestCircuitBreakerRecordSuccessInOpenState(t *testing.T) {
	cb := NewCircuitBreaker(&CircuitBreakerConfig{
		FailureThreshold: 1,
		SuccessThreshold: 2,
		Timeout:          30 * time.Second,
	})

	cb.RecordFailure()

	if cb.State() != CircuitOpen {
		t.Fatalf("expected state Open, got %s", cb.State())
	}

	cb.RecordSuccess()

	if cb.State() != CircuitHalfOpen {
		t.Errorf("expected state HalfOpen after success in Open, got %s", cb.State())
	}
}
