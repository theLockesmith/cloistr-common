// Package errors provides standardized API error types for Cloistr services.
//
// All API errors follow the potato-grade design format:
//
//	{
//	  "code": "STORAGE_TIMEOUT",
//	  "message": "Upload timed out after 30s",
//	  "retry_after": 60,
//	  "debug": {"timeout_at": "write_chunk_3"}
//	}
//
// This format ensures errors are:
//   - Programmatically handleable (via code)
//   - Human readable (via message)
//   - Actionable (via retry_after)
//   - Debuggable (via debug info)
package errors

import (
	"encoding/json"
	"net/http"
)

// APIError represents a standardized API error response.
// All Cloistr services should use this format for error responses.
type APIError struct {
	// Code is a machine-readable error code (e.g., "STORAGE_TIMEOUT").
	Code string `json:"code"`

	// Message is a human-readable error description.
	Message string `json:"message"`

	// RetryAfter suggests when to retry, in seconds. 0 means don't retry.
	RetryAfter int `json:"retry_after,omitempty"`

	// Debug contains additional debugging information.
	// Only populated when debug mode is enabled.
	Debug map[string]any `json:"debug,omitempty"`

	// HTTPStatus is the HTTP status code to return.
	// Not serialized to JSON - used internally.
	HTTPStatus int `json:"-"`
}

// Error implements the error interface.
func (e *APIError) Error() string {
	return e.Message
}

// WithDebug adds debug information to the error.
func (e *APIError) WithDebug(key string, value any) *APIError {
	if e.Debug == nil {
		e.Debug = make(map[string]any)
	}
	e.Debug[key] = value
	return e
}

// WithRetryAfter sets the retry-after duration in seconds.
func (e *APIError) WithRetryAfter(seconds int) *APIError {
	e.RetryAfter = seconds
	return e
}

// JSON returns the error as a JSON byte slice.
func (e *APIError) JSON() []byte {
	data, _ := json.Marshal(e)
	return data
}

// WriteResponse writes the error as an HTTP response.
func (e *APIError) WriteResponse(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	if e.RetryAfter > 0 {
		w.Header().Set("Retry-After", string(rune(e.RetryAfter)))
	}
	w.WriteHeader(e.HTTPStatus)
	w.Write(e.JSON())
}

// New creates a new APIError with the given code, message, and HTTP status.
func New(code string, message string, httpStatus int) *APIError {
	return &APIError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
}

// Common error constructors for standard HTTP error responses.

// BadRequest creates a 400 Bad Request error.
func BadRequest(code, message string) *APIError {
	return New(code, message, http.StatusBadRequest)
}

// Unauthorized creates a 401 Unauthorized error.
func Unauthorized(code, message string) *APIError {
	return New(code, message, http.StatusUnauthorized)
}

// Forbidden creates a 403 Forbidden error.
func Forbidden(code, message string) *APIError {
	return New(code, message, http.StatusForbidden)
}

// NotFound creates a 404 Not Found error.
func NotFound(code, message string) *APIError {
	return New(code, message, http.StatusNotFound)
}

// Conflict creates a 409 Conflict error.
func Conflict(code, message string) *APIError {
	return New(code, message, http.StatusConflict)
}

// TooManyRequests creates a 429 Too Many Requests error.
func TooManyRequests(code, message string, retryAfter int) *APIError {
	return New(code, message, http.StatusTooManyRequests).WithRetryAfter(retryAfter)
}

// InternalError creates a 500 Internal Server Error.
func InternalError(code, message string) *APIError {
	return New(code, message, http.StatusInternalServerError)
}

// ServiceUnavailable creates a 503 Service Unavailable error.
func ServiceUnavailable(code, message string, retryAfter int) *APIError {
	return New(code, message, http.StatusServiceUnavailable).WithRetryAfter(retryAfter)
}

// InsufficientStorage creates a 507 Insufficient Storage error.
func InsufficientStorage(code, message string) *APIError {
	return New(code, message, http.StatusInsufficientStorage)
}

// StatusAndBody returns the HTTP status code and the error itself for use
// with frameworks like Gin. Usage: ctx.JSON(err.StatusAndBody())
func (e *APIError) StatusAndBody() (int, *APIError) {
	return e.HTTPStatus, e
}

// Abort writes the error and aborts the request. For use with Gin:
// err.Abort(ctx) is equivalent to ctx.AbortWithStatusJSON(err.StatusAndBody())
// The ctx parameter must have AbortWithStatusJSON(int, any) method.
type GinContext interface {
	AbortWithStatusJSON(code int, jsonObj any)
}

func (e *APIError) Abort(ctx GinContext) {
	ctx.AbortWithStatusJSON(e.HTTPStatus, e)
}
