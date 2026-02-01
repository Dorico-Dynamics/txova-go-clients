package base

import (
	"encoding/json"
	"net/http"
)

// Response represents an HTTP response.
type Response struct {
	// StatusCode is the HTTP status code.
	StatusCode int

	// Headers contains the response headers.
	Headers http.Header

	// Body contains the raw response body.
	Body []byte
}

// IsSuccess returns true if the response has a 2xx status code.
func (r *Response) IsSuccess() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

// IsClientError returns true if the response has a 4xx status code.
func (r *Response) IsClientError() bool {
	return r.StatusCode >= 400 && r.StatusCode < 500
}

// IsServerError returns true if the response has a 5xx status code.
func (r *Response) IsServerError() bool {
	return r.StatusCode >= 500
}

// Decode decodes the response body into the given destination.
// If the response is not a 2xx status, it returns an appropriate error.
func (r *Response) Decode(dest any) error {
	// Check for error status codes.
	if !r.IsSuccess() {
		return MapHTTPStatus(r.StatusCode, r.Body)
	}

	// Handle empty body.
	if len(r.Body) == 0 {
		return nil
	}

	// Decode JSON.
	if err := json.Unmarshal(r.Body, dest); err != nil {
		return ErrBadGatewayWrap("failed to decode response", err)
	}

	return nil
}

// DecodeError attempts to decode an error response.
// Returns nil if the body cannot be parsed as an error.
func (r *Response) DecodeError() error {
	if r.IsSuccess() {
		return nil
	}

	return MapHTTPStatus(r.StatusCode, r.Body)
}

// String returns the response body as a string.
func (r *Response) String() string {
	return string(r.Body)
}

// Header returns the value of a response header.
func (r *Response) Header(key string) string {
	return r.Headers.Get(key)
}

// RequestID returns the X-Request-ID header value.
func (r *Response) RequestID() string {
	return r.Headers.Get("X-Request-ID")
}
