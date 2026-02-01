package base

import (
	"net/http"
	"testing"

	"github.com/Dorico-Dynamics/txova-go-core/errors"
)

func TestResponseIsSuccess(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		expected   bool
	}{
		{"200 OK", http.StatusOK, true},
		{"201 Created", http.StatusCreated, true},
		{"204 No Content", http.StatusNoContent, true},
		{"299 edge case", 299, true},
		{"300 Multiple Choices", 300, false},
		{"301 Moved", http.StatusMovedPermanently, false},
		{"400 Bad Request", http.StatusBadRequest, false},
		{"404 Not Found", http.StatusNotFound, false},
		{"500 Internal Error", http.StatusInternalServerError, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &Response{StatusCode: tt.statusCode}
			if got := resp.IsSuccess(); got != tt.expected {
				t.Errorf("IsSuccess() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestResponseIsClientError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		expected   bool
	}{
		{"200 OK", http.StatusOK, false},
		{"300 Multiple Choices", 300, false},
		{"400 Bad Request", http.StatusBadRequest, true},
		{"401 Unauthorized", http.StatusUnauthorized, true},
		{"403 Forbidden", http.StatusForbidden, true},
		{"404 Not Found", http.StatusNotFound, true},
		{"429 Too Many Requests", http.StatusTooManyRequests, true},
		{"499 edge case", 499, true},
		{"500 Internal Error", http.StatusInternalServerError, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &Response{StatusCode: tt.statusCode}
			if got := resp.IsClientError(); got != tt.expected {
				t.Errorf("IsClientError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestResponseIsServerError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		expected   bool
	}{
		{"200 OK", http.StatusOK, false},
		{"400 Bad Request", http.StatusBadRequest, false},
		{"499 edge case", 499, false},
		{"500 Internal Error", http.StatusInternalServerError, true},
		{"502 Bad Gateway", http.StatusBadGateway, true},
		{"503 Service Unavailable", http.StatusServiceUnavailable, true},
		{"504 Gateway Timeout", http.StatusGatewayTimeout, true},
		{"599 edge case", 599, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &Response{StatusCode: tt.statusCode}
			if got := resp.IsServerError(); got != tt.expected {
				t.Errorf("IsServerError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestResponseDecode(t *testing.T) {
	t.Run("decodes successful JSON response", func(t *testing.T) {
		resp := &Response{
			StatusCode: http.StatusOK,
			Body:       []byte(`{"name":"test","value":123}`),
		}

		var dest struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}

		err := resp.Decode(&dest)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if dest.Name != "test" {
			t.Errorf("expected name 'test', got %s", dest.Name)
		}
		if dest.Value != 123 {
			t.Errorf("expected value 123, got %d", dest.Value)
		}
	})

	t.Run("handles empty body", func(t *testing.T) {
		resp := &Response{
			StatusCode: http.StatusNoContent,
			Body:       nil,
		}

		var dest struct {
			Name string `json:"name"`
		}

		err := resp.Decode(&dest)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("returns error for non-success status", func(t *testing.T) {
		resp := &Response{
			StatusCode: http.StatusNotFound,
			Body:       []byte(`{"error":{"code":"NOT_FOUND","message":"resource not found"}}`),
		}

		var dest struct{}
		err := resp.Decode(&dest)
		if err == nil {
			t.Error("expected error, got nil")
		}

		appErr := errors.AsAppError(err)
		if appErr == nil {
			t.Error("expected AppError")
		} else if appErr.Code() != errors.CodeNotFound {
			t.Errorf("expected code NOT_FOUND, got %s", appErr.Code())
		}
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		resp := &Response{
			StatusCode: http.StatusOK,
			Body:       []byte(`invalid json`),
		}

		var dest struct {
			Name string `json:"name"`
		}

		err := resp.Decode(&dest)
		if err == nil {
			t.Error("expected error, got nil")
		}
		if !IsBadGateway(err) {
			t.Errorf("expected BadGateway error, got %v", err)
		}
	})
}

func TestResponseDecodeError(t *testing.T) {
	t.Run("returns nil for success status", func(t *testing.T) {
		resp := &Response{
			StatusCode: http.StatusOK,
			Body:       []byte(`{"name":"test"}`),
		}

		err := resp.DecodeError()
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("returns error for client error status", func(t *testing.T) {
		resp := &Response{
			StatusCode: http.StatusBadRequest,
			Body:       []byte(`{"error":{"code":"VALIDATION_ERROR","message":"invalid input"}}`),
		}

		err := resp.DecodeError()
		if err == nil {
			t.Error("expected error, got nil")
		}

		appErr := errors.AsAppError(err)
		if appErr == nil {
			t.Error("expected AppError")
		}
	})

	t.Run("returns error for server error status", func(t *testing.T) {
		resp := &Response{
			StatusCode: http.StatusInternalServerError,
			Body:       []byte(`{"error":{"code":"INTERNAL_ERROR","message":"something went wrong"}}`),
		}

		err := resp.DecodeError()
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestResponseString(t *testing.T) {
	resp := &Response{
		StatusCode: http.StatusOK,
		Body:       []byte("response body content"),
	}

	if got := resp.String(); got != "response body content" {
		t.Errorf("String() = %s, want 'response body content'", got)
	}
}

func TestResponseHeader(t *testing.T) {
	resp := &Response{
		StatusCode: http.StatusOK,
		Headers: http.Header{
			"Content-Type": []string{"application/json"},
			"X-Custom":     []string{"custom-value"},
			"X-Request-ID": []string{"req-123"},
		},
	}

	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{"Content-Type", "Content-Type", "application/json"},
		{"Custom header", "X-Custom", "custom-value"},
		{"Missing header", "X-Missing", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resp.Header(tt.key); got != tt.expected {
				t.Errorf("Header(%s) = %s, want %s", tt.key, got, tt.expected)
			}
		})
	}
}

func TestResponseRequestID(t *testing.T) {
	t.Run("returns request ID when present", func(t *testing.T) {
		headers := make(http.Header)
		headers.Set("X-Request-ID", "req-abc-123")
		resp := &Response{
			StatusCode: http.StatusOK,
			Headers:    headers,
		}

		if got := resp.RequestID(); got != "req-abc-123" {
			t.Errorf("RequestID() = %s, want 'req-abc-123'", got)
		}
	})

	t.Run("returns empty string when missing", func(t *testing.T) {
		resp := &Response{
			StatusCode: http.StatusOK,
			Headers:    http.Header{},
		}

		if got := resp.RequestID(); got != "" {
			t.Errorf("RequestID() = %s, want ''", got)
		}
	})
}
