package transport

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/lukaszraczylo/go-telegram/api"
	"github.com/lukaszraczylo/go-telegram/client"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// LongPoller — unauthorized error causes immediate return
// ---------------------------------------------------------------------------

func TestLongPoller_UnauthorizedExits(t *testing.T) {
	m := &mockDoer{}
	m.On("Do", mock.Anything).Return(
		resp(`{"ok":false,"error_code":401,"description":"Unauthorized"}`), nil,
	).Once()

	b := client.New("bad-token", client.WithHTTPClient(m))
	p := NewLongPoller(b)
	p.Timeout = 0

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := p.Run(ctx)
	require.Error(t, err)
	require.True(t, errors.Is(err, client.ErrUnauthorized), "expected unauthorized: %v", err)
}

// ---------------------------------------------------------------------------
// LongPoller — ctx cancelled while waiting for retry backoff
// ---------------------------------------------------------------------------

func TestLongPoller_CtxCancelledDuringBackoff(t *testing.T) {
	m := &mockDoer{}
	var callCount int
	m.On("Do", mock.Anything).Run(func(args mock.Arguments) {
		callCount++
	}).Return(nil, errors.New("network error")).Maybe()

	b := client.New("t", client.WithHTTPClient(m))
	p := NewLongPoller(b)
	p.Timeout = 0
	// Long backoff ensures ctx cancels before retry fires.
	p.Backoff = &ExponentialBackoff{Base: 5 * time.Second, Max: 5 * time.Second, Factor: 1}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := p.Run(ctx)
	require.Error(t, err)
	// Should fail fast, not wait the full 5s backoff.
	require.LessOrEqual(t, callCount, 3)
}

// ---------------------------------------------------------------------------
// LongPoller — AllowedTypes field
// ---------------------------------------------------------------------------

func TestLongPoller_AllowedTypes(t *testing.T) {
	m := &mockDoer{}
	var seenBody string
	m.On("Do", mock.MatchedBy(func(r *http.Request) bool {
		b, _ := io.ReadAll(r.Body)
		seenBody = string(b)
		return true
	})).Return(resp(`{"ok":true,"result":[]}`), nil).Once()
	m.On("Do", mock.Anything).Return(resp(`{"ok":true,"result":[]}`), nil).Maybe()

	b := client.New("t", client.WithHTTPClient(m))
	p := NewLongPoller(b)
	p.Timeout = 0
	p.AllowedTypes = []api.UpdateType{"message", "callback_query"}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_ = p.Run(ctx)

	require.Contains(t, seenBody, "allowed_updates")
}

// ---------------------------------------------------------------------------
// WebhookServer — ListenAndServe error (bind on in-use port)
// ---------------------------------------------------------------------------

func TestWebhookServer_ListenAndServeError(t *testing.T) {
	// Bind a port to block the webhook server.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = l.Close() })
	addr := l.Addr().String()

	b := client.New("t")
	w := NewWebhookServer(b)

	ctx := context.Background()
	err = w.ListenAndServe(ctx, addr)
	require.Error(t, err, "should error when port is in use")
	require.False(t, errors.Is(err, http.ErrServerClosed))
}

// ---------------------------------------------------------------------------
// WebhookServer — body too large (> 1 MiB)
// ---------------------------------------------------------------------------

func TestWebhookServer_BodyTooLarge(t *testing.T) {
	b := client.New("t")
	w := NewWebhookServer(b)

	// Construct a body slightly over 1 MiB.
	bigBody := bytes.Repeat([]byte("x"), 1<<20+1)
	req, _ := http.NewRequest(http.MethodPost, "/", bytes.NewReader(bigBody))
	rw := newTestResponseWriter()
	w.ServeHTTP(rw, req)
	require.Equal(t, http.StatusRequestEntityTooLarge, rw.code)
}

// ---------------------------------------------------------------------------
// WebhookServer — Stop when srv is nil (no ListenAndServe called)
// ---------------------------------------------------------------------------

func TestWebhookServer_StopNoServer(t *testing.T) {
	b := client.New("t")
	w := NewWebhookServer(b)
	require.NoError(t, w.Stop(context.Background()))
}

// ---------------------------------------------------------------------------
// WebhookServer — no secret token, any POST accepted
// ---------------------------------------------------------------------------

func TestWebhookServer_NoSecretAllowsAnyPost(t *testing.T) {
	b := client.New("t")
	w := NewWebhookServer(b)

	body := `{"update_id":99}`
	req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	// No secret header set.
	rw := newTestResponseWriter()
	// ServeHTTP would block on w.out <- u unless we drain it.
	go func() {
		for range w.Updates() {
		}
	}()
	w.ServeHTTP(rw, req)
	// Bad JSON would return 400; update_id-only body is valid enough for Update.
	require.NotEqual(t, http.StatusUnauthorized, rw.code)
}

// ---------------------------------------------------------------------------
// ExponentialBackoff — negative attempt clamped to 1
// ---------------------------------------------------------------------------

func TestExponentialBackoff_NegativeAttempt(t *testing.T) {
	b := DefaultBackoff()
	d := b.NextDelay(-5)
	require.GreaterOrEqual(t, d, time.Duration(0))
	require.LessOrEqual(t, d, b.Max)
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

type testResponseWriter struct {
	code   int
	header http.Header
}

func newTestResponseWriter() *testResponseWriter {
	return &testResponseWriter{code: http.StatusOK, header: http.Header{}}
}

func (r *testResponseWriter) Header() http.Header         { return r.header }
func (r *testResponseWriter) Write(b []byte) (int, error) { return len(b), nil }
func (r *testResponseWriter) WriteHeader(statusCode int)  { r.code = statusCode }
