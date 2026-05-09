// Package main demonstrates custom middleware for go-telegram.
//
//	TELEGRAM_BOT_TOKEN=xxx go run ./examples/middleware
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lukaszraczylo/go-telegram/api"
	"github.com/lukaszraczylo/go-telegram/client"
	"github.com/lukaszraczylo/go-telegram/dispatch"
	"github.com/lukaszraczylo/go-telegram/transport"
)

// timing wraps a handler chain with start-end timing logged via stdlib log.
func timing() dispatch.Middleware[*api.Update] {
	return func(next dispatch.Handler[*api.Update]) dispatch.Handler[*api.Update] {
		return func(c *dispatch.Context, u *api.Update) error {
			start := time.Now()
			err := next(c, u)
			log.Printf("update %d processed in %s err=%v", u.UpdateID, time.Since(start), err)
			return err
		}
	}
}

// auth restricts updates to a single allowed user ID via env var.
func auth(allowedUserID int64) dispatch.Middleware[*api.Update] {
	return func(next dispatch.Handler[*api.Update]) dispatch.Handler[*api.Update] {
		return func(c *dispatch.Context, u *api.Update) error {
			sender := senderID(u)
			if allowedUserID != 0 && sender != allowedUserID {
				log.Printf("blocked update %d from user %d", u.UpdateID, sender)
				return nil
			}
			return next(c, u)
		}
	}
}

func senderID(u *api.Update) int64 {
	switch {
	case u.Message != nil && u.Message.From != nil:
		return u.Message.From.ID
	case u.CallbackQuery != nil:
		return u.CallbackQuery.From.ID
	}
	return 0
}

func main() {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN required")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	bot := client.New(token)
	router := dispatch.New(bot)
	router.Use(timing())
	// To restrict to a single owner, set OWNER_USER_ID env to your numeric ID.
	var ownerID int64
	_, _ = fmt.Sscanf(os.Getenv("OWNER_USER_ID"), "%d", &ownerID)
	if ownerID != 0 {
		router.Use(auth(ownerID))
	}

	router.OnCommand("/start", func(c *dispatch.Context, m *api.Message) error {
		_, err := api.SendMessage(c.Ctx, c.Bot, &api.SendMessageParams{
			ChatID: api.ChatIDFromInt(m.Chat.ID),
			Text:   "middleware demo: this update was timed and (optionally) auth-checked",
		})
		return err
	})

	if err := router.Run(ctx, transport.NewLongPoller(bot)); err != nil && err != context.Canceled {
		log.Printf("router exited: %v", err)
	}
}
