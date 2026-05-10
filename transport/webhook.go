// Package transport provides update delivery mechanisms (long-poll and
// webhook) that feed updates into the dispatch package's Router.
//
// All implementations satisfy the Updater interface so user code can
// swap one for the other without touching handler logic.
package transport

import (
	"bytes"
	"context"
	"crypto/subtle"
	"errors"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/lukaszraczylo/go-telegram/api"
	"github.com/lukaszraczylo/go-telegram/client"
)

// webhookBufPool reuses *bytes.Buffer for incoming webhook bodies.
// Webhook payloads are typically a single Telegram Update (commonly
// 1-8 KiB), so a buffer that has grown once will satisfy most
// subsequent requests with no additional allocation.
var webhookBufPool = sync.Pool{New: func() any { return new(bytes.Buffer) }}

// maxWebhookBufCap caps the buffer size returned to webhookBufPool so
// a rare oversized update doesn't permanently bloat the pool. Buffers
// larger than this are dropped on the floor.
const maxWebhookBufCap = 256 * 1024

func putWebhookBuf(buf *bytes.Buffer) {
	if buf.Cap() > maxWebhookBufCap {
		return
	}
	webhookBufPool.Put(buf)
}

// WebhookServer implements Updater by exposing an http.Handler that
// receives updates from Telegram. It can be mounted on the user's own
// HTTP server (via ServeHTTP) or run standalone (via ListenAndServe).
type WebhookServer struct {
	Bot         *client.Bot
	SecretToken string // verify X-Telegram-Bot-Api-Secret-Token; empty disables

	out      chan api.Update
	once     sync.Once
	stop     chan struct{}
	mu       sync.Mutex
	handlers sync.WaitGroup

	srv *http.Server
}

// WebhookOption configures a WebhookServer at construction time.
type WebhookOption func(*webhookOptions)

type webhookOptions struct {
	bufferSize int
}

// WithBufferSize sets the size of the updates channel buffer.
// Default is 64.
func WithBufferSize(n int) WebhookOption {
	return func(o *webhookOptions) { o.bufferSize = n }
}

// NewWebhookServer constructs a WebhookServer with default buffer size (64).
// Use WithBufferSize to override.
func NewWebhookServer(b *client.Bot, opts ...WebhookOption) *WebhookServer {
	cfg := webhookOptions{bufferSize: 64}
	for _, o := range opts {
		o(&cfg)
	}
	return &WebhookServer{
		Bot:  b,
		out:  make(chan api.Update, cfg.bufferSize),
		stop: make(chan struct{}),
	}
}

// Updates implements Updater.
func (w *WebhookServer) Updates() <-chan api.Update { return w.out }

// Run implements Updater. It blocks until Stop is called or ctx is
// cancelled. If the server has not been started via ListenAndServe, Run
// only watches for shutdown — the user is expected to mount ServeHTTP
// on their own router.
func (w *WebhookServer) Run(ctx context.Context) error {
	defer close(w.out)
	defer w.handlers.Wait() // drain in-flight ServeHTTP calls before closing out (LIFO: runs first)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-w.stop:
		return nil
	}
}

// Stop implements Updater.
func (w *WebhookServer) Stop(ctx context.Context) error {
	w.once.Do(func() { close(w.stop) })
	w.mu.Lock()
	srv := w.srv
	w.mu.Unlock()
	if srv != nil {
		return srv.Shutdown(ctx)
	}
	return nil
}

// ServeHTTP implements http.Handler. Telegram POSTs each update as JSON
// to this endpoint. Non-POST requests get 405; bad bodies get 400; secret
// token mismatches get 401.
func (w *WebhookServer) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		rw.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if w.SecretToken != "" {
		got := r.Header.Get("X-Telegram-Bot-Api-Secret-Token")
		if subtle.ConstantTimeCompare([]byte(got), []byte(w.SecretToken)) != 1 {
			rw.WriteHeader(http.StatusUnauthorized)
			return
		}
	}
	w.handlers.Add(1)
	defer w.handlers.Done()

	const maxBody = 1 << 20 // 1 MiB cap on body
	r.Body = http.MaxBytesReader(rw, r.Body, maxBody)
	defer func() { _ = r.Body.Close() }()

	buf := webhookBufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer putWebhookBuf(buf)
	if _, err := buf.ReadFrom(r.Body); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			rw.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	var u api.Update
	codec := w.Bot.Codec()
	if err := codec.Unmarshal(buf.Bytes(), &u); err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	select {
	case w.out <- u:
	case <-w.stop:
	}

	rw.WriteHeader(http.StatusOK)
}

// ListenAndServe starts an HTTP server on addr and blocks until Stop is
// called (which triggers Shutdown with the caller's context) or the server
// returns an error other than http.ErrServerClosed. Callers must invoke
// Stop(ctx) to cleanly shut down the server; the ctx passed here is only
// used as the server's base context for incoming requests.
func (w *WebhookServer) ListenAndServe(ctx context.Context, addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/", w)
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		BaseContext:       func(net.Listener) context.Context { return ctx },
		ReadHeaderTimeout: 10 * time.Second,
	}
	w.mu.Lock()
	w.srv = srv
	w.mu.Unlock()
	err := srv.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
