// Package main is a long-poll echo bot. Run with:
//
//	TELEGRAM_BOT_TOKEN=xxx go run ./examples/echo
package main

import (
	"context"
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
		client.WithHTTPClient(client.NewRetryDoer(client.NewDefaultHTTPDoer())))
	me, err := api.GetMe(ctx, bot, &api.GetMeParams{})
	if err != nil {
		log.Fatalf("getMe: %v", err)
	}
	log.Printf("running as @%s", me.Username)

	router := dispatch.New(bot)
	register(router)

	poller := transport.NewLongPoller(bot)
	if err := router.Run(ctx, poller); err != nil && err != context.Canceled {
		log.Printf("router exited: %v", err)
	}
}
