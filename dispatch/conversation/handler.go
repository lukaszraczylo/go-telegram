package conversation

import (
	"context"
	"errors"

	"github.com/lukaszraczylo/go-telegram/api"
	"github.com/lukaszraczylo/go-telegram/dispatch"
)

// stateTransition is a sentinel error type carrying a state transition
// or end signal. Conversation handlers return one of these (via Next or
// End helpers below) to drive the state machine.
type stateTransition struct {
	next State
	end  bool
}

func (e *stateTransition) Error() string {
	if e.end {
		return "conversation: end"
	}
	return "conversation: → " + string(e.next)
}

// Next signals the conversation should advance to the given state.
// Conversation handlers return Next("state_name") to transition.
func Next(s State) error {
	return &stateTransition{next: s}
}

// End signals the conversation has finished and state should be cleared.
// Conversation handlers return End() to terminate.
func End() error {
	return &stateTransition{end: true}
}

// Handler defines a step in the conversation. Receives the dispatch context
// and the raw update. Returns:
//   - nil to stay in the current state
//   - Next("state") to transition to a different state
//   - End() to end the conversation
//   - any other non-nil error to surface to the dispatcher (state unchanged)
type Handler func(ctx *dispatch.Context, u *api.Update) error

// Step pairs a filter with a handler for one conversation step.
type Step struct {
	Filter  dispatch.Filter[*api.Update]
	Handler Handler
}

// Conversation is a stateful handler with entry, per-state, exit and
// fallback steps. A conversation is keyed by KeyStrategy (default
// KeyByUserAndChat) and persisted by Storage (default in-memory).
type Conversation struct {
	// EntryPoints starts a new conversation when a matching filter fires
	// and no conversation is already active for the key.
	EntryPoints []Step

	// States maps each state to the steps that handle it.
	States map[State][]Step

	// Exits, if any match, end the active conversation early. Useful for
	// /cancel-style commands.
	Exits []Step

	// Fallbacks run when no state step matches the current update.
	Fallbacks []Step

	// Storage persists conversation state. Defaults to NewMemoryStorage.
	Storage Storage

	// KeyStrategy derives the persistence key. Defaults to KeyByUserAndChat.
	KeyStrategy KeyStrategy

	// AllowReEntry, when true, lets entry-point steps fire even while a
	// conversation is already active for the key (effectively restarting it).
	AllowReEntry bool
}

// Dispatch is a global middleware-shaped Handler that consumes updates
// and routes them through the conversation graph. Register via
// router.Use(conv.Dispatch).
//
// If the conversation claims an update, downstream handlers are skipped.
// If the conversation does not claim it, downstream handlers run as normal.
func (c *Conversation) Dispatch(next dispatch.Handler[*api.Update]) dispatch.Handler[*api.Update] {
	if c.Storage == nil {
		c.Storage = NewMemoryStorage()
	}
	if c.KeyStrategy == nil {
		c.KeyStrategy = KeyByUserAndChat
	}
	return func(dctx *dispatch.Context, u *api.Update) error {
		key := c.KeyStrategy(u)
		if key == "" {
			return next(dctx, u)
		}

		ctx := dctx.Ctx
		current, err := c.Storage.Get(ctx, key)
		if err != nil && !errors.Is(err, ErrKeyNotFound) {
			return err
		}
		active := !errors.Is(err, ErrKeyNotFound)

		// Try exits first (always allowed if conversation is active).
		if active {
			for _, step := range c.Exits {
				if step.Filter(u) {
					if err := c.runStep(ctx, dctx, u, key, step.Handler); err != nil {
						return err
					}
					return nil
				}
			}
		}

		// Try entry points (only if no active conversation, or AllowReEntry).
		if !active || c.AllowReEntry {
			for _, step := range c.EntryPoints {
				if step.Filter(u) {
					if err := c.runStep(ctx, dctx, u, key, step.Handler); err != nil {
						return err
					}
					return nil
				}
			}
		}

		if !active {
			return next(dctx, u)
		}

		// Active conversation: try state steps.
		for _, step := range c.States[current] {
			if step.Filter(u) {
				if err := c.runStep(ctx, dctx, u, key, step.Handler); err != nil {
					return err
				}
				return nil
			}
		}

		// Fallbacks if no state step matched.
		for _, step := range c.Fallbacks {
			if step.Filter(u) {
				if err := c.runStep(ctx, dctx, u, key, step.Handler); err != nil {
					return err
				}
				return nil
			}
		}

		// Active conversation but no step matched and no fallback: swallow the
		// update (do NOT pass to downstream handlers, since the user is
		// mid-conversation and an unrelated handler would surprise them).
		return nil
	}
}

// runStep invokes the handler and applies its return-value state transition.
func (c *Conversation) runStep(ctx context.Context, dctx *dispatch.Context, u *api.Update, key string, h Handler) error {
	err := h(dctx, u)
	if err == nil {
		return nil
	}
	var trans *stateTransition
	if errors.As(err, &trans) {
		if trans.end {
			return c.Storage.Delete(ctx, key)
		}
		return c.Storage.Set(ctx, key, trans.next)
	}
	return err
}
