package api

import (
	"context"
	"sync"

	"github.com/lukaszraczylo/go-telegram/client"
)

// MeCache caches the result of GetMe across calls. Construct one per
// Bot and call Get to retrieve the cached User on subsequent invocations.
//
//	var meCache api.MeCache
//	me, err := meCache.Get(ctx, bot)
//
// MeCache is safe for concurrent use.
type MeCache struct {
	mu     sync.Mutex
	cached *User
}

// Get returns the User from a cached GetMe call. If the cache is empty,
// it calls GetMe and populates the cache on success.
func (c *MeCache) Get(ctx context.Context, b *client.Bot) (*User, error) {
	c.mu.Lock()
	if c.cached != nil {
		u := c.cached
		c.mu.Unlock()
		return u, nil
	}
	c.mu.Unlock()

	u, err := GetMe(ctx, b, &GetMeParams{})
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.cached = u
	c.mu.Unlock()
	return u, nil
}

// Reset clears the cache. Useful in tests or after the bot's identity
// is known to have changed (very rare).
func (c *MeCache) Reset() {
	c.mu.Lock()
	c.cached = nil
	c.mu.Unlock()
}
