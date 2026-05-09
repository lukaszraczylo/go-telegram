package main

import (
	"github.com/lukaszraczylo/go-telegram/api"
	"github.com/lukaszraczylo/go-telegram/dispatch"
)

// register wires all handlers onto the router.
func register(r *dispatch.Router) {
	r.OnCommand("/ping", handlePing)
}

func handlePing(c *dispatch.Context, m *api.Message) error {
	_, err := api.SendMessage(c.Ctx, c.Bot, &api.SendMessageParams{
		ChatID: api.ChatIDFromInt(m.Chat.ID),
		Text:   "pong",
	})
	return err
}
