package base

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	txcontext "github.com/Dorico-Dynamics/txova-go-core/context"
	"github.com/Dorico-Dynamics/txova-go-core/logging"
)

// Client is the base HTTP client for the Txova platform.
// It provides connection pooling, retry logic, circuit breaker, and request tracing.
type Client struct {
	httpClient     *http.Client
	baseURL        string
	logger         *logging.Logger
	retryer        *Retryer
	circuitBreaker *CircuitBreaker
	serviceName    string
}

// NewClient creates a new Client with the given configuration.
func NewClient(cfg *Config, logger *logging.Logger) (*Client, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Apply defaults.
	cfg = cfg.WithDefaults()

	// Validate configuration.
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Create transport.
	transport := cfg.Transport
	if transport == nil {
		transport = &http.Transport{
			MaxIdleConns:        cfg.MaxIdleConns,
			MaxIdleConnsPerHost: cfg.MaxIdleConnsPerHost,
			IdleConnTimeout:     cfg.IdleConnTimeout,
			TLSClientConfig:     cfg.TLSConfig,
			DisableCompression:  false,
			ForceAttemptHTTP2:   true,
		}
	}

	// Create HTTP client.
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   cfg.RequestTimeout,
	}

	// Create circuit breaker if configured.
	var circuitBreaker *CircuitBreaker
	if cfg.CircuitBreaker != nil {
		circuitBreaker = NewCircuitBreaker(cfg.CircuitBreaker)
	}

	// Extract service name from base URL for logging.
	serviceName := extractServiceName(cfg.BaseURL)

	return &Client{
		httpClient:     httpClient,
		baseURL:        strings.TrimSuffix(cfg.BaseURL, "/"),
		logger:         logger,
		retryer:        NewRetryer(cfg.Retry),
		circuitBreaker: circuitBreaker,
		serviceName:    serviceName,
	}, nil
}

// extractServiceName extracts a service name from a URL for logging purposes.
func extractServiceName(baseURL string) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "unknown"
	}
	return u.Host
}

// ErrBodyNotReplayable is returned when a request with a body cannot be retried
// because the body cannot be recreated.
var ErrBodyNotReplayable = fmt.Errorf("request body is not replayable for retries")

// requestState holds state for a request execution.
type requestState struct {
	hasBody       bool
	canReplayBody bool
	startTime     time.Time
}

// Do executes an HTTP request with retry logic and circuit breaker.
func (c *Client) Do(ctx context.Context, req *http.Request) (*Response, error) {
	state := &requestState{
		hasBody:       req.Body != nil && req.Body != http.NoBody,
		canReplayBody: req.GetBody != nil,
		startTime:     time.Now(),
	}

	// Check circuit breaker.
	if c.circuitBreaker != nil && !c.circuitBreaker.Allow() {
		c.logRequest(ctx, req.Method, req.URL.String(), 0, time.Since(state.startTime), ErrCircuitOpen(c.serviceName))
		return nil, ErrCircuitOpen(c.serviceName)
	}

	// Check if request body can be replayed for retries.
	if state.hasBody && !state.canReplayBody && c.retryer.MaxRetries() > 0 {
		c.logRequest(ctx, req.Method, req.URL.String(), 0, time.Since(state.startTime), ErrBodyNotReplayable)
		return nil, ErrBodyNotReplayable
	}

	c.addTracingHeaders(ctx, req)
	c.logRequestStart(ctx, req)

	return c.executeWithRetry(ctx, req, state)
}

// attemptResult holds the result of a single request attempt.
type attemptResult struct {
	statusCode int
	headers    http.Header
	body       []byte
	retry      bool
}

// executeWithRetry executes the request with retry logic.
func (c *Client) executeWithRetry(ctx context.Context, req *http.Request, state *requestState) (*Response, error) {
	var lastErr error

	for attempt := 0; attempt <= c.retryer.MaxRetries(); attempt++ {
		if err := ctx.Err(); err != nil {
			c.logRequest(ctx, req.Method, req.URL.String(), 0, time.Since(state.startTime), err)
			return nil, ErrTimeoutWrap("context cancelled", err)
		}

		result, err := c.executeAttempt(ctx, req, state, attempt)
		if err != nil {
			lastErr = err
			continue
		}
		if result.retry {
			continue
		}

		return c.handleResponse(ctx, req, result, state)
	}

	return c.handleRetriesExhausted(ctx, req, lastErr, state)
}

