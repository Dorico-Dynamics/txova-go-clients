package base

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	txcontext "github.com/Dorico-Dynamics/txova-go-core/context"
	"github.com/Dorico-Dynamics/txova-go-core/logging"
)

func TestNewClient(t *testing.T) {
	t.Run("creates client with valid config", func(t *testing.T) {
		cfg := &Config{
			BaseURL:        "https://api.example.com",
			Timeout:        30 * time.Second,
			RequestTimeout: 10 * time.Second,
			Retry:          DefaultRetryConfig(),
		}

		client, err := NewClient(cfg, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client.BaseURL() != "https://api.example.com" {
			t.Errorf("expected BaseURL 'https://api.example.com', got %s", client.BaseURL())
		}
	})

	t.Run("creates client with nil config using defaults", func(t *testing.T) {
		_, err := NewClient(nil, nil)
		if err == nil {
			t.Fatal("expected error for nil config without BaseURL")
		}
	})

	t.Run("trims trailing slash from base URL", func(t *testing.T) {
		cfg := &Config{
			BaseURL:        "https://api.example.com/",
			Timeout:        30 * time.Second,
			RequestTimeout: 10 * time.Second,
			Retry:          DefaultRetryConfig(),
		}

		client, err := NewClient(cfg, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if client.BaseURL() != "https://api.example.com" {
			t.Errorf("expected BaseURL without trailing slash, got %s", client.BaseURL())
		}
	})

	t.Run("fails with invalid config", func(t *testing.T) {
		cfg := &Config{
			BaseURL: "",
		}

		_, err := NewClient(cfg, nil)
		if err == nil {
			t.Error("expected error for invalid config")
		}
	})

	t.Run("creates circuit breaker when configured", func(t *testing.T) {
		cfg := &Config{
			BaseURL:        "https://api.example.com",
			Timeout:        30 * time.Second,
			RequestTimeout: 10 * time.Second,
			Retry:          DefaultRetryConfig(),
			CircuitBreaker: DefaultCircuitBreakerConfig("test"),
		}

		client, err := NewClient(cfg, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		stats := client.CircuitBreakerStats()
		if stats == nil {
			t.Error("expected circuit breaker stats")
		}
	})

	t.Run("uses custom transport when provided", func(t *testing.T) {
		transport := &http.Transport{
			MaxIdleConns: 50,
		}

		cfg := &Config{
			BaseURL:        "https://api.example.com",
			Timeout:        30 * time.Second,
			RequestTimeout: 10 * time.Second,
			Retry:          DefaultRetryConfig(),
			Transport:      transport,
		}

		_, err := NewClient(cfg, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestClientDo(t *testing.T) {
	t.Run("successful GET request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		}))
		defer server.Close()

		client, err := NewClient(&Config{
			BaseURL:        server.URL,
			Timeout:        30 * time.Second,
			RequestTimeout: 10 * time.Second,
			Retry:          RetryConfig{MaxRetries: 0},
		}, nil)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		resp, err := client.Get(context.Background(), "/test").Do()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("POST request with body", func(t *testing.T) {
		var receivedBody map[string]string

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
			}

			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &receivedBody)

			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"123"}`))
		}))
		defer server.Close()

		client, err := NewClient(&Config{
			BaseURL:        server.URL,
			Timeout:        30 * time.Second,
			RequestTimeout: 10 * time.Second,
			Retry:          RetryConfig{MaxRetries: 0},
		}, nil)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		reqBody := map[string]string{"name": "test"}
		resp, err := client.Post(context.Background(), "/users", reqBody).Do()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Errorf("expected status 201, got %d", resp.StatusCode)
		}
		if receivedBody["name"] != "test" {
			t.Errorf("expected body name 'test', got %s", receivedBody["name"])
		}
	})

	t.Run("adds tracing headers from context", func(t *testing.T) {
		var receivedRequestID, receivedCorrelationID string

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedRequestID = r.Header.Get(txcontext.HeaderRequestID)
			receivedCorrelationID = r.Header.Get(txcontext.HeaderCorrelationID)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client, err := NewClient(&Config{
			BaseURL:        server.URL,
			Timeout:        30 * time.Second,
			RequestTimeout: 10 * time.Second,
			Retry:          RetryConfig{MaxRetries: 0},
		}, nil)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		ctx := context.Background()
		ctx = txcontext.WithRequestID(ctx, "req-123")
		ctx = txcontext.WithCorrelationID(ctx, "corr-456")

		_, err = client.Get(ctx, "/test").Do()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if receivedRequestID != "req-123" {
			t.Errorf("expected X-Request-ID 'req-123', got %s", receivedRequestID)
		}
		if receivedCorrelationID != "corr-456" {
			t.Errorf("expected X-Correlation-ID 'corr-456', got %s", receivedCorrelationID)
		}
	})

	t.Run("retries on server error", func(t *testing.T) {
		var attempts int32

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			count := atomic.AddInt32(&attempts, 1)
			if count < 3 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		}))
		defer server.Close()

		client, err := NewClient(&Config{
			BaseURL:        server.URL,
			Timeout:        30 * time.Second,
			RequestTimeout: 10 * time.Second,
			Retry: RetryConfig{
				MaxRetries:  3,
				InitialWait: 10 * time.Millisecond,
				MaxWait:     100 * time.Millisecond,
				Multiplier:  2.0,
				Jitter:      0,
			},
		}, nil)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		resp, err := client.Get(context.Background(), "/test").Do()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
		if attempts != 3 {
			t.Errorf("expected 3 attempts, got %d", attempts)
		}
	})

	t.Run("returns error when circuit breaker is open", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client, err := NewClient(&Config{
			BaseURL:        server.URL,
			Timeout:        30 * time.Second,
			RequestTimeout: 10 * time.Second,
			Retry:          RetryConfig{MaxRetries: 0},
			CircuitBreaker: &CircuitBreakerConfig{
				FailureThreshold: 2,
				SuccessThreshold: 1,
				Timeout:          30 * time.Second,
			},
		}, nil)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		_, _ = client.Get(context.Background(), "/test").Do()
		_, _ = client.Get(context.Background(), "/test").Do()

		_, err = client.Get(context.Background(), "/test").Do()
		if err == nil {
			t.Error("expected error when circuit breaker is open")
		}
		if !IsCircuitOpen(err) {
			t.Errorf("expected circuit open error, got %v", err)
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client, err := NewClient(&Config{
			BaseURL:        server.URL,
			Timeout:        30 * time.Second,
			RequestTimeout: 10 * time.Second,
			Retry:          RetryConfig{MaxRetries: 0},
		}, nil)
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err = client.Get(ctx, "/test").Do()
		if err == nil {
			t.Error("expected error for cancelled context")
		}
	})
}

func TestRequestBuilder(t *testing.T) {
	t.Run("WithHeader", func(t *testing.T) {
		var receivedHeader string

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedHeader = r.Header.Get("X-Custom-Header")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client, _ := NewClient(&Config{
			BaseURL:        server.URL,
			Timeout:        30 * time.Second,
			RequestTimeout: 10 * time.Second,
			Retry:          RetryConfig{MaxRetries: 0},
		}, nil)

		_, _ = client.Get(context.Background(), "/test").
			WithHeader("X-Custom-Header", "custom-value").
			Do()

		if receivedHeader != "custom-value" {
			t.Errorf("expected header 'custom-value', got %s", receivedHeader)
		}
	})

	t.Run("WithQuery", func(t *testing.T) {
		var receivedQuery string

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedQuery = r.URL.Query().Get("filter")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client, _ := NewClient(&Config{
			BaseURL:        server.URL,
			Timeout:        30 * time.Second,
			RequestTimeout: 10 * time.Second,
			Retry:          RetryConfig{MaxRetries: 0},
		}, nil)

		_, _ = client.Get(context.Background(), "/test").
			WithQuery("filter", "active").
			Do()

		if receivedQuery != "active" {
			t.Errorf("expected query 'active', got %s", receivedQuery)
		}
	})

	t.Run("WithQueryParams", func(t *testing.T) {
		var receivedValues url.Values

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedValues = r.URL.Query()
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client, _ := NewClient(&Config{
			BaseURL:        server.URL,
			Timeout:        30 * time.Second,
			RequestTimeout: 10 * time.Second,
			Retry:          RetryConfig{MaxRetries: 0},
		}, nil)

		params := url.Values{
			"filter": []string{"active"},
			"sort":   []string{"name"},
		}

		_, _ = client.Get(context.Background(), "/test").
			WithQueryParams(params).
			Do()

		if receivedValues.Get("filter") != "active" {
			t.Errorf("expected filter 'active', got %s", receivedValues.Get("filter"))
		}
		if receivedValues.Get("sort") != "name" {
			t.Errorf("expected sort 'name', got %s", receivedValues.Get("sort"))
		}
	})

	t.Run("Decode", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"name":"test","value":42}`))
		}))
		defer server.Close()

		client, _ := NewClient(&Config{
			BaseURL:        server.URL,
			Timeout:        30 * time.Second,
			RequestTimeout: 10 * time.Second,
			Retry:          RetryConfig{MaxRetries: 0},
		}, nil)

		var result struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}

		err := client.Get(context.Background(), "/test").Decode(&result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Name != "test" {
			t.Errorf("expected name 'test', got %s", result.Name)
		}
		if result.Value != 42 {
			t.Errorf("expected value 42, got %d", result.Value)
		}
	})
}

