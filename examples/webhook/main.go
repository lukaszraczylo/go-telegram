// Package main is a webhook bot. Run with:
//
//	TELEGRAM_BOT_TOKEN=xxx \
//	WEBHOOK_URL=https://example.com/bot \
//	WEBHOOK_SECRET=somethingrandom \
//	go run ./examples/webhook
//
// The bot sets its webhook to WEBHOOK_URL on startup, listens on :8080,
// and clears the webhook on shutdown.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lukaszraczylo/go-telegram/api"
	"github.com/lukaszraczylo/go-telegram/client"
	"github.com/lukaszraczylo/go-telegram/dispatch"
	"github.com/lukaszraczylo/go-telegram/transport"
)

func main() {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	url := os.Getenv("WEBHOOK_URL")
	secret := os.Getenv("WEBHOOK_SECRET")
	if token == "" || url == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN and WEBHOOK_URL required")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	bot := client.New(token,
		client.WithHTTPClient(client.NewRetryDoer(client.NewDefaultHTTPDoer())))

	if _, err := api.SetWebhook(ctx, bot, &api.SetWebhookParams{
		URL:         url,
		SecretToken: secret,
	}); err != nil {
		log.Fatalf("setWebhook: %v", err)
	}
	defer func() {
		_, _ = api.DeleteWebhook(context.Background(), bot, &api.DeleteWebhookParams{})
	}()

	wh := transport.NewWebhookServer(bot)
	wh.SecretToken = secret

	router := dispatch.New(bot)
	register(router)

	mux := http.NewServeMux()
	mux.Handle("/bot", wh)
	srv := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("http server exited: %v", err)
			stop()
		}
	}()

	if err := router.Run(ctx, wh); err != nil && err != context.Canceled {
		log.Printf("router exited: %v", err)
	}
	_ = srv.Shutdown(context.Background())
}
