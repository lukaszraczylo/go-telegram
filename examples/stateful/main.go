// Package main demonstrates per-user state without globals via closures.
// Each user has an independent counter that persists for the bot's lifetime.
//
//	TELEGRAM_BOT_TOKEN=xxx go run ./examples/stateful
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/lukaszraczylo/go-telegram/api"
	"github.com/lukaszraczylo/go-telegram/client"
	"github.com/lukaszraczylo/go-telegram/dispatch"
	"github.com/lukaszraczylo/go-telegram/transport"
)

// counterStore is a concurrent-safe per-user counter. Production code
// would back this with Redis / Postgres / sqlite. For demo purposes,
// in-memory is fine.
type counterStore struct {
	mu     sync.Mutex
	counts map[int64]int
}

func (s *counterStore) inc(userID int64) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counts[userID]++
	return s.counts[userID]
}

func main() {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN required")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	bot := client.New(token)
	store := &counterStore{counts: map[int64]int{}}

	router := dispatch.New(bot)
	router.OnCommand("/count", func(c *dispatch.Context, m *api.Message) error {
		if m.From == nil {
			return nil
		}
		n := store.inc(m.From.ID)
		_, err := api.SendMessage(c.Ctx, c.Bot, &api.SendMessageParams{
			ChatID: api.ChatIDFromInt(m.Chat.ID),
			Text:   fmt.Sprintf("Your count: %d", n),
		})
		return err
	})

	if err := router.Run(ctx, transport.NewLongPoller(bot)); err != nil && err != context.Canceled {
		log.Printf("router exited: %v", err)
	}
}
