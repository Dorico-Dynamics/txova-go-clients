package base

import (
	"net/http"
	"testing"

	"github.com/Dorico-Dynamics/txova-go-core/errors"
)

func TestHTTPStatusForCode(t *testing.T) {
	tests := []struct {
		name     string
		code     errors.Code
		expected int
	}{
		{
			name:     "timeout code",
			code:     CodeTimeout,
			expected: http.StatusGatewayTimeout,
		},
		{
			name:     "circuit open code",
			code:     CodeCircuitOpen,
			expected: http.StatusServiceUnavailable,
		},
		{
			name:     "bad gateway code",
			code:     CodeBadGateway,
			expected: http.StatusBadGateway,
		},
		{
			name:     "core validation error code",
			code:     errors.CodeValidationError,
			expected: http.StatusBadRequest,
		},
		{
			name:     "core not found code",
			code:     errors.CodeNotFound,
			expected: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HTTPStatusForCode(tt.code)
			if got != tt.expected {
				t.Errorf("HTTPStatusForCode(%s) = %d, want %d", tt.code, got, tt.expected)
			}
		})
	}
}

func TestErrorConstructors(t *testing.T) {
	t.Run("ErrTimeout", func(t *testing.T) {
		err := ErrTimeout("request timed out")
		if err.Code() != CodeTimeout {
			t.Errorf("expected code %s, got %s", CodeTimeout, err.Code())
		}
		if err.Message() != "request timed out" {
			t.Errorf("expected message 'request timed out', got %s", err.Message())
		}
	})

	t.Run("ErrTimeoutf", func(t *testing.T) {
		err := ErrTimeoutf("request to %s timed out after %d ms", "service", 5000)
		if err.Code() != CodeTimeout {
			t.Errorf("expected code %s, got %s", CodeTimeout, err.Code())
		}
		expected := "request to service timed out after 5000 ms"
		if err.Message() != expected {
			t.Errorf("expected message '%s', got %s", expected, err.Message())
		}
	})

	t.Run("ErrTimeoutWrap", func(t *testing.T) {
		cause := errors.New(errors.CodeInternalError, "network error")
		err := ErrTimeoutWrap("request failed", cause)
		if err.Code() != CodeTimeout {
			t.Errorf("expected code %s, got %s", CodeTimeout, err.Code())
		}
		if err.Unwrap() == nil {
			t.Error("expected wrapped error, got nil")
		}
	})

	t.Run("ErrCircuitOpen", func(t *testing.T) {
		err := ErrCircuitOpen("user-service")
		if err.Code() != CodeCircuitOpen {
			t.Errorf("expected code %s, got %s", CodeCircuitOpen, err.Code())
		}
		expected := "circuit breaker open for user-service"
		if err.Message() != expected {
			t.Errorf("expected message '%s', got %s", expected, err.Message())
		}
	})

	t.Run("ErrBadGateway", func(t *testing.T) {
		err := ErrBadGateway("invalid response")
		if err.Code() != CodeBadGateway {
			t.Errorf("expected code %s, got %s", CodeBadGateway, err.Code())
		}
	})

	t.Run("ErrBadGatewayWrap", func(t *testing.T) {
		cause := errors.New(errors.CodeInternalError, "parse error")
		err := ErrBadGatewayWrap("failed to parse response", cause)
		if err.Code() != CodeBadGateway {
			t.Errorf("expected code %s, got %s", CodeBadGateway, err.Code())
		}
		if err.Unwrap() == nil {
			t.Error("expected wrapped error, got nil")
		}
	})
}

