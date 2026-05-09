# files

Bot that downloads documents users send and re-uploads them. Demonstrates `api.DownloadFile`, `api.SendDocument` with `*InputFile`, and middleware-based dispatch on document updates.

## Run

```bash
export TELEGRAM_BOT_TOKEN=...
go run ./examples/files
```

Send any document to the bot.
