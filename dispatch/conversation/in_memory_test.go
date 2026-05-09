package conversation

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMemoryStorage_GetSetDelete(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStorage()

	// Get on empty key returns ErrKeyNotFound.
	_, err := s.Get(ctx, "k1")
	require.ErrorIs(t, err, ErrKeyNotFound)

	// Set then Get returns the stored state.
	require.NoError(t, s.Set(ctx, "k1", "step_a"))
	v, err := s.Get(ctx, "k1")
	require.NoError(t, err)
	require.Equal(t, State("step_a"), v)

	// Overwrite works.
	require.NoError(t, s.Set(ctx, "k1", "step_b"))
	v, err = s.Get(ctx, "k1")
	require.NoError(t, err)
	require.Equal(t, State("step_b"), v)

	// Delete removes the key.
	require.NoError(t, s.Delete(ctx, "k1"))
	_, err = s.Get(ctx, "k1")
	require.ErrorIs(t, err, ErrKeyNotFound)

	// Delete of non-existent key is a no-op (no error).
	require.NoError(t, s.Delete(ctx, "nonexistent"))
}

func TestMemoryStorage_MultipleKeys(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStorage()

	require.NoError(t, s.Set(ctx, "a", "stateA"))
	require.NoError(t, s.Set(ctx, "b", "stateB"))

	va, err := s.Get(ctx, "a")
	require.NoError(t, err)
	require.Equal(t, State("stateA"), va)

	vb, err := s.Get(ctx, "b")
	require.NoError(t, err)
	require.Equal(t, State("stateB"), vb)

	// Delete one key; the other remains.
	require.NoError(t, s.Delete(ctx, "a"))
	_, err = s.Get(ctx, "a")
	require.ErrorIs(t, err, ErrKeyNotFound)

	vb, err = s.Get(ctx, "b")
	require.NoError(t, err)
	require.Equal(t, State("stateB"), vb)
}

func TestMemoryStorage_Concurrent(t *testing.T) {
	// 20 goroutines hammering the same key concurrently — no data race.
	ctx := context.Background()
	s := NewMemoryStorage()

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(i int) {
			defer wg.Done()
			key := "shared"
			_ = s.Set(ctx, key, State("step"))
			_, _ = s.Get(ctx, key)
			if i%3 == 0 {
				_ = s.Delete(ctx, key)
			}
		}(i)
	}
	wg.Wait()
	// No assertion needed — race detector catches the bug if present.
}
