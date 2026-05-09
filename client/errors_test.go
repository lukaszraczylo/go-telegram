package client

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAPIError_FieldsAndMethods(t *testing.T) {
	e := &APIError{
		Code:        429,
		Description: "Too Many Requests: retry after 5",
		Parameters:  &ResponseParameters{RetryAfter: 5},
	}
	require.Equal(t, "telegram: 429 Too Many Requests: retry after 5", e.Error())
	require.True(t, e.IsRetryable())
	require.Equal(t, 5*time.Second, e.RetryAfter())
}

func TestAPIError_Sentinels(t *testing.T) {
	cases := []struct {
		code     int
		desc     string
		sentinel error
	}{
		{401, "Unauthorized", ErrUnauthorized},
		{400, "Bad Request: chat not found", ErrChatNotFound},
		{400, "Bad Request: message is not modified", ErrMessageNotModified},
		{429, "Too Many Requests: retry after 1", ErrTooManyRequests},
		{400, "Bad Request: user not found", ErrUserNotFound},
		{400, "Bad Request: message to delete not found", ErrMessageNotFound},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			e := mapAPIError(c.code, c.desc, nil)
			require.True(t, errors.Is(e, c.sentinel), "expected %v to wrap %v", e, c.sentinel)
		})
	}
}

func TestAPIError_IsRetryable(t *testing.T) {
	require.True(t, (&APIError{Code: 500}).IsRetryable())
	require.True(t, (&APIError{Code: 502}).IsRetryable())
	require.True(t, (&APIError{Code: 429}).IsRetryable())
	require.False(t, (&APIError{Code: 400}).IsRetryable())
	require.False(t, (&APIError{Code: 401}).IsRetryable())
}

func TestNetworkAndParseErrorWrapping(t *testing.T) {
	inner := errors.New("dial tcp: timeout")
	ne := &NetworkError{Err: inner}
	require.ErrorIs(t, ne, inner)

	pe := &ParseError{Err: errors.New("unexpected EOF"), Body: []byte("garbage")}
	require.Contains(t, pe.Error(), "garbage")
}
