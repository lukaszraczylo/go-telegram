# webhook

Bot using HTTPS webhooks. Replies to `/ping` with `pong`.

## Run

You need a public HTTPS endpoint pointed at port 8080. For local development use a tunnel like Cloudflare Tunnel or ngrok.

```bash
export TELEGRAM_BOT_TOKEN=123456:ABC...
export WEBHOOK_URL=https://your.tunnel.example/bot
export WEBHOOK_SECRET=randomsecret123
go run ./examples/webhook
```
