package base

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"
)

// Config holds the configuration for the HTTP client.
type Config struct {
	// BaseURL is the base URL for all requests (required).
	BaseURL string

	// Timeout is the total request timeout including retries (default: 30s).
	Timeout time.Duration

	// RequestTimeout is the timeout for a single request attempt (default: 10s).
	RequestTimeout time.Duration

	// MaxIdleConns is the maximum number of idle connections (default: 100).
	MaxIdleConns int

	// MaxIdleConnsPerHost is the maximum idle connections per host (default: 10).
	MaxIdleConnsPerHost int

	// IdleConnTimeout is how long idle connections stay open (default: 90s).
	IdleConnTimeout time.Duration

	// TLSConfig is the TLS configuration for HTTPS connections.
	TLSConfig *tls.Config

	// Transport is a custom transport for testing or advanced configuration.
	// If set, connection pooling options are ignored.
	Transport http.RoundTripper

	// Retry configuration.
	Retry RetryConfig

	// CircuitBreaker configuration. If nil, circuit breaker is disabled.
	CircuitBreaker *CircuitBreakerConfig
}

// RetryConfig holds retry configuration.
type RetryConfig struct {
	// MaxRetries is the maximum number of retry attempts (default: 3).
	MaxRetries int

	// InitialWait is the initial wait time between retries (default: 100ms).
	InitialWait time.Duration

	// MaxWait is the maximum wait time between retries (default: 2s).
	MaxWait time.Duration

	// Multiplier is the backoff multiplier (default: 2.0).
	Multiplier float64

	// Jitter is the random jitter factor to add to backoff (default: 0.1).
	Jitter float64
}

// CircuitBreakerConfig holds circuit breaker configuration.
type CircuitBreakerConfig struct {
	// FailureThreshold is the number of failures before opening the circuit (default: 5).
	FailureThreshold int

	// SuccessThreshold is the number of successes to close the circuit (default: 2).
	SuccessThreshold int

	// Timeout is how long the circuit stays open before allowing a probe (default: 30s).
	Timeout time.Duration

	// MaxConcurrentProbes is the max number of concurrent probe requests in half-open state (default: 1).
	MaxConcurrentProbes int

	// Name is an identifier for this circuit breaker (used in logging/metrics).
	Name string
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Timeout:             30 * time.Second,
		RequestTimeout:      10 * time.Second,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		Retry:               DefaultRetryConfig(),
	}
}

// DefaultRetryConfig returns a RetryConfig with sensible defaults.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:  3,
		InitialWait: 100 * time.Millisecond,
		MaxWait:     2 * time.Second,
		Multiplier:  2.0,
		Jitter:      0.1,
	}
}

// DefaultCircuitBreakerConfig returns a CircuitBreakerConfig with sensible defaults.
func DefaultCircuitBreakerConfig(name string) *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		FailureThreshold: 5,
		SuccessThreshold: 2,
		Timeout:          30 * time.Second,
		Name:             name,
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.BaseURL == "" {
		return fmt.Errorf("base URL is required")
	}

	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	if c.RequestTimeout <= 0 {
		return fmt.Errorf("request timeout must be positive")
	}

	if c.RequestTimeout > c.Timeout {
		return fmt.Errorf("request timeout cannot exceed total timeout")
	}

	if err := c.Retry.Validate(); err != nil {
		return fmt.Errorf("retry config: %w", err)
	}

	if c.CircuitBreaker != nil {
		if err := c.CircuitBreaker.Validate(); err != nil {
			return fmt.Errorf("circuit breaker config: %w", err)
		}
	}

	return nil
}

// Validate validates the retry configuration.
func (c *RetryConfig) Validate() error {
	if c.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative")
	}

	if c.InitialWait < 0 {
		return fmt.Errorf("initial wait cannot be negative")
	}

	if c.MaxWait < 0 {
		return fmt.Errorf("max wait cannot be negative")
	}

	if c.MaxWait > 0 && c.InitialWait > c.MaxWait {
		return fmt.Errorf("initial wait cannot exceed max wait")
	}

	if c.Multiplier < 1.0 {
		return fmt.Errorf("multiplier must be at least 1.0")
	}

	if c.Jitter < 0 || c.Jitter > 1.0 {
		return fmt.Errorf("jitter must be between 0 and 1")
	}

	return nil
}

// Validate validates the circuit breaker configuration.
func (c *CircuitBreakerConfig) Validate() error {
	if c.FailureThreshold <= 0 {
		return fmt.Errorf("failure threshold must be positive")
	}

	if c.SuccessThreshold <= 0 {
		return fmt.Errorf("success threshold must be positive")
	}

	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	return nil
}

// WithDefaults returns a new Config with defaults applied for any zero values.
func (c *Config) WithDefaults() *Config {
	cfg := *c

	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	if cfg.RequestTimeout == 0 {
		cfg.RequestTimeout = 10 * time.Second
	}

	if cfg.MaxIdleConns == 0 {
		cfg.MaxIdleConns = 100
	}

	if cfg.MaxIdleConnsPerHost == 0 {
		cfg.MaxIdleConnsPerHost = 10
	}

	if cfg.IdleConnTimeout == 0 {
		cfg.IdleConnTimeout = 90 * time.Second
	}

	cfg.Retry = cfg.Retry.WithDefaults()

	return &cfg
}

// WithDefaults returns a new RetryConfig with defaults applied for any zero values.
func (c RetryConfig) WithDefaults() RetryConfig {
	cfg := c

	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}

	if cfg.InitialWait == 0 {
		cfg.InitialWait = 100 * time.Millisecond
	}

	if cfg.MaxWait == 0 {
		cfg.MaxWait = 2 * time.Second
	}

	if cfg.Multiplier == 0 {
		cfg.Multiplier = 2.0
	}

	if cfg.Jitter == 0 {
		cfg.Jitter = 0.1
	}

	return cfg
}
