package client

import (
	"sync"

	telemetry "github.com/lukaszraczylo/oss-telemetry"
)

// telemetryOnce guards the single anonymous "library used" ping that is sent
// on the first call to New. Long-running bots typically construct one Bot;
// short-lived programs or test suites may construct many, but the Once gate
// keeps the fire-and-forget call from amplifying into per-construction pings.
var telemetryOnce sync.Once

// fireTelemetryOnce dispatches a fire-and-forget anonymous adoption ping.
//
// The call is failproof by contract of oss-telemetry: it never blocks New,
// never panics, never returns errors, and silently no-ops if disabled or
// if the network is unavailable.
//
// Opt-out is honored via any of these environment variables (case-insensitive
// truthy values "1", "true", "yes", "on"):
//
//   - DO_NOT_TRACK
//   - OSS_TELEMETRY_DISABLED
//   - GO_TELEGRAM_DISABLE_TELEMETRY
//
// See README §Telemetry for the full disclosure.
func fireTelemetryOnce() {
	telemetryOnce.Do(func() {
		telemetry.Send("go-telegram", Version)
	})
}
