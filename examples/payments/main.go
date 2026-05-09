// Package main demonstrates the Telegram Payments flow:
//
//  1. /buy  → bot sends an invoice via sendInvoice
//  2. User confirms → Telegram sends pre_checkout_query → bot answers ok=true
//  3. User pays → Telegram sends successful_payment in a Message
//
// For testing, use Telegram's test payment provider.
// Set PAYMENT_PROVIDER_TOKEN to the token from @BotFather (test or live).
// For Telegram Stars payments, set PAYMENT_PROVIDER_TOKEN="" and CURRENCY="XTR".
//
//	TELEGRAM_BOT_TOKEN=xxx PAYMENT_PROVIDER_TOKEN=yyy go run ./examples/payments
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
	providerToken := os.Getenv("PAYMENT_PROVIDER_TOKEN") // empty = Telegram Stars (XTR)
	currency := os.Getenv("CURRENCY")
	if currency == "" {
		currency = "USD"
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	bot := client.New(token,
		client.WithHTTPClient(client.NewRetryDoer(client.NewDefaultHTTPDoer())),
	)

	router := dispatch.New(bot)

	// Step 1: user sends /buy — bot replies with an invoice.
	router.OnCommand("/buy", func(c *dispatch.Context, m *api.Message) error {
		_, err := api.SendInvoice(c.Ctx, c.Bot, &api.SendInvoiceParams{
			ChatID:        api.ChatIDFromInt(m.Chat.ID),
			Title:         "Premium Widget",
			Description:   "A top-quality widget that does absolutely everything.",
			Payload:       "widget-purchase-v1",
			ProviderToken: providerToken,
			Currency:      currency,
			Prices: []api.LabeledPrice{
				{Label: "Widget", Amount: 199}, // $1.99
				{Label: "Tax", Amount: 20},     // $0.20
			},
		})
		if err != nil {
			log.Printf("sendInvoice error: %v", err)
		}
		return err
	})

	// Step 2: user confirms order — Telegram sends pre_checkout_query.
	// Bot MUST respond within 10 seconds.
	router.OnPreCheckoutQuery(func(c *dispatch.Context, q *api.PreCheckoutQuery) error {
		log.Printf("pre_checkout: id=%s payload=%q total=%d %s from=%d",
			q.ID, q.InvoicePayload, q.TotalAmount, q.Currency, q.From.ID)

		// Validate the order here (check stock, pricing, etc.).
		// For this demo, always approve.
		_, err := api.AnswerPreCheckoutQuery(c.Ctx, c.Bot, &api.AnswerPreCheckoutQueryParams{
			PreCheckoutQueryID: q.ID,
			Ok:                 true,
		})
		return err
	})

	// Step 3: payment completed — Telegram delivers a Message.SuccessfulPayment.
	router.OnMessageFilter(
		func(m *api.Message) bool { return m.SuccessfulPayment != nil },
		func(c *dispatch.Context, m *api.Message) error {
			sp := m.SuccessfulPayment
			log.Printf("payment success: charge_id=%s payload=%q amount=%d %s",
				sp.TelegramPaymentChargeID, sp.InvoicePayload, sp.TotalAmount, sp.Currency)
			_, err := api.SendMessage(c.Ctx, c.Bot, &api.SendMessageParams{
				ChatID: api.ChatIDFromInt(m.Chat.ID),
				Text:   "Payment received! Your widget is on its way.",
			})
			return err
		},
	)

	poller := transport.NewLongPoller(bot)
	if err := router.Run(ctx, poller); err != nil && err != context.Canceled {
		log.Printf("router exited: %v", err)
	}
}
