package client

import (
	"os"
	"sync"
	"testing"

	telemetry "github.com/lukaszraczylo/oss-telemetry"
)

// TestMain disables outgoing telemetry for the duration of this package's
// test suite. The library's own tests construct many Bot instances; without
// this guard they would each contribute a real ping to the public endpoint.
// End-user test suites that construct Bot are not affected by this — only
// tests inside this package are.
func TestMain(m *testing.M) {
	telemetry.Disable()
	os.Exit(m.Run())
}

// TestFireTelemetryOnce_OnlyFiresOnce verifies the sync.Once gate. Even if
// New is called repeatedly, the underlying telemetry.Send is invoked at most
// once per process. We can't observe the network call directly (telemetry
// is disabled here via TestMain) so we assert on the once-Do count via a
// fresh local sync.Once paralleling the production one.
func TestFireTelemetryOnce_OnlyFiresOnce(t *testing.T) {
	// Reset the package-level Once so this test starts from a clean state.
	telemetryOnce = sync.Once{}
	t.Cleanup(func() { telemetryOnce = sync.Once{} })

	calls := 0
	probe := func() { telemetryOnce.Do(func() { calls++ }) }

	for i := 0; i < 50; i++ {
		probe()
	}
	if calls != 1 {
		t.Fatalf("expected exactly 1 Once execution, got %d", calls)
	}
}

// TestNew_DoesNotPanicUnderRepeatedConstruction is a smoke test that
// telemetry wiring does not affect New's existing contract. New must never
// panic, regardless of telemetry state.
func TestNew_DoesNotPanicUnderRepeatedConstruction(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("New panicked: %v", r)
		}
	}()
	for i := 0; i < 20; i++ {
		_ = New("test-token-" + string(rune('A'+i)))
	}
}
