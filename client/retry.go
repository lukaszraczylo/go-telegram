package client

import (
	"bytes"
	"context"
	crand "crypto/rand"
	"encoding/binary"
	"github.com/goccy/go-json"
	"io"
	"math"
	"net/http"
	"time"
)

// RetryDoer is an HTTPDoer that retries transient failures (429, 5xx,
// and network errors) with exponential backoff. It honours the
// retry_after value Telegram supplies on rate-limit responses.
//
// Wrap any HTTPDoer to add retry behaviour:
//
//	bot := client.New(token, client.WithHTTPClient(
//	    client.NewRetryDoer(client.NewDefaultHTTPDoer())))
type RetryDoer struct {
	inner       HTTPDoer
	maxAttempts int
	base        time.Duration
	max         time.Duration
	factor      float64
	jitter      float64
}

// RetryOption configures a RetryDoer.
type RetryOption func(*RetryDoer)

// WithMaxAttempts sets the maximum number of attempts (including the
// initial one). Default 4 (one initial + three retries).
func WithMaxAttempts(n int) RetryOption {
	return func(d *RetryDoer) { d.maxAttempts = n }
}

// WithBaseBackoff sets the initial backoff duration. Default 500ms.
func WithBaseBackoff(d time.Duration) RetryOption {
	return func(r *RetryDoer) { r.base = d }
}

// WithMaxBackoff caps the backoff at max. Default 30s.
func WithMaxBackoff(d time.Duration) RetryOption {
	return func(r *RetryDoer) { r.max = d }
}

// WithBackoffFactor sets the exponential growth factor. Default 2.0.
func WithBackoffFactor(f float64) RetryOption {
	return func(r *RetryDoer) { r.factor = f }
}

// WithJitter sets the jitter fraction (0..1) applied to each backoff.
// Default 0.2.
func WithJitter(j float64) RetryOption {
	return func(r *RetryDoer) { r.jitter = j }
}

// NewRetryDoer wraps inner with retry behaviour.
func NewRetryDoer(inner HTTPDoer, opts ...RetryOption) *RetryDoer {
	d := &RetryDoer{
		inner:       inner,
		maxAttempts: 4,
		base:        500 * time.Millisecond,
		max:         30 * time.Second,
		factor:      2.0,
		jitter:      0.2,
	}
	for _, o := range opts {
		o(d)
	}
	return d
}

// Do dispatches via the inner HTTPDoer and retries on transient failures.
// The request body is buffered on first attempt so it can be replayed.
func (d *RetryDoer) Do(req *http.Request) (*http.Response, error) {
	// Buffer the body so we can replay it across attempts.
	var body []byte
	if req.Body != nil {
		b, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, &NetworkError{Err: err}
		}
		_ = req.Body.Close()
		body = b
	}

	var lastResp *http.Response
	var lastErr error

	for attempt := 1; attempt <= d.maxAttempts; attempt++ {
		if body != nil {
			req.Body = io.NopCloser(bytes.NewReader(body))
		}
		resp, err := d.inner.Do(req)

		// Network errors: maybe retry.
		if err != nil {
			// Honour ctx cancellation.
			if ctxErr := req.Context().Err(); ctxErr != nil {
				return nil, ctxErr
			}
			lastErr = err
			if attempt < d.maxAttempts {
				if !d.sleep(req.Context(), d.delay(attempt, 0)) {
					return nil, req.Context().Err()
				}
				continue
			}
			return nil, err
		}

		// HTTP 200: Telegram almost always returns 200 even for errors.
		// Peek the body to detect retryable Telegram error payloads.
		if resp.StatusCode == http.StatusOK {
			data, readErr := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if readErr != nil {
				return nil, &NetworkError{Err: readErr}
			}
			// Re-attach the buffered body for the caller.
			resp.Body = io.NopCloser(bytes.NewReader(data))

			if isRetryablePayload(data) && attempt < d.maxAttempts {
				lastResp = resp
				wait := retryAfterFromPayload(data)
				if !d.sleep(req.Context(), d.delay(attempt, wait)) {
					return nil, req.Context().Err()
				}
				continue
			}
			return resp, nil
		}

		// Non-200 status (rare with Telegram; usually 200 + ok:false).
		// Treat 5xx and 429 as retryable.
		if (resp.StatusCode == http.StatusTooManyRequests ||
			resp.StatusCode >= http.StatusInternalServerError) && attempt < d.maxAttempts {
			_ = resp.Body.Close()
			lastResp = resp
			if !d.sleep(req.Context(), d.delay(attempt, 0)) {
				return nil, req.Context().Err()
			}
			continue
		}
		return resp, nil
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return lastResp, nil
}

// delay computes the wait duration for the given attempt (1-based).
// override, when non-zero, takes precedence (used to honour Telegram's
// retry_after value).
func (d *RetryDoer) delay(attempt int, override time.Duration) time.Duration {
	if override > 0 {
		return override
	}
	delay := float64(d.base) * math.Pow(d.factor, float64(attempt-1))
	if d.jitter > 0 {
		var b [8]byte
		_, _ = crand.Read(b[:])
		f := float64(binary.LittleEndian.Uint64(b[:])) / (1 << 64)
		delay *= 1 + (f*2-1)*d.jitter
	}
	if delay > float64(d.max) {
		delay = float64(d.max)
	}
	if delay < 0 {
		delay = 0
	}
	return time.Duration(delay)
}

// sleep waits for dur or ctx cancellation. Returns false if cancelled.
func (d *RetryDoer) sleep(ctx context.Context, dur time.Duration) bool {
	if dur <= 0 {
		return true
	}
	t := time.NewTimer(dur)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-ctx.Done():
		return false
	}
}

// isRetryablePayload reports whether body is a Telegram error response
// indicating a retryable failure (429 or 5xx error_code).
func isRetryablePayload(body []byte) bool {
	var env struct {
		OK        bool `json:"ok"`
		ErrorCode int  `json:"error_code"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return false
	}
	if env.OK {
		return false
	}
	return env.ErrorCode == 429 || (env.ErrorCode >= 500 && env.ErrorCode < 600)
}

// retryAfterFromPayload extracts the retry_after value from a Telegram
// error response body and returns it as a duration. Returns 0 if absent.
func retryAfterFromPayload(body []byte) time.Duration {
	var env struct {
		Parameters struct {
			RetryAfter int `json:"retry_after"`
		} `json:"parameters"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return 0
	}
	return time.Duration(env.Parameters.RetryAfter) * time.Second
}
