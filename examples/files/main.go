// Package main demonstrates file upload and download with go-telegram.
// Send any document or photo to the bot — it downloads, then re-uploads
// the file with a "received: <bytes>" caption.
//
//	TELEGRAM_BOT_TOKEN=xxx go run ./examples/files
package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
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
		client.WithHTTPClient(client.NewRetryDoer(client.NewDefaultHTTPDoer())),
	)

	router := dispatch.New(bot)

	router.OnCommand("/start", func(c *dispatch.Context, m *api.Message) error {
		_, err := api.SendMessage(c.Ctx, c.Bot, &api.SendMessageParams{
			ChatID: api.ChatIDFromInt(m.Chat.ID),
			Text:   "Send me a document and I'll download then re-upload it.",
		})
		return err
	})

	// Document uploads land in Message.Document. Use middleware to handle them.
	router.Use(func(next dispatch.Handler[*api.Update]) dispatch.Handler[*api.Update] {
		return func(c *dispatch.Context, u *api.Update) error {
			if u.Message != nil && u.Message.Document != nil {
				return handleDocument(c, u.Message)
			}
			return next(c, u)
		}
	})

	poller := transport.NewLongPoller(bot)
	if err := router.Run(ctx, poller); err != nil && err != context.Canceled {
		log.Printf("router exited: %v", err)
	}
}

func handleDocument(c *dispatch.Context, m *api.Message) error {
	doc := m.Document
	fileSize := int64(0)
	if doc.FileSize != nil {
		fileSize = *doc.FileSize
	}
	log.Printf("received: %s (%d bytes)", doc.FileName, fileSize)

	// Download.
	rc, _, err := api.DownloadFile(c.Ctx, c.Bot, doc.FileID)
	if err != nil {
		return reply(c, m.Chat.ID, fmt.Sprintf("download failed: %v", err))
	}
	defer func() {
		_ = rc.Close()
	}()
	body, err := io.ReadAll(rc)
	if err != nil {
		return reply(c, m.Chat.ID, fmt.Sprintf("read failed: %v", err))
	}

	// Re-upload.
	name := filepath.Base(doc.FileName)
	if name == "" || name == "." {
		name = "file"
	}
	_, err = api.SendDocument(c.Ctx, c.Bot, &api.SendDocumentParams{
		ChatID: api.ChatIDFromInt(m.Chat.ID),
		Document: &api.InputFile{
			Reader:   bytes.NewReader(body),
			Filename: name,
		},
		Caption: fmt.Sprintf("received: %d bytes", len(body)),
	})
	if err != nil {
		return reply(c, m.Chat.ID, fmt.Sprintf("upload failed: %v", err))
	}
	return nil
}

func reply(c *dispatch.Context, chatID int64, text string) error {
	_, err := api.SendMessage(c.Ctx, c.Bot, &api.SendMessageParams{
		ChatID: api.ChatIDFromInt(chatID),
		Text:   text,
	})
	return err
}
