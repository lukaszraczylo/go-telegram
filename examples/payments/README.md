# payments

Full Telegram Payments flow: invoice → pre-checkout confirmation → successful payment.

## What it shows

- `api.SendInvoice` to send a product invoice with `LabeledPrice` breakdown
- `router.OnPreCheckoutQuery` + `api.AnswerPreCheckoutQuery` — must respond within 10 s
- `router.OnMessageFilter` matching `Message.SuccessfulPayment` to confirm payment

## Environment variables

| Variable | Required | Description |
|---|---|---|
| `TELEGRAM_BOT_TOKEN` | Yes | Bot token from @BotFather |
| `PAYMENT_PROVIDER_TOKEN` | No | Provider token from @BotFather. Leave empty for Telegram Stars (XTR). |
| `CURRENCY` | No | ISO 4217 code (default: `USD`). Use `XTR` for Stars. |

## Test payments

Telegram provides a test payment provider to avoid real charges during development:

1. In @BotFather, use `/mybots` → choose your bot → **Payments** → select "Stripe TEST".
2. Use the test provider token — test payments are free and won't charge users.
3. In the Telegram client, use a test card number such as `4242 4242 4242 4242`.

**Never expose a live provider token in source code.** Use environment variables or secrets management.

## Flow

```
User: /buy
Bot:  [Invoice message — "Premium Widget $2.19"]
User: [taps Pay]
Telegram → Bot: pre_checkout_query (bot has 10 s to respond)
Bot → Telegram: answerPreCheckoutQuery ok=true
Telegram → Bot: Message.SuccessfulPayment
Bot: "Payment received! Your widget is on its way."
```

## Running

```bash
export TELEGRAM_BOT_TOKEN=123456:ABC...
export PAYMENT_PROVIDER_TOKEN=<stripe-test-token>
go run ./examples/payments
```
