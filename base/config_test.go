package base

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Timeout != 30*time.Second {
		t.Errorf("expected Timeout 30s, got %v", cfg.Timeout)
	}
	if cfg.RequestTimeout != 10*time.Second {
		t.Errorf("expected RequestTimeout 10s, got %v", cfg.RequestTimeout)
	}
	if cfg.MaxIdleConns != 100 {
		t.Errorf("expected MaxIdleConns 100, got %d", cfg.MaxIdleConns)
	}
	if cfg.MaxIdleConnsPerHost != 10 {
		t.Errorf("expected MaxIdleConnsPerHost 10, got %d", cfg.MaxIdleConnsPerHost)
	}
	if cfg.IdleConnTimeout != 90*time.Second {
		t.Errorf("expected IdleConnTimeout 90s, got %v", cfg.IdleConnTimeout)
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()

	if cfg.MaxRetries != 3 {
		t.Errorf("expected MaxRetries 3, got %d", cfg.MaxRetries)
	}
	if cfg.InitialWait != 100*time.Millisecond {
		t.Errorf("expected InitialWait 100ms, got %v", cfg.InitialWait)
	}
	if cfg.MaxWait != 2*time.Second {
		t.Errorf("expected MaxWait 2s, got %v", cfg.MaxWait)
	}
	if cfg.Multiplier != 2.0 {
		t.Errorf("expected Multiplier 2.0, got %f", cfg.Multiplier)
	}
	if cfg.Jitter != 0.1 {
		t.Errorf("expected Jitter 0.1, got %f", cfg.Jitter)
	}
}

