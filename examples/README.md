# Examples

Each subdirectory contains a self-contained sample bot demonstrating one feature area.

| Example | What it shows |
|---|---|
| [echo](./echo) | Long-poll bot that echoes text back to the sender |
| [webhook](./webhook) | Webhook delivery with secret-token verification |
| [callback](./callback) | Inline keyboard with callback queries and counter state |
| [conversation](./conversation) | Multi-step conversation flow with `dispatch/conversation` |
| [files](./files) | Upload and download files via `api.DownloadFile` |
| [inline](./inline) | Inline-mode bot returning search-style results |
| [middleware](./middleware) | Custom middleware chains via `Router.Use` |
| [stateful](./stateful) | Per-user state managed via closures |
| [welcome](./welcome) | Greet new chat members; detect and log departures |
| [moderation](./moderation) | `/kick`, `/ban`, `/mute`, `/warn` with admin permission checks |
| [polls](./polls) | Create polls and tally answers via `OnPollAnswer` |
| [payments](./payments) | Telegram Payments: sendInvoice → pre_checkout_query → successful_payment |
| [pagination](./pagination) | Multi-page inline keyboard with stateless prev/next navigation |
| [admin](./admin) | Auth middleware allowlisting specific user IDs via `Router.Use` |

## Running

All examples follow the same pattern:

```bash
export TELEGRAM_BOT_TOKEN=123456:ABC...
go run ./examples/<name>
```

Webhook examples need a public HTTPS endpoint (use Cloudflare Tunnel, ngrok, or similar).

## Common patterns

**Retry-safe HTTP** — every example wraps the HTTP client with `client.NewRetryDoer`, which automatically honours Telegram's `retry_after` field on 429 responses.

**Graceful shutdown** — all examples use `signal.NotifyContext` so the bot drains cleanly on `SIGINT`/`SIGTERM`.

**Structured logging** — for production, wire a logger via `client.WithLogger` and wrap the process in supervision (systemd unit, k8s liveness probe, etc.).
