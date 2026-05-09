// Package main demonstrates inline-mode queries with go-telegram.
//
// Enable inline mode for your bot via @BotFather: /setinline → enable.
// Then type @yourbot something in any chat to see results.
//
//	TELEGRAM_BOT_TOKEN=xxx go run ./examples/inline
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
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

	bot := client.New(token)
	router := dispatch.New(bot)

	router.OnInlineQuery(func(c *dispatch.Context, q *api.InlineQuery) error {
		// Echo the query as article results.
		results := []api.InlineQueryResult{
			&api.InlineQueryResultArticle{
				ID:    "echo",
				Title: "Echo: " + q.Query,
				InputMessageContent: &api.InputTextMessageContent{
					MessageText: q.Query,
				},
			},
			&api.InlineQueryResultArticle{
				ID:    "upper",
				Title: "UPPER: " + strings.ToUpper(q.Query),
				InputMessageContent: &api.InputTextMessageContent{
					MessageText: strings.ToUpper(q.Query),
				},
			},
		}
		_, err := api.AnswerInlineQuery(c.Ctx, c.Bot, &api.AnswerInlineQueryParams{
			InlineQueryID: q.ID,
			Results:       results,
		})
		return err
	})

	if err := router.Run(ctx, transport.NewLongPoller(bot)); err != nil && err != context.Canceled {
		log.Printf("router exited: %v", err)
	}
}
