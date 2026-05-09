package transport

import (
	"math"
	"math/rand/v2"
	"time"
)

// BackoffStrategy returns the duration to wait before the next attempt
// after `attempt` consecutive failures (1-based). Implementations must
// be safe to call from a single goroutine.
type BackoffStrategy interface {
	NextDelay(attempt int) time.Duration
}

// ExponentialBackoff implements capped exponential back-off with jitter.
// Defaults: Base=500ms, Max=30s, Factor=2.0, Jitter=0.2.
type ExponentialBackoff struct {
	Base   time.Duration
	Max    time.Duration
	Factor float64
	Jitter float64 // 0..1; fraction of computed delay added/subtracted at random
}

// DefaultBackoff returns an ExponentialBackoff with library defaults.
func DefaultBackoff() *ExponentialBackoff {
	return &ExponentialBackoff{
		Base:   500 * time.Millisecond,
		Max:    30 * time.Second,
		Factor: 2.0,
		Jitter: 0.2,
	}
}

// NextDelay implements BackoffStrategy.
func (b *ExponentialBackoff) NextDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	d := float64(b.Base) * math.Pow(b.Factor, float64(attempt-1))
	if b.Jitter > 0 {
		d *= 1 + (rand.Float64()*2-1)*b.Jitter
	}
	if d > float64(b.Max) {
		d = float64(b.Max)
	}
	if d < 0 {
		d = 0
	}
	return time.Duration(d)
}
