package transport

import (
	"context"

	"github.com/lukaszraczylo/go-telegram/api"
)

// Updater is the abstraction over update sources. Implementations must:
//   - return a channel from Updates() that receives every Update they read.
//   - close the channel after Run returns.
//   - honour ctx cancellation in Run.
type Updater interface {
	// Updates returns the channel updates flow into. Multiple readers
	// is implementation-defined; users should treat it as single-reader.
	Updates() <-chan api.Update
	// Run blocks until ctx is cancelled or a fatal error occurs. It is
	// the user's responsibility to call Run in a goroutine if needed.
	Run(ctx context.Context) error
	// Stop signals Run to exit and waits for the channel to drain.
	// Implementations must be idempotent.
	Stop(ctx context.Context) error
}
