package conversation

import (
	"context"
	"errors"
)

// ErrKeyNotFound is returned by Storage.Get when no conversation is active
// for the given key.
var ErrKeyNotFound = errors.New("conversation: key not found")

// Storage persists per-user (or per-chat, per-message — depending on the
// KeyStrategy in use) conversation state across update deliveries.
//
// Implementations must be safe for concurrent use.
type Storage interface {
	Get(ctx context.Context, key string) (State, error)
	Set(ctx context.Context, key string, state State) error
	Delete(ctx context.Context, key string) error
}
