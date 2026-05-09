package main

import (
	"github.com/lukaszraczylo/go-telegram/api"
	"github.com/lukaszraczylo/go-telegram/client"
	"github.com/lukaszraczylo/go-telegram/dispatch"
	"github.com/lukaszraczylo/go-telegram/dispatch/conversation"
	msgfilter "github.com/lukaszraczylo/go-telegram/dispatch/filters/message"
)

// onMsg lifts a Filter[*api.Message] into a Filter[*api.Update].
func onMsg(f dispatch.Filter[*api.Message]) dispatch.Filter[*api.Update] {
	return func(u *api.Update) bool { return u.Message != nil && f(u.Message) }
}

// notCmd matches message updates that are NOT a bot command.
var notCmd = onMsg(msgfilter.AnyCommand().Not())

// buildConv constructs the /newbot conversation. Accepts an optional Storage
// so tests can inject MemoryStorage for isolation; pass nil for the default.
func buildConv(storage conversation.Storage) *conversation.Conversation {
	conv := &conversation.Conversation{
		Storage: storage,
		EntryPoints: []conversation.Step{{
			Filter: onMsg(msgfilter.Command("/newbot")),
			Handler: func(c *dispatch.Context, u *api.Update) error {
				_, _ = api.SendMessage(c.Ctx, c.Bot, &api.SendMessageParams{
					ChatID: api.ChatIDFromInt(u.Message.Chat.ID),
					Text:   "What should the bot's name be?",
				})
				return conversation.Next("await_name")
			},
		}},
		States: map[conversation.State][]conversation.Step{
			"await_name": {{
				Filter: notCmd,
				Handler: func(c *dispatch.Context, u *api.Update) error {
					_, _ = api.SendMessage(c.Ctx, c.Bot, &api.SendMessageParams{
						ChatID: api.ChatIDFromInt(u.Message.Chat.ID),
						Text:   "Got it! What's the description?",
					})
					return conversation.Next("await_desc")
				},
			}},
			"await_desc": {{
				Filter: notCmd,
				Handler: func(c *dispatch.Context, u *api.Update) error {
					_, _ = api.SendMessage(c.Ctx, c.Bot, &api.SendMessageParams{
						ChatID: api.ChatIDFromInt(u.Message.Chat.ID),
						Text:   "Done! Your bot has been configured.",
					})
					return conversation.End()
				},
			}},
		},
		Exits: []conversation.Step{{
			Filter: onMsg(msgfilter.Command("/cancel")),
			Handler: func(c *dispatch.Context, u *api.Update) error {
				_, _ = api.SendMessage(c.Ctx, c.Bot, &api.SendMessageParams{
					ChatID: api.ChatIDFromInt(u.Message.Chat.ID),
					Text:   "Cancelled.",
				})
				return conversation.End()
			},
		}},
	}
	return conv
}

// register wires the conversation middleware onto the router.
func register(r *dispatch.Router, bot *client.Bot) {
	_ = bot // bot available for future handlers if needed
	conv := buildConv(nil)
	r.Use(conv.Dispatch)
}
