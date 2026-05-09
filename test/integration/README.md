# Integration tests

Live tests against the real Telegram Bot API. Skipped by default.

## Run

```bash
export TELEGRAM_BOT_TOKEN=test_bot_token_here
export TELEGRAM_TEST_CHAT_ID=123456789  # a chat the bot can post in
make integration
```

The suite covers `getMe`, `sendMessage`, and the `setWebhook`/`deleteWebhook` cycle.
