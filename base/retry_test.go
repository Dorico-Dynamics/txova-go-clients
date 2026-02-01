package base

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestNewRetryer(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries:  5,
		InitialWait: 200 * time.Millisecond,
		MaxWait:     5 * time.Second,
		Multiplier:  3.0,
		Jitter:      0.2,
	}

	retryer := NewRetryer(cfg)

	if retryer.MaxRetries() != 5 {
		t.Errorf("expected MaxRetries 5, got %d", retryer.MaxRetries())
	}
}

func TestRetryerShouldRetry(t *testing.T) {
	retryer := NewRetryer(RetryConfig{
		MaxRetries:  3,
		InitialWait: 100 * time.Millisecond,
		MaxWait:     2 * time.Second,
		Multiplier:  2.0,
		Jitter:      0,
	})

	tests := []struct {
		name     string
		resp     *http.Response
		err      error
		attempt  int
		expected bool
	}{
		{
			name:     "retry on network error",
			resp:     nil,
			err:      context.DeadlineExceeded,
			attempt:  0,
			expected: true,
		},
		{
			name:     "retry on 500 status",
			resp:     &http.Response{StatusCode: http.StatusInternalServerError},
			err:      nil,
			attempt:  0,
			expected: true,
		},
		{
			name:     "retry on 502 status",
			resp:     &http.Response{StatusCode: http.StatusBadGateway},
			err:      nil,
			attempt:  0,
			expected: true,
		},
		{
			name:     "retry on 503 status",
			resp:     &http.Response{StatusCode: http.StatusServiceUnavailable},
			err:      nil,
			attempt:  0,
			expected: true,
		},
		{
			name:     "retry on 504 status",
			resp:     &http.Response{StatusCode: http.StatusGatewayTimeout},
			err:      nil,
			attempt:  0,
			expected: true,
		},
		{
			name:     "retry on 429 status",
			resp:     &http.Response{StatusCode: http.StatusTooManyRequests},
			err:      nil,
			attempt:  0,
			expected: true,
		},
		{
			name:     "retry on 408 status",
			resp:     &http.Response{StatusCode: http.StatusRequestTimeout},
			err:      nil,
			attempt:  0,
			expected: true,
		},
		{
			name:     "no retry on 200 status",
			resp:     &http.Response{StatusCode: http.StatusOK},
			err:      nil,
			attempt:  0,
			expected: false,
		},
		{
			name:     "no retry on 400 status",
			resp:     &http.Response{StatusCode: http.StatusBadRequest},
			err:      nil,
			attempt:  0,
			expected: false,
		},
		{
			name:     "no retry on 401 status",
			resp:     &http.Response{StatusCode: http.StatusUnauthorized},
			err:      nil,
			attempt:  0,
			expected: false,
		},
		{
			name:     "no retry on 404 status",
			resp:     &http.Response{StatusCode: http.StatusNotFound},
			err:      nil,
			attempt:  0,
			expected: false,
		},
		{
			name:     "no retry when max attempts reached",
			resp:     nil,
			err:      context.DeadlineExceeded,
			attempt:  3,
			expected: false,
		},
		{
			name:     "no retry when attempts exceeded",
			resp:     &http.Response{StatusCode: http.StatusInternalServerError},
			err:      nil,
			attempt:  4,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := retryer.ShouldRetry(tt.resp, tt.err, tt.attempt)
			if got != tt.expected {
				t.Errorf("ShouldRetry() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRetryerWaitDuration(t *testing.T) {
	// Create config without using WithDefaults to have exact control
	cfg := RetryConfig{
		MaxRetries:  3,
		InitialWait: 100 * time.Millisecond,
		MaxWait:     2 * time.Second,
		Multiplier:  2.0,
		Jitter:      0,
	}
	retryer := &Retryer{config: cfg}

	tests := []struct {
		name             string
		attempt          int
		expectedDuration time.Duration
	}{
		{
			name:             "first attempt",
			attempt:          0,
			expectedDuration: 100 * time.Millisecond,
		},
		{
			name:             "second attempt",
			attempt:          1,
			expectedDuration: 200 * time.Millisecond,
		},
		{
			name:             "third attempt",
			attempt:          2,
			expectedDuration: 400 * time.Millisecond,
		},
		{
			name:             "fourth attempt",
			attempt:          3,
			expectedDuration: 800 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := retryer.WaitDuration(nil, tt.attempt)
			if got != tt.expectedDuration {
				t.Errorf("WaitDuration() = %v, want %v", got, tt.expectedDuration)
			}
		})
	}
}

func TestRetryerWaitDurationCapped(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries:  10,
		InitialWait: 100 * time.Millisecond,
		MaxWait:     500 * time.Millisecond,
		Multiplier:  2.0,
		Jitter:      0,
	}
	retryer := &Retryer{config: cfg}

	duration := retryer.WaitDuration(nil, 5)
	if duration != 500*time.Millisecond {
		t.Errorf("expected duration to be capped at 500ms, got %v", duration)
	}
}

func TestRetryerWaitDurationWithJitter(t *testing.T) {
	retryer := NewRetryer(RetryConfig{
		MaxRetries:  3,
		InitialWait: 100 * time.Millisecond,
		MaxWait:     2 * time.Second,
		Multiplier:  2.0,
		Jitter:      0.5,
	})

	baseDuration := 100 * time.Millisecond
	minDuration := time.Duration(float64(baseDuration) * 0.5)
	maxDuration := time.Duration(float64(baseDuration) * 1.5)

	for i := 0; i < 100; i++ {
		got := retryer.WaitDuration(nil, 0)
		if got < minDuration || got > maxDuration {
			t.Errorf("WaitDuration() = %v, want between %v and %v", got, minDuration, maxDuration)
		}
	}
}

func TestRetryerWaitDurationWithRetryAfterHeader(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries:  3,
		InitialWait: 100 * time.Millisecond,
		MaxWait:     2 * time.Second,
		Multiplier:  2.0,
		Jitter:      0,
	}
	retryer := &Retryer{config: cfg}

	t.Run("respects Retry-After header in seconds", func(t *testing.T) {
		resp := &http.Response{
			Header: http.Header{
				"Retry-After": []string{"1"},
			},
		}

		got := retryer.WaitDuration(resp, 0)
		if got != 1*time.Second {
			t.Errorf("WaitDuration() = %v, want 1s", got)
		}
	})

	t.Run("caps Retry-After at max wait", func(t *testing.T) {
		resp := &http.Response{
			Header: http.Header{
				"Retry-After": []string{"60"},
			},
		}

		got := retryer.WaitDuration(resp, 0)
		if got != 2*time.Second {
			t.Errorf("WaitDuration() = %v, want 2s (capped)", got)
		}
	})

	t.Run("ignores invalid Retry-After header", func(t *testing.T) {
		resp := &http.Response{
			Header: http.Header{
				"Retry-After": []string{"invalid"},
			},
		}

		got := retryer.WaitDuration(resp, 0)
		if got != 100*time.Millisecond {
			t.Errorf("WaitDuration() = %v, want 100ms (default)", got)
		}
	})
}

func TestRetryerWait(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries:  3,
		InitialWait: 50 * time.Millisecond,
		MaxWait:     2 * time.Second,
		Multiplier:  2.0,
		Jitter:      0,
	}
	retryer := &Retryer{config: cfg}

	t.Run("waits for calculated duration", func(t *testing.T) {
		ctx := context.Background()
		start := time.Now()

		err := retryer.Wait(ctx, nil, 0)

		elapsed := time.Since(start)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if elapsed < 45*time.Millisecond {
			t.Errorf("wait was too short: %v", elapsed)
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			time.Sleep(10 * time.Millisecond)
			cancel()
		}()

		err := retryer.Wait(ctx, nil, 0)
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	})

	t.Run("respects context deadline", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := retryer.Wait(ctx, nil, 0)
		if err != context.DeadlineExceeded {
			t.Errorf("expected context.DeadlineExceeded, got %v", err)
		}
	})
}

func TestRetryerWithZeroInitialWait(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries:  3,
		InitialWait: 0,
		MaxWait:     0,
		Multiplier:  1.0,
		Jitter:      0,
	}
	retryer := &Retryer{config: cfg}

	ctx := context.Background()
	start := time.Now()

	err := retryer.Wait(ctx, nil, 0)

	elapsed := time.Since(start)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if elapsed > 50*time.Millisecond {
		t.Errorf("wait should be near-instant for zero duration, got %v", elapsed)
	}
}
