package errors

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIError_Error(t *testing.T) {
	err := New("TEST_ERROR", "test message", http.StatusBadRequest)
	if err.Error() != "test message" {
		t.Errorf("Error() = %q, want %q", err.Error(), "test message")
	}
}

func TestAPIError_WithDebug(t *testing.T) {
	err := New("TEST_ERROR", "test", http.StatusBadRequest).
		WithDebug("key1", "value1").
		WithDebug("key2", 42)

	if err.Debug["key1"] != "value1" {
		t.Errorf("Debug[key1] = %v, want %q", err.Debug["key1"], "value1")
	}
	if err.Debug["key2"] != 42 {
		t.Errorf("Debug[key2] = %v, want %d", err.Debug["key2"], 42)
	}
}

func TestAPIError_WithRetryAfter(t *testing.T) {
	err := New("TEST_ERROR", "test", http.StatusTooManyRequests).
		WithRetryAfter(60)

	if err.RetryAfter != 60 {
		t.Errorf("RetryAfter = %d, want %d", err.RetryAfter, 60)
	}
}

func TestAPIError_JSON(t *testing.T) {
	err := New("TEST_ERROR", "test message", http.StatusBadRequest).
		WithRetryAfter(30).
		WithDebug("detail", "info")

	data := err.JSON()
	var decoded map[string]any
	if jsonErr := json.Unmarshal(data, &decoded); jsonErr != nil {
		t.Fatalf("JSON() produced invalid JSON: %v", jsonErr)
	}

	if decoded["code"] != "TEST_ERROR" {
		t.Errorf("code = %v, want %q", decoded["code"], "TEST_ERROR")
	}
	if decoded["message"] != "test message" {
		t.Errorf("message = %v, want %q", decoded["message"], "test message")
	}
	if decoded["retry_after"] != float64(30) {
		t.Errorf("retry_after = %v, want %d", decoded["retry_after"], 30)
	}
	debug, ok := decoded["debug"].(map[string]any)
	if !ok {
		t.Fatal("debug is not a map")
	}
	if debug["detail"] != "info" {
		t.Errorf("debug.detail = %v, want %q", debug["detail"], "info")
	}
}

func TestAPIError_JSON_OmitsEmpty(t *testing.T) {
	err := New("TEST_ERROR", "test", http.StatusBadRequest)
	data := err.JSON()

	var decoded map[string]any
	if jsonErr := json.Unmarshal(data, &decoded); jsonErr != nil {
		t.Fatalf("JSON() produced invalid JSON: %v", jsonErr)
	}

	if _, ok := decoded["retry_after"]; ok {
		t.Error("retry_after should be omitted when zero")
	}
	if _, ok := decoded["debug"]; ok {
		t.Error("debug should be omitted when nil")
	}
}

func TestAPIError_WriteResponse(t *testing.T) {
	err := New("TEST_ERROR", "test message", http.StatusBadRequest)

	rr := httptest.NewRecorder()
	err.WriteResponse(rr)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}

	var decoded map[string]any
	if jsonErr := json.NewDecoder(rr.Body).Decode(&decoded); jsonErr != nil {
		t.Fatalf("response is not valid JSON: %v", jsonErr)
	}
	if decoded["code"] != "TEST_ERROR" {
		t.Errorf("code = %v, want %q", decoded["code"], "TEST_ERROR")
	}
}

func TestAPIError_WriteResponse_RetryAfterHeader(t *testing.T) {
	err := ServiceUnavailable("BUSY", "try later", 60)

	rr := httptest.NewRecorder()
	err.WriteResponse(rr)

	if got := rr.Header().Get("Retry-After"); got != "60" {
		t.Errorf("Retry-After = %q, want %q (regression: string(rune(n)) emits garbage)", got, "60")
	}
}

func TestAPIError_WriteResponse_NoRetryAfterWhenZero(t *testing.T) {
	err := New("TEST", "msg", http.StatusBadRequest) // RetryAfter == 0

	rr := httptest.NewRecorder()
	err.WriteResponse(rr)

	if _, ok := rr.Header()["Retry-After"]; ok {
		t.Error("Retry-After header should be absent when RetryAfter == 0")
	}
}

// fakeGinContext records what Abort does so we can assert both the body and header.
type fakeGinContext struct {
	headers    map[string]string
	abortCode  int
	abortBody  any
	aborted    bool
}

