package client

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type retryMockDoer struct{ mock.Mock }

func (m *retryMockDoer) Do(r *http.Request) (*http.Response, error) {
	args := m.Called(r)
	if v := args.Get(0); v != nil {
		return v.(*http.Response), args.Error(1)
	}
	return nil, args.Error(1)
}

func okResp(body string) *http.Response {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}

func TestRetryDoer_HappyPath(t *testing.T) {
	m := &retryMockDoer{}
	m.On("Do", mock.Anything).Return(okResp(`{"ok":true,"result":"hi"}`), nil).Once()

	d := NewRetryDoer(m)
	req, _ := http.NewRequest("POST", "http://x", strings.NewReader(`{}`))
	resp, err := d.Do(req)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	m.AssertExpectations(t)
}

func TestRetryDoer_RetriesOnNetworkError(t *testing.T) {
	m := &retryMockDoer{}
	m.On("Do", mock.Anything).Return(nil, errors.New("dial timeout")).Once()
	m.On("Do", mock.Anything).Return(okResp(`{"ok":true,"result":"hi"}`), nil).Once()

	d := NewRetryDoer(m, WithBaseBackoff(time.Millisecond))
	req, _ := http.NewRequest("POST", "http://x", strings.NewReader(`{}`))
	resp, err := d.Do(req)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	m.AssertExpectations(t)
}

func TestRetryDoer_HonoursRetryAfter(t *testing.T) {
	m := &retryMockDoer{}
	m.On("Do", mock.Anything).Return(
		okResp(`{"ok":false,"error_code":429,"description":"Too Many","parameters":{"retry_after":1}}`), nil).Once()
	m.On("Do", mock.Anything).Return(okResp(`{"ok":true,"result":1}`), nil).Once()

	// base is 10s — retry_after=1s should override it (much shorter wait).
	d := NewRetryDoer(m, WithBaseBackoff(10*time.Second))
	req, _ := http.NewRequest("POST", "http://x", strings.NewReader(`{}`))
	start := time.Now()
	resp, err := d.Do(req)
	elapsed := time.Since(start)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	require.GreaterOrEqual(t, elapsed, 900*time.Millisecond, "should honour retry_after=1s")
	require.Less(t, elapsed, 3*time.Second, "should NOT use base backoff (10s)")
	m.AssertExpectations(t)
}

func TestRetryDoer_Retries5xx(t *testing.T) {
	m := &retryMockDoer{}
	m.On("Do", mock.Anything).Return(
		okResp(`{"ok":false,"error_code":500,"description":"Internal Server Error"}`), nil).Once()
	m.On("Do", mock.Anything).Return(okResp(`{"ok":true,"result":1}`), nil).Once()

	d := NewRetryDoer(m, WithBaseBackoff(time.Millisecond))
	req, _ := http.NewRequest("POST", "http://x", strings.NewReader(`{}`))
	resp, err := d.Do(req)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	m.AssertExpectations(t)
}

func TestRetryDoer_AllAttemptsFail(t *testing.T) {
	m := &retryMockDoer{}
	m.On("Do", mock.Anything).Return(nil, errors.New("dial timeout"))

	d := NewRetryDoer(m, WithMaxAttempts(3), WithBaseBackoff(time.Millisecond))
	req, _ := http.NewRequest("POST", "http://x", strings.NewReader(`{}`))
	_, err := d.Do(req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "dial timeout")
	require.Equal(t, 3, len(m.Calls))
}

func TestRetryDoer_ContextCancellationAborts(t *testing.T) {
	m := &retryMockDoer{}
	m.On("Do", mock.Anything).Return(
		okResp(`{"ok":false,"error_code":500,"description":"server error"}`), nil).Maybe()

	d := NewRetryDoer(m, WithBaseBackoff(100*time.Millisecond))
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "POST", "http://x", strings.NewReader(`{}`))
	_, err := d.Do(req)
	require.Error(t, err)
	require.True(t, errors.Is(err, context.DeadlineExceeded))
}

func TestRetryDoer_ReplaysBody(t *testing.T) {
	m := &retryMockDoer{}
	var seen []string

	// First call: capture body, return 500 to trigger retry.
	m.On("Do", mock.Anything).Return(okResp(`{"ok":false,"error_code":500}`), nil).Once().Run(func(args mock.Arguments) {
		r := args.Get(0).(*http.Request)
		body, _ := io.ReadAll(r.Body)
		seen = append(seen, string(body))
	})
	// Second call: capture body, return success.
	m.On("Do", mock.Anything).Return(okResp(`{"ok":true}`), nil).Once().Run(func(args mock.Arguments) {
		r := args.Get(0).(*http.Request)
		body, _ := io.ReadAll(r.Body)
		seen = append(seen, string(body))
	})

	d := NewRetryDoer(m, WithBaseBackoff(time.Millisecond))
	req, _ := http.NewRequest("POST", "http://x", strings.NewReader(`{"chat_id":42}`))
	_, err := d.Do(req)
	require.NoError(t, err)
	require.Len(t, seen, 2)
	require.Equal(t, seen[0], seen[1])
	require.Equal(t, `{"chat_id":42}`, seen[0])
	m.AssertExpectations(t)
}
