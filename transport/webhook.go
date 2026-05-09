// Package transport provides update delivery mechanisms (long-poll and
// webhook) that feed updates into the dispatch package's Router.
//
// All implementations satisfy the Updater interface so user code can
// swap one for the other without touching handler logic.
package transport

import (
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
	defer func() { _ = r.Body.Close() }()

	const max = 1 << 20 // 1 MiB cap on body
	buf := make([]byte, 0, 1024)
	tmp := make([]byte, 4096)
	for {
		n, err := r.Body.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
			if len(buf) > max {
				rw.WriteHeader(http.StatusRequestEntityTooLarge)
				return
			}
		}
		if errors.Is(err, http.ErrBodyReadAfterClose) || err != nil {
			break
		}
	}

	var u api.Update
	codec := w.Bot.Codec()
	if err := codec.Unmarshal(buf, &u); err != nil {
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
