// Package main demonstrates multi-page inline keyboard navigation.
//
// /list shows page 1 of a sample item list with [« Prev] [Next »] buttons.
// Tapping a button edits the message in-place to show the next/previous page.
// Page state is encoded directly in callback data ("page:<n>") — no server-side
// session needed.
//
//	TELEGRAM_BOT_TOKEN=xxx go run ./examples/pagination
package main

import (
	"context"
	"fmt"
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

const itemsPerPage = 5

// sampleItems is the list being paginated.
var sampleItems = []string{
	"Alpha", "Bravo", "Charlie", "Delta", "Echo",
	"Foxtrot", "Golf", "Hotel", "India", "Juliet",
	"Kilo", "Lima", "Mike", "November", "Oscar",
	"Papa", "Quebec", "Romeo", "Sierra", "Tango",
}

// renderPage builds the message text and keyboard for the given page number.
func renderPage(items []string, page int) (string, *api.InlineKeyboardMarkup) {
	start := page * itemsPerPage
	if start >= len(items) {
		start = 0
		page = 0
	}
	end := start + itemsPerPage
	if end > len(items) {
		end = len(items)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Items (page %d of %d):\n\n", page+1, totalPages(len(items)))
	for i := start; i < end; i++ {
		fmt.Fprintf(&sb, "%d. %s\n", i+1, items[i])
	}

	var btns []api.InlineKeyboardButton
	if page > 0 {
		btns = append(btns, api.InlineKeyboardButton{
			Text:         "« Prev",
			CallbackData: fmt.Sprintf("page:%d", page-1),
		})
	}
	if end < len(items) {
		btns = append(btns, api.InlineKeyboardButton{
			Text:         "Next »",
			CallbackData: fmt.Sprintf("page:%d", page+1),
		})
	}

	var markup *api.InlineKeyboardMarkup
	if len(btns) > 0 {
		markup = &api.InlineKeyboardMarkup{
			InlineKeyboard: [][]api.InlineKeyboardButton{btns},
		}
	}
	return sb.String(), markup
}

func totalPages(total int) int {
	pages := total / itemsPerPage
	if total%itemsPerPage != 0 {
		pages++
	}
	return pages
}

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

	// /list — send page 0.
	router.OnCommand("/list", func(c *dispatch.Context, m *api.Message) error {
		text, markup := renderPage(sampleItems, 0)
		_, err := api.SendMessage(c.Ctx, c.Bot, &api.SendMessageParams{
			ChatID:      api.ChatIDFromInt(m.Chat.ID),
			Text:        text,
			ReplyMarkup: markup,
		})
		return err
	})

	// page:<n> callbacks — edit message in-place.
	router.OnCallback(`^page:(\d+)$`, func(c *dispatch.Context, q *api.CallbackQuery) error {
		groups := c.Values["regex_match"].([]string)
		page, _ := strconv.Atoi(groups[1])

		// Acknowledge the tap first.
		_, _ = api.AnswerCallbackQuery(c.Ctx, c.Bot, &api.AnswerCallbackQueryParams{
			CallbackQueryID: q.ID,
		})

		if q.Message == nil {
			return nil
		}
		msg, ok := q.Message.(*api.Message)
		if !ok {
			return nil
		}

		text, markup := renderPage(sampleItems, page)
		chatID := api.ChatIDFromInt(msg.Chat.ID)
		mid := msg.MessageID
		_, err := api.EditMessageText(c.Ctx, c.Bot, &api.EditMessageTextParams{
			ChatID:      &chatID,
			MessageID:   &mid,
			Text:        text,
			ReplyMarkup: markup,
		})
		return err
	})

	poller := transport.NewLongPoller(bot)
	if err := router.Run(ctx, poller); err != nil && err != context.Canceled {
		log.Printf("router exited: %v", err)
	}
}
