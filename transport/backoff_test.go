package transport

import (
	"testing"
	"time"
)

// TestExponentialBackoff_MaxCapAfterJitter verifies that the Max cap is applied
// after jitter so no delay can exceed Max regardless of jitter magnitude.
func TestExponentialBackoff_MaxCapAfterJitter(t *testing.T) {
	b := &ExponentialBackoff{
		Base:   10 * time.Second,
		Max:    20 * time.Second,
		Factor: 2.0,
		Jitter: 0.5,
	}

	// Run many times to account for randomness.
	for i := 0; i < 10_000; i++ {
		d := b.NextDelay(10)
		if d > b.Max {
			t.Fatalf("attempt 10: got %v, want ≤ %v (jitter exceeded Max cap)", d, b.Max)
		}
		if d < 0 {
			t.Fatalf("attempt 10: got negative delay %v", d)
		}
	}
}

// TestExponentialBackoff_ZeroAttemptClamped ensures attempt < 1 is treated as 1.
func TestExponentialBackoff_ZeroAttemptClamped(t *testing.T) {
	b := DefaultBackoff()
	d0 := b.NextDelay(0)
	d1 := b.NextDelay(1)
	// Both should be in the same ballpark (Base ± Jitter*Base).
	maxBase := float64(b.Base) * (1 + b.Jitter)
	if float64(d0) > maxBase || float64(d1) > maxBase {
		t.Fatalf("unexpected delay: d0=%v d1=%v maxBase=%v", d0, d1, time.Duration(maxBase))
	}
}
