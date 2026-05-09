// Package main demonstrates group moderation commands: /kick, /ban, /mute, /warn.
//
// The bot must be an admin in the group with "can ban users" permission.
// Use by replying to a target user's message, e.g.: /kick (as a reply).
//
//	TELEGRAM_BOT_TOKEN=xxx go run ./examples/moderation
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/lukaszraczylo/go-telegram/api"
	"github.com/lukaszraczylo/go-telegram/client"
	"github.com/lukaszraczylo/go-telegram/dispatch"
	"github.com/lukaszraczylo/go-telegram/transport"
)

const maxWarns = 3

// warnKey uniquely identifies a user in a chat.
type warnKey struct {
	chatID int64
	userID int64
}

var warnCounts sync.Map // map[warnKey]int — in-memory only; use a DB in production

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
	router.OnCommand("/kick", kickHandler)
	router.OnCommand("/ban", banHandler)
	router.OnCommand("/mute", muteHandler)
	router.OnCommand("/warn", warnHandler)
	router.OnCommand("/unwarn", unwarnHandler)

	poller := transport.NewLongPoller(bot)
	if err := router.Run(ctx, poller); err != nil && err != context.Canceled {
		log.Printf("router exited: %v", err)
	}
}

// resolveTarget returns the user ID of the moderation target.
// Priority: replied-to message sender, then first @mention in the text.
func resolveTarget(m *api.Message) int64 {
	if m.ReplyToMessage != nil && m.ReplyToMessage.From != nil {
		return m.ReplyToMessage.From.ID
	}
	return 0
}

func reply(c *dispatch.Context, chatID int64, text string) {
	_, _ = api.SendMessage(c.Ctx, c.Bot, &api.SendMessageParams{
		ChatID: api.ChatIDFromInt(chatID),
		Text:   text,
	})
}

func handleAdminErr(c *dispatch.Context, chatID int64, err error) error {
	if errors.Is(err, client.ErrForbidden) {
		reply(c, chatID, "I need admin rights with 'can ban users' permission.")
		return nil
	}
	return err
}

func kickHandler(c *dispatch.Context, m *api.Message) error {
	target := resolveTarget(m)
	if target == 0 {
		reply(c, m.Chat.ID, "Reply to a user's message with /kick to remove them.")
		return nil
	}
	// Kick = ban then immediately unban (removes from group, can rejoin).
	if _, err := api.BanChatMember(c.Ctx, c.Bot, &api.BanChatMemberParams{
		ChatID: api.ChatIDFromInt(m.Chat.ID),
		UserID: target,
	}); err != nil {
		return handleAdminErr(c, m.Chat.ID, err)
	}
	if _, err := api.UnbanChatMember(c.Ctx, c.Bot, &api.UnbanChatMemberParams{
		ChatID:       api.ChatIDFromInt(m.Chat.ID),
		UserID:       target,
		OnlyIfBanned: api.Ptr(true),
	}); err != nil {
		log.Printf("unban after kick: %v", err)
	}
	reply(c, m.Chat.ID, fmt.Sprintf("User %d has been kicked.", target))
	return nil
}

func banHandler(c *dispatch.Context, m *api.Message) error {
	target := resolveTarget(m)
	if target == 0 {
		reply(c, m.Chat.ID, "Reply to a user's message with /ban to ban them permanently.")
		return nil
	}
	if _, err := api.BanChatMember(c.Ctx, c.Bot, &api.BanChatMemberParams{
		ChatID: api.ChatIDFromInt(m.Chat.ID),
		UserID: target,
	}); err != nil {
		return handleAdminErr(c, m.Chat.ID, err)
	}
	reply(c, m.Chat.ID, fmt.Sprintf("User %d has been banned.", target))
	return nil
}

func muteHandler(c *dispatch.Context, m *api.Message) error {
	target := resolveTarget(m)
	if target == 0 {
		reply(c, m.Chat.ID, "Reply to a user's message with /mute to silence them for 1 hour.")
		return nil
	}
	if _, err := api.RestrictChatMember(c.Ctx, c.Bot, &api.RestrictChatMemberParams{
		ChatID: api.ChatIDFromInt(m.Chat.ID),
		UserID: target,
		Permissions: api.ChatPermissions{
			CanSendMessages:      api.Ptr(false),
			CanSendAudios:        api.Ptr(false),
			CanSendDocuments:     api.Ptr(false),
			CanSendPhotos:        api.Ptr(false),
			CanSendVideos:        api.Ptr(false),
			CanSendVideoNotes:    api.Ptr(false),
			CanSendVoiceNotes:    api.Ptr(false),
			CanSendPolls:         api.Ptr(false),
			CanSendOtherMessages: api.Ptr(false),
		},
		UntilDate: api.Ptr(time.Now().Add(time.Hour).Unix()),
	}); err != nil {
		return handleAdminErr(c, m.Chat.ID, err)
	}
	reply(c, m.Chat.ID, fmt.Sprintf("User %d muted for 1 hour.", target))
	return nil
}

func warnHandler(c *dispatch.Context, m *api.Message) error {
	target := resolveTarget(m)
	if target == 0 {
		reply(c, m.Chat.ID, "Reply to a user's message with /warn to issue a warning.")
		return nil
	}
	key := warnKey{chatID: m.Chat.ID, userID: target}
	var count int
	if v, ok := warnCounts.Load(key); ok {
		count = v.(int)
	}
	count++
	warnCounts.Store(key, count)

	if count >= maxWarns {
		warnCounts.Delete(key)
		reply(c, m.Chat.ID, fmt.Sprintf("User %d reached %d warnings — auto-banning.", target, maxWarns))
		if _, err := api.BanChatMember(c.Ctx, c.Bot, &api.BanChatMemberParams{
			ChatID: api.ChatIDFromInt(m.Chat.ID),
			UserID: target,
		}); err != nil {
			return handleAdminErr(c, m.Chat.ID, err)
		}
		return nil
	}
	reply(c, m.Chat.ID, fmt.Sprintf("User %d warned (%d/%d). Reach %d and they're banned.", target, count, maxWarns, maxWarns))
	return nil
}

func unwarnHandler(c *dispatch.Context, m *api.Message) error {
	target := resolveTarget(m)
	if target == 0 {
		reply(c, m.Chat.ID, "Reply to a user's message with /unwarn to remove their last warning.")
		return nil
	}
	key := warnKey{chatID: m.Chat.ID, userID: target}
	if v, ok := warnCounts.Load(key); ok {
		count := v.(int) - 1
		if count <= 0 {
			warnCounts.Delete(key)
			reply(c, m.Chat.ID, fmt.Sprintf("User %d has no more warnings.", target))
		} else {
			warnCounts.Store(key, count)
			reply(c, m.Chat.ID, fmt.Sprintf("User %d warning removed (%d/%d remaining).", target, count, maxWarns))
		}
	} else {
		reply(c, m.Chat.ID, fmt.Sprintf("User %d has no warnings.", target))
	}
	return nil
}