func TestErrorCheckers(t *testing.T) {
	t.Run("IsTimeout", func(t *testing.T) {
		timeoutErr := ErrTimeout("timeout")
		if !IsTimeout(timeoutErr) {
			t.Error("expected IsTimeout to return true for timeout error")
		}

		otherErr := ErrBadGateway("bad gateway")
		if IsTimeout(otherErr) {
			t.Error("expected IsTimeout to return false for non-timeout error")
		}
	})

	t.Run("IsCircuitOpen", func(t *testing.T) {
		circuitErr := ErrCircuitOpen("service")
		if !IsCircuitOpen(circuitErr) {
			t.Error("expected IsCircuitOpen to return true for circuit open error")
		}

		otherErr := ErrTimeout("timeout")
		if IsCircuitOpen(otherErr) {
			t.Error("expected IsCircuitOpen to return false for non-circuit error")
		}
	})

	t.Run("IsBadGateway", func(t *testing.T) {
		bgErr := ErrBadGateway("bad gateway")
		if !IsBadGateway(bgErr) {
			t.Error("expected IsBadGateway to return true for bad gateway error")
		}

		otherErr := ErrTimeout("timeout")
		if IsBadGateway(otherErr) {
			t.Error("expected IsBadGateway to return false for non-bad-gateway error")
		}
	})
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "timeout error",
			err:      ErrTimeout("timeout"),
			expected: true,
		},
		{
			name:     "service unavailable error",
			err:      errors.ServiceUnavailable("service down"),
			expected: true,
		},
		{
			name:     "rate limited error",
			err:      errors.RateLimited("too many requests"),
			expected: true,
		},
		{
			name:     "internal error",
			err:      errors.InternalError("internal error"),
			expected: true,
		},
		{
			name:     "validation error not retryable",
			err:      errors.ValidationError("bad input"),
			expected: false,
		},
		{
			name:     "not found error not retryable",
			err:      errors.NotFound("not found"),
			expected: false,
		},
		{
			name:     "circuit open not retryable",
			err:      ErrCircuitOpen("service"),
			expected: false,
		},
		{
			name:     "bad gateway not retryable",
			err:      ErrBadGateway("bad gateway"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRetryable(tt.err)
			if got != tt.expected {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsRetryableStatus(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		expected   bool
	}{
		{
			name:       "200 OK not retryable",
			statusCode: http.StatusOK,
			expected:   false,
		},
		{
			name:       "400 Bad Request not retryable",
			statusCode: http.StatusBadRequest,
			expected:   false,
		},
		{
			name:       "401 Unauthorized not retryable",
			statusCode: http.StatusUnauthorized,
			expected:   false,
		},
		{
			name:       "404 Not Found not retryable",
			statusCode: http.StatusNotFound,
			expected:   false,
		},
		{
			name:       "408 Request Timeout retryable",
			statusCode: http.StatusRequestTimeout,
			expected:   true,
		},
		{
			name:       "429 Too Many Requests retryable",
			statusCode: http.StatusTooManyRequests,
			expected:   true,
		},
		{
			name:       "500 Internal Server Error retryable",
			statusCode: http.StatusInternalServerError,
			expected:   true,
		},
		{
			name:       "502 Bad Gateway retryable",
			statusCode: http.StatusBadGateway,
			expected:   true,
		},
		{
			name:       "503 Service Unavailable retryable",
			statusCode: http.StatusServiceUnavailable,
			expected:   true,
		},
		{
			name:       "504 Gateway Timeout retryable",
			statusCode: http.StatusGatewayTimeout,
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRetryableStatus(tt.statusCode)
			if got != tt.expected {
				t.Errorf("IsRetryableStatus(%d) = %v, want %v", tt.statusCode, got, tt.expected)
			}
		})
	}
}

func TestMapHTTPStatus(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		body         []byte
		expectedCode errors.Code
	}{
		{
			name:         "400 maps to validation error",
			statusCode:   http.StatusBadRequest,
			body:         nil,
			expectedCode: errors.CodeValidationError,
		},
		{
			name:         "401 maps to invalid credentials",
			statusCode:   http.StatusUnauthorized,
			body:         nil,
			expectedCode: errors.CodeInvalidCredentials,
		},
		{
			name:         "403 maps to forbidden",
			statusCode:   http.StatusForbidden,
			body:         nil,
			expectedCode: errors.CodeForbidden,
		},
		{
			name:         "404 maps to not found",
			statusCode:   http.StatusNotFound,
			body:         nil,
			expectedCode: errors.CodeNotFound,
		},
		{
			name:         "409 maps to conflict",
			statusCode:   http.StatusConflict,
			body:         nil,
			expectedCode: errors.CodeConflict,
		},
		{
			name:         "429 maps to rate limited",
			statusCode:   http.StatusTooManyRequests,
			body:         nil,
			expectedCode: errors.CodeRateLimited,
		},
		{
			name:         "408 maps to timeout",
			statusCode:   http.StatusRequestTimeout,
			body:         nil,
			expectedCode: CodeTimeout,
		},
		{
			name:         "504 maps to timeout",
			statusCode:   http.StatusGatewayTimeout,
			body:         nil,
			expectedCode: CodeTimeout,
		},
		{
			name:         "502 maps to bad gateway",
			statusCode:   http.StatusBadGateway,
			body:         nil,
			expectedCode: CodeBadGateway,
		},
		{
			name:         "503 maps to service unavailable",
			statusCode:   http.StatusServiceUnavailable,
			body:         nil,
			expectedCode: errors.CodeServiceUnavailable,
		},
		{
			name:         "500 maps to internal error",
			statusCode:   http.StatusInternalServerError,
			body:         nil,
			expectedCode: errors.CodeInternalError,
		},
		{
			name:         "other 5xx maps to internal error",
			statusCode:   505,
			body:         nil,
			expectedCode: errors.CodeInternalError,
		},
		{
			name:         "other 4xx maps to validation error",
			statusCode:   418,
			body:         nil,
			expectedCode: errors.CodeValidationError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := MapHTTPStatus(tt.statusCode, tt.body)
			if err.Code() != tt.expectedCode {
				t.Errorf("MapHTTPStatus(%d) code = %s, want %s", tt.statusCode, err.Code(), tt.expectedCode)
			}
		})
	}
}

func TestMapHTTPStatusWithBody(t *testing.T) {
	t.Run("parses standard error response", func(t *testing.T) {
		body := []byte(`{"error":{"code":"CUSTOM_ERROR","message":"custom error message"}}`)
		err := MapHTTPStatus(http.StatusBadRequest, body)
		if err.Code() != "CUSTOM_ERROR" {
			t.Errorf("expected code CUSTOM_ERROR, got %s", err.Code())
		}
		if err.Message() != "custom error message" {
			t.Errorf("expected message 'custom error message', got %s", err.Message())
		}
	})

	t.Run("extracts message from body", func(t *testing.T) {
		body := []byte(`{"message":"specific error message"}`)
		err := MapHTTPStatus(http.StatusBadRequest, body)
		if err.Message() != "specific error message" {
			t.Errorf("expected message 'specific error message', got %s", err.Message())
		}
	})

	t.Run("uses short body as message", func(t *testing.T) {
		body := []byte("short error text")
		err := MapHTTPStatus(http.StatusBadRequest, body)
		if err.Message() != "short error text" {
			t.Errorf("expected message 'short error text', got %s", err.Message())
		}
	})

	t.Run("uses default message for empty body", func(t *testing.T) {
		err := MapHTTPStatus(http.StatusNotFound, nil)
		if err.Message() != "not found" {
			t.Errorf("expected message 'not found', got %s", err.Message())
		}
	})
}
