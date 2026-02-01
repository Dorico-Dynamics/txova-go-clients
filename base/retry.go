package base

import (
	"context"
	"math"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"
)

// Retryer handles retry logic with exponential backoff.
type Retryer struct {
	config RetryConfig
}

// NewRetryer creates a new Retryer with the given configuration.
func NewRetryer(config RetryConfig) *Retryer {
	return &Retryer{
		config: config.WithDefaults(),
	}
}

// ShouldRetry determines if a request should be retried based on the response.
func (r *Retryer) ShouldRetry(resp *http.Response, err error, attempt int) bool {
	// Don't retry if max attempts reached.
	if attempt >= r.config.MaxRetries {
		return false
	}

	// Retry on network errors.
	if err != nil {
		return true
	}

	// Retry on retryable status codes.
	if resp != nil && IsRetryableStatus(resp.StatusCode) {
		return true
	}

	return false
}

// WaitDuration calculates the wait duration before the next retry attempt.
// It uses exponential backoff with jitter and respects the Retry-After header.
func (r *Retryer) WaitDuration(resp *http.Response, attempt int) time.Duration {
	// Check for Retry-After header.
	if resp != nil {
		if retryAfter := r.parseRetryAfter(resp); retryAfter > 0 {
			// Cap the Retry-After value at max wait.
			if retryAfter > r.config.MaxWait {
				return r.config.MaxWait
			}
			return retryAfter
		}
	}

	// Calculate exponential backoff.
	wait := float64(r.config.InitialWait) * math.Pow(r.config.Multiplier, float64(attempt))

	// Add jitter (cryptographic randomness not required for backoff jitter).
	jitterRange := wait * r.config.Jitter
	jitter := (rand.Float64() * 2 * jitterRange) - jitterRange // #nosec G404 -- jitter does not require crypto rand
	wait += jitter

	// Cap at max wait.
	if wait > float64(r.config.MaxWait) {
		wait = float64(r.config.MaxWait)
	}

	// Ensure non-negative.
	if wait < 0 {
		wait = 0
	}

	return time.Duration(wait)
}

// Wait waits for the calculated duration, respecting context cancellation.
// Returns an error if the context is cancelled during the wait.
func (r *Retryer) Wait(ctx context.Context, resp *http.Response, attempt int) error {
	duration := r.WaitDuration(resp, attempt)
	if duration == 0 {
		return nil
	}

	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// parseRetryAfter parses the Retry-After header value.
// It supports both seconds and HTTP-date formats.
func (r *Retryer) parseRetryAfter(resp *http.Response) time.Duration {
	if resp == nil {
		return 0
	}

	retryAfter := resp.Header.Get("Retry-After")
	if retryAfter == "" {
		return 0
	}

	// Try parsing as seconds.
	if seconds, err := strconv.ParseInt(retryAfter, 10, 64); err == nil {
		if seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
		return 0
	}

	// Try parsing as HTTP-date.
	if t, err := http.ParseTime(retryAfter); err == nil {
		duration := time.Until(t)
		if duration > 0 {
			return duration
		}
	}

	return 0
}

// MaxRetries returns the maximum number of retries.
func (r *Retryer) MaxRetries() int {
	return r.config.MaxRetries
}