func (f *fakeGinContext) Header(key, value string) {
	if f.headers == nil {
		f.headers = map[string]string{}
	}
	f.headers[key] = value
}

func (f *fakeGinContext) AbortWithStatusJSON(code int, jsonObj any) {
	f.aborted = true
	f.abortCode = code
	f.abortBody = jsonObj
}

func TestAPIError_Abort_SetsRetryAfterHeader(t *testing.T) {
	err := TooManyRequests("RATE_LIMIT", "slow down", 120)

	ctx := &fakeGinContext{}
	err.Abort(ctx)

	if !ctx.aborted {
		t.Fatal("Abort did not call AbortWithStatusJSON")
	}
	if ctx.abortCode != http.StatusTooManyRequests {
		t.Errorf("abort code = %d, want %d", ctx.abortCode, http.StatusTooManyRequests)
	}
	if got := ctx.headers["Retry-After"]; got != "120" {
		t.Errorf("Retry-After = %q, want %q", got, "120")
	}
	if ctx.abortBody != err {
		t.Errorf("abort body = %v, want the APIError itself", ctx.abortBody)
	}
}

func TestAPIError_Abort_NoRetryAfterWhenZero(t *testing.T) {
	err := BadRequest("BAD", "nope") // RetryAfter == 0

	ctx := &fakeGinContext{}
	err.Abort(ctx)

	if _, ok := ctx.headers["Retry-After"]; ok {
		t.Error("Retry-After header should be absent when RetryAfter == 0")
	}
}

func TestErrorConstructors(t *testing.T) {
	tests := []struct {
		name       string
		err        *APIError
		wantStatus int
		wantCode   string
	}{
		{"BadRequest", BadRequest("CODE", "msg"), http.StatusBadRequest, "CODE"},
		{"Unauthorized", Unauthorized("CODE", "msg"), http.StatusUnauthorized, "CODE"},
		{"Forbidden", Forbidden("CODE", "msg"), http.StatusForbidden, "CODE"},
		{"NotFound", NotFound("CODE", "msg"), http.StatusNotFound, "CODE"},
		{"Conflict", Conflict("CODE", "msg"), http.StatusConflict, "CODE"},
		{"TooManyRequests", TooManyRequests("CODE", "msg", 60), http.StatusTooManyRequests, "CODE"},
		{"InternalError", InternalError("CODE", "msg"), http.StatusInternalServerError, "CODE"},
		{"ServiceUnavailable", ServiceUnavailable("CODE", "msg", 30), http.StatusServiceUnavailable, "CODE"},
		{"InsufficientStorage", InsufficientStorage("CODE", "msg"), http.StatusInsufficientStorage, "CODE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.HTTPStatus != tt.wantStatus {
				t.Errorf("HTTPStatus = %d, want %d", tt.err.HTTPStatus, tt.wantStatus)
			}
			if tt.err.Code != tt.wantCode {
				t.Errorf("Code = %q, want %q", tt.err.Code, tt.wantCode)
			}
		})
	}
}

func TestTooManyRequests_HasRetryAfter(t *testing.T) {
	err := TooManyRequests("RATE_LIMIT", "too many requests", 120)
	if err.RetryAfter != 120 {
		t.Errorf("RetryAfter = %d, want %d", err.RetryAfter, 120)
	}
}

func TestPrebuiltErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        *APIError
		wantCode   string
		wantStatus int
	}{
		{"ErrAuthRequired", ErrAuthRequired, CodeAuthRequired, http.StatusUnauthorized},
		{"ErrAuthInvalid", ErrAuthInvalid, CodeAuthInvalid, http.StatusUnauthorized},
		{"ErrAccessDenied", ErrAccessDenied, CodeAccessDenied, http.StatusForbidden},
		{"ErrNotAdmin", ErrNotAdmin, CodeNotAdmin, http.StatusForbidden},
		{"ErrQuotaExceeded", ErrQuotaExceeded, CodeQuotaExceeded, http.StatusInsufficientStorage},
		{"ErrResourceNotFound", ErrResourceNotFound, CodeResourceNotFound, http.StatusNotFound},
		{"ErrInternalError", ErrInternalError, CodeInternalError, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code != tt.wantCode {
				t.Errorf("Code = %q, want %q", tt.err.Code, tt.wantCode)
			}
			if tt.err.HTTPStatus != tt.wantStatus {
				t.Errorf("HTTPStatus = %d, want %d", tt.err.HTTPStatus, tt.wantStatus)
			}
		})
	}
}
