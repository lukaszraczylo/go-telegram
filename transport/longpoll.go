package transport

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/lukaszraczylo/go-telegram/api"
	"github.com/lukaszraczylo/go-telegram/client"
)

// LongPoller pulls updates via Bot.GetUpdates in a loop, advancing the
// offset cursor after each batch. It applies BackoffStrategy on transient
// errors (network failures, 5xx, 429).
//
// At-least-once semantics on shutdown: when ctx is cancelled or Stop is
// called mid-batch, any updates already fetched but not yet dispatched are
// dropped without advancing the offset. On the next restart those updates
// will be re-delivered by Telegram.
type LongPoller struct {
	Bot          *client.Bot
	Timeout      int // seconds, default 30
	Limit        int // 1..100, default 100
	AllowedTypes []api.UpdateType
	Backoff      BackoffStrategy

	out  chan api.Update
	once sync.Once
	stop chan struct{}
}

// NewLongPoller constructs a LongPoller with sensible defaults.
func NewLongPoller(b *client.Bot) *LongPoller {
	return &LongPoller{
		Bot:     b,
		Timeout: 30,
		Limit:   100,
		Backoff: DefaultBackoff(),
		out:     make(chan api.Update, 64),
		stop:    make(chan struct{}),
	}
}

// Updates implements Updater.
func (p *LongPoller) Updates() <-chan api.Update { return p.out }

// Run implements Updater. It blocks until ctx is cancelled, Stop is
// called, or a fatal error occurs (e.g. unauthorized). See LongPoller
// for at-least-once delivery semantics on shutdown.
func (p *LongPoller) Run(ctx context.Context) error {
	defer close(p.out)

	var offset int64
	failures := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-p.stop:
			return nil
		default:
		}

		params := &api.GetUpdatesParams{Offset: &offset}
		if p.Limit > 0 {
			lim := int64(p.Limit)
			params.Limit = &lim
		}
		if p.Timeout > 0 {
			to := int64(p.Timeout)
			params.Timeout = &to
		}
		if len(p.AllowedTypes) > 0 {
			allowed := make([]string, len(p.AllowedTypes))
			for i, t := range p.AllowedTypes {
				allowed[i] = string(t)
			}
			params.AllowedUpdates = allowed
		}
		ups, err := api.GetUpdates(ctx, p.Bot, params)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return err
			}
			// Fatal: unauthorized -> bail.
			if errors.Is(err, client.ErrUnauthorized) {
				return err
			}
			var ae *client.APIError
			var delay time.Duration
			if errors.As(err, &ae) && ae.RetryAfter() > 0 {
				delay = ae.RetryAfter()
				// Don't escalate failures count — Telegram is dictating the wait.
			} else {
				failures++
				delay = p.Backoff.NextDelay(failures)
			}
			select {
			case <-time.After(delay):
				continue
			case <-ctx.Done():
				return ctx.Err()
			case <-p.stop:
				return nil
			}
		}
		failures = 0

		for _, u := range ups {
			select {
			case p.out <- u:
				if u.UpdateID >= offset {
					offset = u.UpdateID + 1
				}
			case <-ctx.Done():
				return ctx.Err()
			case <-p.stop:
				return nil
			}
		}
	}
}

// Stop implements Updater.
func (p *LongPoller) Stop(ctx context.Context) error {
	p.once.Do(func() { close(p.stop) })
	return nil
}
