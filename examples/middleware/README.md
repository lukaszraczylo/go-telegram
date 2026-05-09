# middleware

Demonstrates two custom dispatch middlewares: timing (logs handler latency) and auth (restricts updates to a single owner via `OWNER_USER_ID` env var).

## Run

```bash
export TELEGRAM_BOT_TOKEN=...
export OWNER_USER_ID=123456789  # optional
go run ./examples/middleware
```