// executeAttempt executes a single request attempt.
func (c *Client) executeAttempt(ctx context.Context, req *http.Request, state *requestState, attempt int) (*attemptResult, error) {
	reqCopy := req.Clone(ctx)

	// Recreate the body for each attempt.
	if state.hasBody && state.canReplayBody {
		newBody, err := req.GetBody()
		if err != nil {
			c.logRequest(ctx, req.Method, req.URL.String(), 0, time.Since(state.startTime), err)
			return nil, fmt.Errorf("failed to recreate request body: %w", err)
		}
		reqCopy.Body = newBody
	}

	resp, err := c.httpClient.Do(reqCopy)
	if err != nil {
		if reqCopy.Body != nil {
			_ = reqCopy.Body.Close()
		}
		return c.handleRequestError(ctx, req, err, state, attempt)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.recordResult(false)
		c.logRequest(ctx, req.Method, req.URL.String(), resp.StatusCode, time.Since(state.startTime), err)
		return nil, ErrBadGatewayWrap("failed to read response body", err)
	}

	// Check if should retry based on status code.
	if c.retryer.ShouldRetry(resp, nil, attempt) {
		c.logRetry(ctx, req.Method, req.URL.String(), attempt, fmt.Errorf("status %d", resp.StatusCode))
		if waitErr := c.retryer.Wait(ctx, resp, attempt); waitErr != nil {
			c.logRequest(ctx, req.Method, req.URL.String(), resp.StatusCode, time.Since(state.startTime), waitErr)
			return nil, ErrTimeoutWrap("retry wait cancelled", waitErr)
		}
		return &attemptResult{retry: true}, nil
	}

	return &attemptResult{
		statusCode: resp.StatusCode,
		headers:    resp.Header,
		body:       body,
	}, nil
}

// handleRequestError handles errors from httpClient.Do.
func (c *Client) handleRequestError(ctx context.Context, req *http.Request, err error, state *requestState, attempt int) (*attemptResult, error) {
	if c.retryer.ShouldRetry(nil, err, attempt) {
		c.logRetry(ctx, req.Method, req.URL.String(), attempt, err)
		if waitErr := c.retryer.Wait(ctx, nil, attempt); waitErr != nil {
			c.logRequest(ctx, req.Method, req.URL.String(), 0, time.Since(state.startTime), waitErr)
			return nil, ErrTimeoutWrap("retry wait cancelled", waitErr)
		}
		return &attemptResult{retry: true}, nil
	}

	c.recordResult(false)
	c.logRequest(ctx, req.Method, req.URL.String(), 0, time.Since(state.startTime), err)
	return nil, ErrTimeoutWrap("request failed", err)
}

// handleResponse processes a successful response.
func (c *Client) handleResponse(ctx context.Context, req *http.Request, result *attemptResult, state *requestState) (*Response, error) {
	response := &Response{
		StatusCode: result.statusCode,
		Headers:    result.headers,
		Body:       result.body,
	}

	isSuccess := result.statusCode >= 200 && result.statusCode < 500
	c.recordResult(isSuccess)

	var logErr error
	if result.statusCode >= 400 {
		logErr = MapHTTPStatus(result.statusCode, result.body)
	}
	c.logRequest(ctx, req.Method, req.URL.String(), result.statusCode, time.Since(state.startTime), logErr)

	return response, nil
}

// handleRetriesExhausted handles the case when all retries are exhausted.
func (c *Client) handleRetriesExhausted(ctx context.Context, req *http.Request, lastErr error, state *requestState) (*Response, error) {
	c.recordResult(false)
	if lastErr != nil {
		c.logRequest(ctx, req.Method, req.URL.String(), 0, time.Since(state.startTime), lastErr)
		return nil, ErrTimeoutWrap("all retries exhausted", lastErr)
	}

	c.logRequest(ctx, req.Method, req.URL.String(), 0, time.Since(state.startTime), nil)
	return nil, ErrTimeout("all retries exhausted")
}

// recordResult records the result for the circuit breaker.
func (c *Client) recordResult(success bool) {
	if c.circuitBreaker == nil {
		return
	}

	if success {
		c.circuitBreaker.RecordSuccess()
	} else {
		c.circuitBreaker.RecordFailure()
	}
}

// addTracingHeaders adds X-Request-ID and X-Correlation-ID headers from context.
func (c *Client) addTracingHeaders(ctx context.Context, req *http.Request) {
	if requestID := txcontext.RequestID(ctx); requestID != "" {
		req.Header.Set(txcontext.HeaderRequestID, requestID)
	}

	if correlationID := txcontext.CorrelationID(ctx); correlationID != "" {
		req.Header.Set(txcontext.HeaderCorrelationID, correlationID)
	}
}

// logRequestStart logs the start of a request at DEBUG level.
func (c *Client) logRequestStart(ctx context.Context, req *http.Request) {
	if c.logger == nil {
		return
	}

	c.logger.DebugContext(ctx, "http request started",
		"method", req.Method,
		"url", req.URL.String(),
		"service", c.serviceName,
	)
}

