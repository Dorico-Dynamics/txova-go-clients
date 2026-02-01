// Package base provides the foundation HTTP client for the Txova platform.
// It includes connection pooling, retry logic, circuit breaker, and request tracing.
package base

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Dorico-Dynamics/txova-go-core/errors"
)

// Client-specific error codes extending txova-go-core/errors.
const (
	// CodeTimeout indicates a request timed out.
	CodeTimeout errors.Code = "TIMEOUT"
	// CodeCircuitOpen indicates the circuit breaker is open.
	CodeCircuitOpen errors.Code = "CIRCUIT_OPEN"
	// CodeBadGateway indicates the upstream service returned an invalid response.
	CodeBadGateway errors.Code = "BAD_GATEWAY"
)

// codeHTTPStatus maps client-specific error codes to HTTP status codes.
var codeHTTPStatus = map[errors.Code]int{
	CodeTimeout:     http.StatusGatewayTimeout,
	CodeCircuitOpen: http.StatusServiceUnavailable,
	CodeBadGateway:  http.StatusBadGateway,
}

// HTTPStatusForCode returns the HTTP status code for a client-specific error code.
// Falls back to the core errors package for standard codes.
func HTTPStatusForCode(code errors.Code) int {
	if status, ok := codeHTTPStatus[code]; ok {
		return status
	}
	return code.HTTPStatus()
}

// Error constructors for client-specific errors.

// ErrTimeout creates a timeout error.
func ErrTimeout(message string) *errors.AppError {
	return errors.New(CodeTimeout, message)
}

// ErrTimeoutf creates a timeout error with a formatted message.
func ErrTimeoutf(format string, args ...any) *errors.AppError {
	return errors.New(CodeTimeout, fmt.Sprintf(format, args...))
}

// ErrTimeoutWrap creates a timeout error wrapping an existing error.
func ErrTimeoutWrap(message string, cause error) *errors.AppError {
	return errors.Wrap(CodeTimeout, message, cause)
}

// ErrCircuitOpen creates a circuit breaker open error.
func ErrCircuitOpen(service string) *errors.AppError {
	return errors.New(CodeCircuitOpen, fmt.Sprintf("circuit breaker open for %s", service))
}

// ErrBadGateway creates a bad gateway error for invalid upstream responses.
func ErrBadGateway(message string) *errors.AppError {
	return errors.New(CodeBadGateway, message)
}

// ErrBadGatewayWrap creates a bad gateway error wrapping an existing error.
func ErrBadGatewayWrap(message string, cause error) *errors.AppError {
	return errors.Wrap(CodeBadGateway, message, cause)
}

// IsTimeout checks if the error is a timeout error.
func IsTimeout(err error) bool {
	return errors.IsCode(err, CodeTimeout)
}

// IsCircuitOpen checks if the error is a circuit breaker open error.
func IsCircuitOpen(err error) bool {
	return errors.IsCode(err, CodeCircuitOpen)
}

// IsBadGateway checks if the error is a bad gateway error.
func IsBadGateway(err error) bool {
	return errors.IsCode(err, CodeBadGateway)
}

// IsRetryable returns true if the error is retryable.
// Retryable errors include: timeouts, rate limits, service unavailable, and server errors.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	appErr := errors.AsAppError(err)
	if appErr == nil {
		return false
	}

	switch appErr.Code() {
	case CodeTimeout,
		errors.CodeServiceUnavailable,
		errors.CodeRateLimited,
		errors.CodeInternalError:
		return true
	default:
		return false
	}
}

// MapHTTPStatus maps an HTTP status code and response body to an AppError.
// It attempts to parse the standard Txova error response format first.
func MapHTTPStatus(statusCode int, body []byte) *errors.AppError {
	// Try to parse standard error response.
	var errResp errors.ErrorResponse
	if len(body) > 0 {
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Code != "" {
			code := errors.Code(errResp.Error.Code)
			return errors.New(code, errResp.Error.Message)
		}
	}

	// Map status code to error.
	switch statusCode {
	case http.StatusBadRequest:
		return errors.ValidationError(extractMessage(body, "bad request"))
	case http.StatusUnauthorized:
		return errors.InvalidCredentials(extractMessage(body, "unauthorized"))
	case http.StatusForbidden:
		return errors.Forbidden(extractMessage(body, "forbidden"))
	case http.StatusNotFound:
		return errors.NotFound(extractMessage(body, "not found"))
	case http.StatusConflict:
		return errors.Conflict(extractMessage(body, "conflict"))
	case http.StatusTooManyRequests:
		return errors.RateLimited(extractMessage(body, "rate limited"))
	case http.StatusRequestTimeout, http.StatusGatewayTimeout:
		return ErrTimeout(extractMessage(body, "request timeout"))
	case http.StatusBadGateway:
		return ErrBadGateway(extractMessage(body, "bad gateway"))
	case http.StatusServiceUnavailable:
		return errors.ServiceUnavailable(extractMessage(body, "service unavailable"))
	default:
		if statusCode >= 500 {
			return errors.InternalError(extractMessage(body, "server error"))
		}
		return errors.ValidationError(extractMessage(body, fmt.Sprintf("request failed with status %d", statusCode)))
	}
}

// IsRetryableStatus returns true if the HTTP status code is retryable.
func IsRetryableStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusRequestTimeout,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

// extractMessage extracts a message from a response body or returns a default.
func extractMessage(body []byte, defaultMsg string) string {
	if len(body) == 0 {
		return defaultMsg
	}

	// Try to extract message from JSON.
	msg := extractJSONMessage(body)
	if msg != "" {
		return msg
	}

	// Return body as string if short enough.
	if len(body) <= 200 {
		return string(body)
	}

	return defaultMsg
}

// extractJSONMessage attempts to extract a message from JSON body.
func extractJSONMessage(body []byte) string {
	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return ""
	}

	if msg, ok := data["message"].(string); ok && msg != "" {
		return msg
	}

	errObj, ok := data["error"].(map[string]any)
	if !ok {
		return ""
	}

	if msg, ok := errObj["message"].(string); ok && msg != "" {
		return msg
	}

	return ""
}
