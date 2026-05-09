//go:build integration

// Package integration_test contains tests that hit the live Telegram Bot
// API. These tests are gated behind the "integration" build tag and the
// TELEGRAM_BOT_TOKEN environment variable; they do not run on default
// `go test ./...`.
package integration_test

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/lukaszraczylo/go-telegram/api"
	"github.com/lukaszraczylo/go-telegram/client"
	"github.com/stretchr/testify/require"
)

func botFromEnv(t *testing.T) *client.Bot {
	tok := os.Getenv("TELEGRAM_BOT_TOKEN")
	if tok == "" {
		t.Skip("TELEGRAM_BOT_TOKEN not set")
	}
	return client.New(tok)
}

func TestGetMe(t *testing.T) {
	b := botFromEnv(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	u, err := api.GetMe(ctx, b)
	require.NoError(t, err)
	require.True(t, u.IsBot)
}

func TestSendMessage(t *testing.T) {
	b := botFromEnv(t)
	chatRaw := os.Getenv("TELEGRAM_TEST_CHAT_ID")
	if chatRaw == "" {
		t.Skip("TELEGRAM_TEST_CHAT_ID not set")
	}
	chatID, err := strconv.ParseInt(chatRaw, 10, 64)
	require.NoError(t, err, "TELEGRAM_TEST_CHAT_ID must be an integer")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	msg, err := api.SendMessage(ctx, b, &api.SendMessageParams{
		ChatID: chatID,
		Text:   "integration test " + time.Now().UTC().Format(time.RFC3339),
	})
	require.NoError(t, err)
	require.NotZero(t, msg.MessageID)
}

func TestWebhookCycle(t *testing.T) {
	b := botFromEnv(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Make sure no webhook is set first (long-poll mode).
	_, _ = api.DeleteWebhook(ctx, b, &api.DeleteWebhookParams{})

	ok, err := api.SetWebhook(ctx, b, &api.SetWebhookParams{URL: "https://example.invalid/no-receive"})
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = api.DeleteWebhook(ctx, b, &api.DeleteWebhookParams{})
	require.NoError(t, err)
	require.True(t, ok)
}
