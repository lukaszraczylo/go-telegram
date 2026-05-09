package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/lukaszraczylo/go-telegram/api"
	"github.com/lukaszraczylo/go-telegram/client"
	"github.com/lukaszraczylo/go-telegram/dispatch"
)

// register wires all handlers onto the router.
func register(r *dispatch.Router) {
	r.OnCommand("/start", handleStart)
	r.OnCallback(`^count:(-?\d+):(inc|dec)$`, handleCallback)
}

func handleStart(c *dispatch.Context, m *api.Message) error {
	return sendMenu(c.Ctx, c.Bot, m.Chat.ID, 0)
}

func handleCallback(c *dispatch.Context, q *api.CallbackQuery) error {
	groups := c.Values["regex_match"].([]string)
	current, _ := strconv.Atoi(groups[1])
	if groups[2] == "inc" {
		current++
	} else {
		current--
	}

	// Acknowledge the callback (removes the loading spinner).
	_, _ = api.AnswerCallbackQuery(c.Ctx, c.Bot, &api.AnswerCallbackQueryParams{
		CallbackQueryID: q.ID,
		Text:            fmt.Sprintf("counter is now %d", current),
	})

	// Edit the message to reflect the new state.
	if q.Message == nil {
		return nil
	}
	msg, ok := q.Message.(*api.Message)
	if !ok {
		return nil
	}
	chatID := api.ChatIDFromInt(msg.Chat.ID)
	mid := msg.MessageID
	_, err := api.EditMessageText(c.Ctx, c.Bot, &api.EditMessageTextParams{
		ChatID:      &chatID,
		MessageID:   &mid,
		Text:        fmt.Sprintf("Counter: %d", current),
		ReplyMarkup: counterKeyboard(current),
	})
	return err
}

func sendMenu(ctx context.Context, bot *client.Bot, chatID int64, value int) error {
	_, err := api.SendMessage(ctx, bot, &api.SendMessageParams{
		ChatID:      api.ChatIDFromInt(chatID),
		Text:        fmt.Sprintf("Counter: %d", value),
		ReplyMarkup: counterKeyboard(value),
	})
	return err
}

func counterKeyboard(value int) *api.InlineKeyboardMarkup {
	return &api.InlineKeyboardMarkup{
		InlineKeyboard: [][]api.InlineKeyboardButton{
			{
				{Text: "−", CallbackData: fmt.Sprintf("count:%d:dec", value)},
				{Text: "+", CallbackData: fmt.Sprintf("count:%d:inc", value)},
			},
		},
	}
}
