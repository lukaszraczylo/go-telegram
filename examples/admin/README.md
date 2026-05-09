# admin

Authentication middleware that restricts the bot to an allowlist of Telegram user IDs.

## What it shows

- `router.Use(...)` to install a global `Middleware[*api.Update]`
- Parsing `ALLOWED_USERS` env var into a `map[int64]bool` lookup set
- Extracting sender ID from multiple update types in one helper
- Silent drop pattern for unauthorized updates (no error, no reply)

## Environment variables

| Variable | Required | Description |
|---|---|---|
| `TELEGRAM_BOT_TOKEN` | Yes | Bot token from @BotFather |
| `ALLOWED_USERS` | No | Comma-separated numeric user IDs, e.g. `123456,789012`. If unset, all users are permitted. |

## Finding your user ID

Send `/whoami` to the bot — it replies with your numeric Telegram user ID. Add that ID to `ALLOWED_USERS` to restrict the bot to you.

## Extending

Combine with `examples/moderation` to ensure only group admins can invoke moderation commands:

```go
router.Use(allowlistMiddleware(adminIDs))
router.OnCommand("/ban", banHandler)
```

For group-context admin checks (verify the sender is an admin of *that specific group*), use `api.GetChatAdministrators` and check the result dynamically rather than a static ID list.

## Running

```bash
export TELEGRAM_BOT_TOKEN=123456:ABC...
export ALLOWED_USERS=111111,222222
go run ./examples/admin
```
