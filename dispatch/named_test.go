package dispatch

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/lukaszraczylo/go-telegram/api"
	"github.com/stretchr/testify/require"
)

// makeMsg returns a minimal *api.Message for use in handler tests.
func makeMsg() *api.Message {
	return &api.Message{MessageID: 1, Chat: api.Chat{ID: 1, Type: "private"}}
}

// makeCtx returns a minimal *Context (nil bot is fine for unit tests).
func makeCtx() *Context {
	return NewContext(context.Background(), nil, &api.Update{})
}

func TestNamedHandlers_SetAndHas(t *testing.T) {
	n := NewNamedHandlers[*api.Message]()
	require.False(t, n.Has("a"))
	n.Set("a", func(c *Context, m *api.Message) error { return nil })
	require.True(t, n.Has("a"))
}

func TestNamedHandlers_Names_RegistrationOrder(t *testing.T) {
	n := NewNamedHandlers[*api.Message]()
	n.Set("first", func(c *Context, m *api.Message) error { return nil })
	n.Set("second", func(c *Context, m *api.Message) error { return nil })
	n.Set("third", func(c *Context, m *api.Message) error { return nil })
	require.Equal(t, []string{"first", "second", "third"}, n.Names())
}

func TestNamedHandlers_Remove(t *testing.T) {
	n := NewNamedHandlers[*api.Message]()
	n.Set("a", func(c *Context, m *api.Message) error { return nil })
	n.Set("b", func(c *Context, m *api.Message) error { return nil })

	removed := n.Remove("a")
	require.True(t, removed)
	require.False(t, n.Has("a"))
	require.Equal(t, []string{"b"}, n.Names())

	// Remove non-existent returns false.
	require.False(t, n.Remove("nonexistent"))
}

func TestNamedHandlers_Replacement_SameOrderSlot(t *testing.T) {
	n := NewNamedHandlers[*api.Message]()
	n.Set("a", func(c *Context, m *api.Message) error { return nil })
	n.Set("b", func(c *Context, m *api.Message) error { return nil })

	var called string
	n.Set("a", func(c *Context, m *api.Message) error {
		called = "replaced-a"
		return nil
	})

	// Order must not change; "a" stays first.
	require.Equal(t, []string{"a", "b"}, n.Names())

	h := n.Handler()
	_ = h(makeCtx(), makeMsg())
	require.Equal(t, "replaced-a", called)
}

func TestNamedHandlers_Handler_RunsInOrder(t *testing.T) {
	n := NewNamedHandlers[*api.Message]()
	var calls []string

	n.Set("first", func(c *Context, m *api.Message) error {
		calls = append(calls, "first")
		return nil
	})
	n.Set("second", func(c *Context, m *api.Message) error {
		calls = append(calls, "second")
		return nil
	})

	h := n.Handler()
	require.NoError(t, h(makeCtx(), makeMsg()))
	require.Equal(t, []string{"first", "second"}, calls)
}

func TestNamedHandlers_Handler_ErrorWrappedAndStops(t *testing.T) {
	n := NewNamedHandlers[*api.Message]()
	sentinel := errors.New("boom")

	n.Set("ok", func(c *Context, m *api.Message) error { return nil })
	n.Set("fail", func(c *Context, m *api.Message) error { return sentinel })
	n.Set("never", func(c *Context, m *api.Message) error {
		t.Fatal("should not be called after an error")
		return nil
	})

	h := n.Handler()
	err := h(makeCtx(), makeMsg())
	require.Error(t, err)
	require.True(t, errors.Is(err, sentinel))
	require.Contains(t, err.Error(), `named handler "fail"`)
}

func TestNamedHandlers_Concurrent_SetRemove(t *testing.T) {
	n := NewNamedHandlers[*api.Message]()

	// Pre-populate so Handler() has something to iterate.
	for i := range 5 {
		name := fmt.Sprintf("h%d", i)
		n.Set(name, func(c *Context, m *api.Message) error { return nil })
	}

	h := n.Handler()
	var wg sync.WaitGroup

	// Concurrent readers (invoke handler).
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = h(makeCtx(), makeMsg())
		}()
	}

	// Concurrent writers.
	for i := range 5 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			name := fmt.Sprintf("new%d", i)
			n.Set(name, func(c *Context, m *api.Message) error { return nil })
			n.Remove(fmt.Sprintf("h%d", i))
		}(i)
	}

	wg.Wait()
}

func TestNamedHandlers_RemoveAndReinstate(t *testing.T) {
	n := NewNamedHandlers[*api.Message]()
	n.Set("a", func(c *Context, m *api.Message) error { return nil })
	n.Remove("a")
	require.False(t, n.Has("a"))

	// Re-register after removal; should be added at end.
	n.Set("b", func(c *Context, m *api.Message) error { return nil })
	n.Set("a", func(c *Context, m *api.Message) error { return nil })
	require.Equal(t, []string{"b", "a"}, n.Names())
}
