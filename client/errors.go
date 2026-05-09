package client

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// APIError represents a non-OK Telegram Bot API response.
// It satisfies error and unwraps to a sentinel (ErrUnauthorized, etc.)
// where the description matches a known prefix, enabling errors.Is checks.
type APIError struct {
	Code        int
	Description string
	Parameters  *ResponseParameters

	// sentinel, if non-nil, is the wrapped sentinel error returned by
	// Unwrap. It is set by mapAPIError based on Code+Description.
	sentinel error
}

// Error implements error.
func (e *APIError) Error() string {
	return fmt.Sprintf("telegram: %d %s", e.Code, e.Description)
}

// Unwrap returns the matched sentinel error, if any.
func (e *APIError) Unwrap() error { return e.sentinel }

// IsRetryable returns true for transient HTTP statuses (429, 5xx).
func (e *APIError) IsRetryable() bool {
	return e.Code == 429 || (e.Code >= 500 && e.Code < 600)
}

// RetryAfter returns the recommended back-off duration. It honours the
// Telegram-supplied retry_after parameter; if absent, returns 0.
func (e *APIError) RetryAfter() time.Duration {
	if e.Parameters == nil {
		return 0
	}
	return time.Duration(e.Parameters.RetryAfter) * time.Second
}

// NetworkError wraps a transport-level failure (DNS, TCP, TLS, timeout
// short of an HTTP response).
type NetworkError struct{ Err error }

func (e *NetworkError) Error() string { return "telegram: network: " + redactToken(e.Err.Error()) }

func (e *NetworkError) Unwrap() error { return e.Err }

// ParseError wraps a JSON decode failure on a response body. Body is
// retained (truncated to 4 KiB); Error() displays up to 256 bytes for diagnostics.
type ParseError struct {
	Err  error
	Body []byte
}

func (e *ParseError) Error() string {
	body := e.Body
	if len(body) > 256 {
		body = body[:256]
	}
	return fmt.Sprintf("telegram: parse: %s (body=%q)", redactToken(e.Err.Error()), body)
}

func (e *ParseError) Unwrap() error { return e.Err }

// Sentinel errors returned via APIError.Unwrap when the description matches.
// Compare with errors.Is.
var (
	ErrUnauthorized       = errors.New("telegram: unauthorized")
	ErrChatNotFound       = errors.New("telegram: chat not found")
	ErrMessageNotModified = errors.New("telegram: message is not modified")
	ErrTooManyRequests    = errors.New("telegram: too many requests")
	ErrBadRequest         = errors.New("telegram: bad request")
	ErrForbidden          = errors.New("telegram: forbidden")
	ErrUserNotFound       = errors.New("telegram: user not found")
	ErrMessageNotFound    = errors.New("telegram: message not found")
)

// mapAPIError builds an *APIError and attaches the appropriate sentinel
// based on Code+Description. It is the single point where wire-level
// failures are translated into the Go error taxonomy.
func mapAPIError(code int, description string, params *ResponseParameters) *APIError {
	e := &APIError{Code: code, Description: description, Parameters: params}
	switch {
	case code == 401:
		e.sentinel = ErrUnauthorized
	case code == 403:
		e.sentinel = ErrForbidden
	case code == 429:
		e.sentinel = ErrTooManyRequests
	case code == 400 && strings.Contains(description, "user not found"):
		e.sentinel = ErrUserNotFound
	case code == 400 && strings.Contains(description, "message to") && strings.Contains(description, "not found"):
		e.sentinel = ErrMessageNotFound
	case code == 400 && strings.Contains(description, "chat not found"):
		e.sentinel = ErrChatNotFound
	case code == 400 && strings.Contains(description, "message is not modified"):
		e.sentinel = ErrMessageNotModified
	case code == 400:
		e.sentinel = ErrBadRequest
	}
	return e
}