// logRequest logs a completed request.
func (c *Client) logRequest(ctx context.Context, method, reqURL string, statusCode int, duration time.Duration, err error) {
	if c.logger == nil {
		return
	}

	attrs := []any{
		"method", method,
		"url", reqURL,
		"service", c.serviceName,
		"duration_ms", duration.Milliseconds(),
	}

	if statusCode > 0 {
		attrs = append(attrs, "status", statusCode)
	}

	if err != nil {
		attrs = append(attrs, "error", err.Error())
		c.logger.WarnContext(ctx, "http request failed", attrs...)
		return
	}

	if statusCode >= 400 {
		c.logger.WarnContext(ctx, "http request completed with error status", attrs...)
		return
	}

	c.logger.DebugContext(ctx, "http request completed", attrs...)
}

// logRetry logs a retry attempt.
func (c *Client) logRetry(ctx context.Context, method, reqURL string, attempt int, err error) {
	if c.logger == nil {
		return
	}

	c.logger.DebugContext(ctx, "http request retrying",
		"method", method,
		"url", reqURL,
		"service", c.serviceName,
		"attempt", attempt+1,
		"max_attempts", c.retryer.MaxRetries()+1,
		"error", err.Error(),
	)
}

// Request methods.

// Get creates a GET request.
func (c *Client) Get(ctx context.Context, path string) *Request {
	return c.newRequest(ctx, http.MethodGet, path, nil)
}

// Post creates a POST request with a JSON body.
func (c *Client) Post(ctx context.Context, path string, body any) *Request {
	return c.newRequest(ctx, http.MethodPost, path, body)
}

// Put creates a PUT request with a JSON body.
func (c *Client) Put(ctx context.Context, path string, body any) *Request {
	return c.newRequest(ctx, http.MethodPut, path, body)
}

// Patch creates a PATCH request with a JSON body.
func (c *Client) Patch(ctx context.Context, path string, body any) *Request {
	return c.newRequest(ctx, http.MethodPatch, path, body)
}

// Delete creates a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) *Request {
	return c.newRequest(ctx, http.MethodDelete, path, nil)
}

// newRequest creates a new Request.
func (c *Client) newRequest(ctx context.Context, method, path string, body any) *Request {
	return &Request{
		client:  c,
		ctx:     ctx,
		method:  method,
		path:    path,
		headers: make(http.Header),
		query:   make(url.Values),
		body:    body,
	}
}

// Request represents an HTTP request being built.
type Request struct {
	client  *Client
	ctx     context.Context
	method  string
	path    string
	headers http.Header
	query   url.Values
	body    any
	err     error
}

// WithHeader adds a header to the request.
func (r *Request) WithHeader(key, value string) *Request {
	r.headers.Set(key, value)
	return r
}

// WithQuery adds a query parameter to the request.
func (r *Request) WithQuery(key, value string) *Request {
	r.query.Set(key, value)
	return r
}

// WithQueryParams adds multiple query parameters to the request.
func (r *Request) WithQueryParams(params url.Values) *Request {
	for key, values := range params {
		for _, value := range values {
			r.query.Add(key, value)
		}
	}
	return r
}

// WithBody sets the request body.
func (r *Request) WithBody(body any) *Request {
	r.body = body
	return r
}

// Do executes the request and returns the response.
func (r *Request) Do() (*Response, error) {
	if r.err != nil {
		return nil, r.err
	}

	// Build URL.
	fullURL := r.client.baseURL + r.path
	if len(r.query) > 0 {
		fullURL += "?" + r.query.Encode()
	}

	// Build body.
	var bodyReader io.Reader
	var bodyBytes []byte
	if r.body != nil {
		var err error
		bodyBytes, err = json.Marshal(r.body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Create HTTP request.
	req, err := http.NewRequestWithContext(r.ctx, r.method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set GetBody function to allow body replay for retries.
	if bodyBytes != nil {
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(bodyBytes)), nil
		}
		req.ContentLength = int64(len(bodyBytes))
	}

	// Set headers.
	for key, values := range r.headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Set content type for requests with body.
	if r.body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Set accept header.
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json")
	}

	return r.client.Do(r.ctx, req)
}

// Decode executes the request and decodes the response into dest.
func (r *Request) Decode(dest any) error {
	resp, err := r.Do()
	if err != nil {
		return err
	}

	return resp.Decode(dest)
}

// CircuitBreakerStats returns the circuit breaker stats, or nil if no circuit breaker.
func (c *Client) CircuitBreakerStats() *CircuitBreakerStats {
	if c.circuitBreaker == nil {
		return nil
	}
	stats := c.circuitBreaker.Stats()
	return &stats
}

// BaseURL returns the base URL of the client.
func (c *Client) BaseURL() string {
	return c.baseURL
}