func TestHTTPMethods(t *testing.T) {
	methods := []struct {
		name     string
		method   string
		callFunc func(c *Client, ctx context.Context, path string) *Request
	}{
		{"GET", http.MethodGet, func(c *Client, ctx context.Context, path string) *Request { return c.Get(ctx, path) }},
		{"POST", http.MethodPost, func(c *Client, ctx context.Context, path string) *Request { return c.Post(ctx, path, nil) }},
		{"PUT", http.MethodPut, func(c *Client, ctx context.Context, path string) *Request { return c.Put(ctx, path, nil) }},
		{"PATCH", http.MethodPatch, func(c *Client, ctx context.Context, path string) *Request { return c.Patch(ctx, path, nil) }},
		{"DELETE", http.MethodDelete, func(c *Client, ctx context.Context, path string) *Request { return c.Delete(ctx, path) }},
	}

	for _, m := range methods {
		t.Run(m.name, func(t *testing.T) {
			var receivedMethod string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedMethod = r.Method
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			client, _ := NewClient(&Config{
				BaseURL:        server.URL,
				Timeout:        30 * time.Second,
				RequestTimeout: 10 * time.Second,
				Retry:          RetryConfig{MaxRetries: 0},
			}, nil)

			_, err := m.callFunc(client, context.Background(), "/test").Do()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if receivedMethod != m.method {
				t.Errorf("expected method %s, got %s", m.method, receivedMethod)
			}
		})
	}
}

func TestClientWithLogger(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	var logOutput strings.Builder
	logger := logging.New(logging.Config{
		Level:  slog.LevelDebug,
		Format: logging.FormatText,
		Output: &logOutput,
	})

	client, err := NewClient(&Config{
		BaseURL:        server.URL,
		Timeout:        30 * time.Second,
		RequestTimeout: 10 * time.Second,
		Retry:          RetryConfig{MaxRetries: 0},
	}, logger)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = client.Get(context.Background(), "/test").Do()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if logOutput.Len() == 0 {
		t.Error("expected log output")
	}
}

func TestClientCircuitBreakerStats(t *testing.T) {
	t.Run("returns nil when no circuit breaker", func(t *testing.T) {
		client, _ := NewClient(&Config{
			BaseURL:        "https://api.example.com",
			Timeout:        30 * time.Second,
			RequestTimeout: 10 * time.Second,
			Retry:          RetryConfig{MaxRetries: 0},
		}, nil)

		stats := client.CircuitBreakerStats()
		if stats != nil {
			t.Error("expected nil stats when no circuit breaker")
		}
	})
}
