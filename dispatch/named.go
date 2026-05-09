package dispatch

import (
	"fmt"
	"sync"
)

// NamedHandlers manages handlers by string name, allowing runtime
// registration, replacement, and removal. This complements the Router's
// registration methods: each registration via Named*() also gets a name
// for later lookup.
//
// Use case: a plugin system that loads/unloads command handlers without
// restarting the bot.
type NamedHandlers[T any] struct {
	mu       sync.RWMutex
	handlers map[string]Handler[T]
	order    []string // preserves registration order
}

// NewNamedHandlers returns a new, empty NamedHandlers[T].
func NewNamedHandlers[T any]() *NamedHandlers[T] {
	return &NamedHandlers[T]{handlers: map[string]Handler[T]{}}
}

// Set registers or replaces the handler under name. If name is new, it is
// appended to the end of the registration order.
func (n *NamedHandlers[T]) Set(name string, h Handler[T]) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if _, exists := n.handlers[name]; !exists {
		n.order = append(n.order, name)
	}
	n.handlers[name] = h
}

// Remove unregisters the handler under name. Returns true if it existed.
func (n *NamedHandlers[T]) Remove(name string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	if _, ok := n.handlers[name]; !ok {
		return false
	}
	delete(n.handlers, name)
	for i, k := range n.order {
		if k == name {
			n.order = append(n.order[:i], n.order[i+1:]...)
			break
		}
	}
	return true
}

// Has reports whether name is registered.
func (n *NamedHandlers[T]) Has(name string) bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	_, ok := n.handlers[name]
	return ok
}

// Names returns the registered names in registration order.
func (n *NamedHandlers[T]) Names() []string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	out := make([]string, len(n.order))
	copy(out, n.order)
	return out
}

// Handler returns a single Handler[T] that runs each registered handler
// in registration order, first non-nil error stops the chain. Use this
// to wire NamedHandlers into a Router.OnXxx call:
//
//	names := dispatch.NewNamedHandlers[*api.Message]()
//	names.Set("logger", loggingHandler)
//	names.Set("audit", auditHandler)
//	router.OnCommand("/admin", names.Handler())
//
// Subsequent Set/Remove calls take effect on the next dispatch.
func (n *NamedHandlers[T]) Handler() Handler[T] {
	return func(c *Context, payload T) error {
		n.mu.RLock()
		names := make([]string, len(n.order))
		copy(names, n.order)
		n.mu.RUnlock()

		for _, name := range names {
			n.mu.RLock()
			h, ok := n.handlers[name]
			n.mu.RUnlock()
			if !ok {
				continue
			}
			if err := h(c, payload); err != nil {
				return fmt.Errorf("named handler %q: %w", name, err)
			}
		}
		return nil
	}
}
