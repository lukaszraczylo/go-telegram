// Package main demonstrates greeting new chat members and detecting leaves.
//
// The bot must be an admin in the group with "can read messages" (or at least
// be able to receive service messages) to get new-member and left-member events.
//
//	TELEGRAM_BOT_TOKEN=xxx go run ./examples/welcome
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/lukaszraczylo/go-telegram/api"
	"github.com/lukaszraczylo/go-telegram/client"
	"github.com/lukaszraczylo/go-telegram/dispatch"
	"github.com/lukaszraczylo/go-telegram/transport"
)

func main() {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN required")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	bot := client.New(token,
		client.WithHTTPClient(client.NewRetryDoer(client.NewDefaultHTTPDoer())),
	)

	router := dispatch.New(bot)

	// Greet every new member that joins the group.
	router.OnMessageFilter(
		func(m *api.Message) bool { return len(m.NewChatMembers) > 0 },
		func(c *dispatch.Context, m *api.Message) error {
			for _, u := range m.NewChatMembers {
				name := u.FirstName
				if u.LastName != "" {
					name += " " + u.LastName
				}
				_, err := api.SendMessage(c.Ctx, c.Bot, &api.SendMessageParams{
					ChatID: api.ChatIDFromInt(m.Chat.ID),
					Text:   fmt.Sprintf("Welcome, %s! Please read the pinned rules before posting.", name),
				})
				if err != nil {
					log.Printf("send welcome: %v", err)
				}
			}
			return nil
		},
	)

	// Log when a member leaves (or is removed from) the group.
	router.OnMessageFilter(
		func(m *api.Message) bool { return m.LeftChatMember != nil },
		func(c *dispatch.Context, m *api.Message) error {
			log.Printf("user %d (%s) left chat %d",
				m.LeftChatMember.ID,
				m.LeftChatMember.FirstName,
				m.Chat.ID,
			)
			return nil
		},
	)

	// Detect when the bot itself is added to a group.
	router.OnMyChatMember(func(c *dispatch.Context, u *api.ChatMemberUpdated) error {
		log.Printf("bot chat membership changed in %d: new status = %T", u.Chat.ID, u.NewChatMember)
		return nil
	})

	poller := transport.NewLongPoller(bot)
	if err := router.Run(ctx, poller); err != nil && err != context.Canceled {
		log.Printf("router exited: %v", err)
	}
}
