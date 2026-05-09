package main

import (
	"fmt"

	"github.com/lukaszraczylo/go-telegram/api"
	"github.com/lukaszraczylo/go-telegram/dispatch"
)

// register wires all handlers onto the router. Exposed so tests can call
// handlers directly without going through the router run loop.
func register(r *dispatch.Router) {
	r.OnCommand("/start", handleStart)
	r.OnText(`.+`, handleEcho)
}

func handleStart(c *dispatch.Context, m *api.Message) error {
	_, err := api.SendMessage(c.Ctx, c.Bot, &api.SendMessageParams{
		ChatID: api.ChatIDFromInt(m.Chat.ID),
		Text:   fmt.Sprintf("hello %s, send me anything to echo", m.From.FirstName),
	})
	return err
}

func handleEcho(c *dispatch.Context, m *api.Message) error {
	_, err := api.SendMessage(c.Ctx, c.Bot, &api.SendMessageParams{
		ChatID:          api.ChatIDFromInt(m.Chat.ID),
		Text:            m.Text,
		ReplyParameters: &api.ReplyParameters{MessageID: m.MessageID},
	})
	return err
}
