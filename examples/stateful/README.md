# stateful

Per-user counter with no globals: state lives in a struct passed by closure into handlers.

## Run

```bash
export TELEGRAM_BOT_TOKEN=...
go run ./examples/stateful
```

Send `/count` to the bot in any chat. Each user has an independent counter.