func TestDefaultCircuitBreakerConfig(t *testing.T) {
	cfg := DefaultCircuitBreakerConfig("test-service")

	if cfg.FailureThreshold != 5 {
		t.Errorf("expected FailureThreshold 5, got %d", cfg.FailureThreshold)
	}
	if cfg.SuccessThreshold != 2 {
		t.Errorf("expected SuccessThreshold 2, got %d", cfg.SuccessThreshold)
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("expected Timeout 30s, got %v", cfg.Timeout)
	}
	if cfg.Name != "test-service" {
		t.Errorf("expected Name 'test-service', got %s", cfg.Name)
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &Config{
				BaseURL:        "https://api.example.com",
				Timeout:        30 * time.Second,
				RequestTimeout: 10 * time.Second,
				Retry:          DefaultRetryConfig(),
			},
			wantErr: false,
		},
		{
			name: "missing base URL",
			config: &Config{
				Timeout:        30 * time.Second,
				RequestTimeout: 10 * time.Second,
				Retry:          DefaultRetryConfig(),
			},
			wantErr: true,
			errMsg:  "base URL is required",
		},
		{
			name: "zero timeout",
			config: &Config{
				BaseURL:        "https://api.example.com",
				Timeout:        0,
				RequestTimeout: 10 * time.Second,
				Retry:          DefaultRetryConfig(),
			},
			wantErr: true,
			errMsg:  "timeout must be positive",
		},
		{
			name: "negative timeout",
			config: &Config{
				BaseURL:        "https://api.example.com",
				Timeout:        -1 * time.Second,
				RequestTimeout: 10 * time.Second,
				Retry:          DefaultRetryConfig(),
			},
			wantErr: true,
			errMsg:  "timeout must be positive",
		},
		{
			name: "zero request timeout",
			config: &Config{
				BaseURL:        "https://api.example.com",
				Timeout:        30 * time.Second,
				RequestTimeout: 0,
				Retry:          DefaultRetryConfig(),
			},
			wantErr: true,
			errMsg:  "request timeout must be positive",
		},
		{
			name: "request timeout exceeds timeout",
			config: &Config{
				BaseURL:        "https://api.example.com",
				Timeout:        10 * time.Second,
				RequestTimeout: 30 * time.Second,
				Retry:          DefaultRetryConfig(),
			},
			wantErr: true,
			errMsg:  "request timeout cannot exceed total timeout",
		},
		{
			name: "invalid retry config",
			config: &Config{
				BaseURL:        "https://api.example.com",
				Timeout:        30 * time.Second,
				RequestTimeout: 10 * time.Second,
				Retry: RetryConfig{
					MaxRetries: -1,
				},
			},
			wantErr: true,
			errMsg:  "retry config",
		},
		{
			name: "invalid circuit breaker config",
			config: &Config{
				BaseURL:        "https://api.example.com",
				Timeout:        30 * time.Second,
				RequestTimeout: 10 * time.Second,
				Retry:          DefaultRetryConfig(),
				CircuitBreaker: &CircuitBreakerConfig{
					FailureThreshold: 0,
				},
			},
			wantErr: true,
			errMsg:  "circuit breaker config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestRetryConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  RetryConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid config",
			config:  DefaultRetryConfig(),
			wantErr: false,
		},
		{
			name: "negative max retries",
			config: RetryConfig{
				MaxRetries:  -1,
				InitialWait: 100 * time.Millisecond,
				MaxWait:     2 * time.Second,
				Multiplier:  2.0,
				Jitter:      0.1,
			},
			wantErr: true,
			errMsg:  "max retries cannot be negative",
		},
		{
			name: "negative initial wait",
			config: RetryConfig{
				MaxRetries:  3,
				InitialWait: -100 * time.Millisecond,
				MaxWait:     2 * time.Second,
				Multiplier:  2.0,
				Jitter:      0.1,
			},
			wantErr: true,
			errMsg:  "initial wait cannot be negative",
		},
		{
			name: "negative max wait",
			config: RetryConfig{
				MaxRetries:  3,
				InitialWait: 100 * time.Millisecond,
				MaxWait:     -2 * time.Second,
				Multiplier:  2.0,
				Jitter:      0.1,
			},
			wantErr: true,
			errMsg:  "max wait cannot be negative",
		},
		{
			name: "initial wait exceeds max wait",
			config: RetryConfig{
				MaxRetries:  3,
				InitialWait: 5 * time.Second,
				MaxWait:     2 * time.Second,
				Multiplier:  2.0,
				Jitter:      0.1,
			},
			wantErr: true,
			errMsg:  "initial wait cannot exceed max wait",
		},
		{
			name: "multiplier less than 1",
			config: RetryConfig{
				MaxRetries:  3,
				InitialWait: 100 * time.Millisecond,
				MaxWait:     2 * time.Second,
				Multiplier:  0.5,
				Jitter:      0.1,
			},
			wantErr: true,
			errMsg:  "multiplier must be at least 1.0",
		},
		{
			name: "negative jitter",
			config: RetryConfig{
				MaxRetries:  3,
				InitialWait: 100 * time.Millisecond,
				MaxWait:     2 * time.Second,
				Multiplier:  2.0,
				Jitter:      -0.1,
			},
			wantErr: true,
			errMsg:  "jitter must be between 0 and 1",
		},
		{
			name: "jitter greater than 1",
			config: RetryConfig{
				MaxRetries:  3,
				InitialWait: 100 * time.Millisecond,
				MaxWait:     2 * time.Second,
				Multiplier:  2.0,
				Jitter:      1.5,
			},
			wantErr: true,
			errMsg:  "jitter must be between 0 and 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestCircuitBreakerConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *CircuitBreakerConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid config",
			config:  DefaultCircuitBreakerConfig("test"),
			wantErr: false,
		},
		{
			name: "zero failure threshold",
			config: &CircuitBreakerConfig{
				FailureThreshold: 0,
				SuccessThreshold: 2,
				Timeout:          30 * time.Second,
			},
			wantErr: true,
			errMsg:  "failure threshold must be positive",
		},
		{
			name: "negative failure threshold",
			config: &CircuitBreakerConfig{
				FailureThreshold: -1,
				SuccessThreshold: 2,
				Timeout:          30 * time.Second,
			},
			wantErr: true,
			errMsg:  "failure threshold must be positive",
		},
		{
			name: "zero success threshold",
			config: &CircuitBreakerConfig{
				FailureThreshold: 5,
				SuccessThreshold: 0,
				Timeout:          30 * time.Second,
			},
			wantErr: true,
			errMsg:  "success threshold must be positive",
		},
		{
			name: "zero timeout",
			config: &CircuitBreakerConfig{
				FailureThreshold: 5,
				SuccessThreshold: 2,
				Timeout:          0,
			},
			wantErr: true,
			errMsg:  "timeout must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestConfigWithDefaults(t *testing.T) {
	cfg := &Config{
		BaseURL: "https://api.example.com",
	}

	withDefaults := cfg.WithDefaults()

	if withDefaults.Timeout != 30*time.Second {
		t.Errorf("expected Timeout 30s, got %v", withDefaults.Timeout)
	}
	if withDefaults.RequestTimeout != 10*time.Second {
		t.Errorf("expected RequestTimeout 10s, got %v", withDefaults.RequestTimeout)
	}
	if withDefaults.MaxIdleConns != 100 {
		t.Errorf("expected MaxIdleConns 100, got %d", withDefaults.MaxIdleConns)
	}
	if withDefaults.MaxIdleConnsPerHost != 10 {
		t.Errorf("expected MaxIdleConnsPerHost 10, got %d", withDefaults.MaxIdleConnsPerHost)
	}
	if withDefaults.IdleConnTimeout != 90*time.Second {
		t.Errorf("expected IdleConnTimeout 90s, got %v", withDefaults.IdleConnTimeout)
	}
	if withDefaults.Retry.MaxRetries != 3 {
		t.Errorf("expected Retry.MaxRetries 3, got %d", withDefaults.Retry.MaxRetries)
	}
}

func TestRetryConfigWithDefaults(t *testing.T) {
	cfg := RetryConfig{}
	withDefaults := cfg.WithDefaults()

	if withDefaults.MaxRetries != 3 {
		t.Errorf("expected MaxRetries 3, got %d", withDefaults.MaxRetries)
	}
	if withDefaults.InitialWait != 100*time.Millisecond {
		t.Errorf("expected InitialWait 100ms, got %v", withDefaults.InitialWait)
	}
	if withDefaults.MaxWait != 2*time.Second {
		t.Errorf("expected MaxWait 2s, got %v", withDefaults.MaxWait)
	}
	if withDefaults.Multiplier != 2.0 {
		t.Errorf("expected Multiplier 2.0, got %f", withDefaults.Multiplier)
	}
	if withDefaults.Jitter != 0.1 {
		t.Errorf("expected Jitter 0.1, got %f", withDefaults.Jitter)
	}
}

func TestConfigWithDefaultsPreservesExisting(t *testing.T) {
	cfg := &Config{
		BaseURL:        "https://api.example.com",
		Timeout:        60 * time.Second,
		RequestTimeout: 20 * time.Second,
		MaxIdleConns:   50,
	}

	withDefaults := cfg.WithDefaults()

	if withDefaults.Timeout != 60*time.Second {
		t.Errorf("expected Timeout 60s (preserved), got %v", withDefaults.Timeout)
	}
	if withDefaults.RequestTimeout != 20*time.Second {
		t.Errorf("expected RequestTimeout 20s (preserved), got %v", withDefaults.RequestTimeout)
	}
	if withDefaults.MaxIdleConns != 50 {
		t.Errorf("expected MaxIdleConns 50 (preserved), got %d", withDefaults.MaxIdleConns)
	}
	if withDefaults.MaxIdleConnsPerHost != 10 {
		t.Errorf("expected MaxIdleConnsPerHost 10 (default), got %d", withDefaults.MaxIdleConnsPerHost)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || s != "" && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
