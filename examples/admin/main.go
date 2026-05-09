// Package main demonstrates auth middleware that restricts the bot to an
// allowlist of Telegram user IDs.
//
// Set ALLOWED_USERS to a comma-separated list of numeric user IDs.
// Messages from all other users are silently dropped.
//
//	TELEGRAM_BOT_TOKEN=xxx ALLOWED_USERS=123456,789012 go run ./examples/admin
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/lukaszraczylo/go-telegram/api"
	"github.com/lukaszraczylo/go-telegram/client"
	"github.com/lukaszraczylo/go-telegram/dispatch"
	"github.com/lukaszraczylo/go-telegram/transport"
)

// parseAllowedIDs parses a comma-separated list of user IDs from an env var.
func parseAllowedIDs(raw string) map[int64]bool {
	out := make(map[int64]bool)
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			log.Printf("invalid user ID in ALLOWED_USERS: %q — skipping", part)
			continue
		}
		out[id] = true
	}
	return out
}

// extractSenderID returns the Telegram user ID from the most common update types.
func extractSenderID(u *api.Update) int64 {
	if u.Message != nil && u.Message.From != nil {
		return u.Message.From.ID
	}
	if u.EditedMessage != nil && u.EditedMessage.From != nil {
		return u.EditedMessage.From.ID
	}
	if u.CallbackQuery != nil && u.CallbackQuery.From.ID != 0 {
		return u.CallbackQuery.From.ID
	}
	if u.InlineQuery != nil {
		return u.InlineQuery.From.ID
	}
	return 0
}

// allowlistMiddleware drops updates from users not in the allowlist.
// Passing an empty allowlist (ALLOWED_USERS unset) allows everyone through,
// so this example is safe to run without the env var set.
func allowlistMiddleware(allowed map[int64]bool) dispatch.Middleware[*api.Update] {
	return func(next dispatch.Handler[*api.Update]) dispatch.Handler[*api.Update] {
		return func(c *dispatch.Context, u *api.Update) error {
			if len(allowed) == 0 {
				// No allowlist configured — permit all.
				return next(c, u)
			}
			senderID := extractSenderID(u)
			if senderID != 0 && !allowed[senderID] {
				log.Printf("dropping update from unauthorized user %d", senderID)
				return nil // Silent drop.
			}
			return next(c, u)
		}
	}
}

func main() {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN required")
	}

	allowedIDs := parseAllowedIDs(os.Getenv("ALLOWED_USERS"))
	if len(allowedIDs) == 0 {
		log.Println("ALLOWED_USERS not set — all users permitted (demo mode)")
	} else {
		log.Printf("allowlist: %d user(s)", len(allowedIDs))
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	bot := client.New(token,
		client.WithHTTPClient(client.NewRetryDoer(client.NewDefaultHTTPDoer())),
	)

	router := dispatch.New(bot)
	router.Use(allowlistMiddleware(allowedIDs))

	router.OnCommand("/start", func(c *dispatch.Context, m *api.Message) error {
		_, err := api.SendMessage(c.Ctx, c.Bot, &api.SendMessageParams{
			ChatID: api.ChatIDFromInt(m.Chat.ID),
			Text:   "You are authorized.",
		})
		return err
	})

	router.OnCommand("/whoami", func(c *dispatch.Context, m *api.Message) error {
		from := m.From
		if from == nil {
			return nil
		}
		_, err := api.SendMessage(c.Ctx, c.Bot, &api.SendMessageParams{
			ChatID: api.ChatIDFromInt(m.Chat.ID),
			Text:   "Your user ID is: " + strconv.FormatInt(from.ID, 10),
		})
		return err
	})

	poller := transport.NewLongPoller(bot)
	if err := router.Run(ctx, poller); err != nil && err != context.Canceled {
		log.Printf("router exited: %v", err)
	}
}
