package conversation

import (
	"context"
	"sync"
)

// MemoryStorage is the default in-process Storage. It is safe for
// concurrent use. Conversation state is lost on process restart; use
// a custom Storage backed by a database for persistent flows.
type MemoryStorage struct {
	mu    sync.RWMutex
	state map[string]State
}

// NewMemoryStorage constructs an empty in-memory storage.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{state: map[string]State{}}
}

func (s *MemoryStorage) Get(_ context.Context, key string) (State, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.state[key]
	if !ok {
		return "", ErrKeyNotFound
	}
	return v, nil
}

func (s *MemoryStorage) Set(_ context.Context, key string, state State) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state[key] = state
	return nil
}

func (s *MemoryStorage) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.state, key)
	return nil
}
