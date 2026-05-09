# welcome

Greet new chat members as they join a group and log member departures.

## What it shows

- `OnMessageFilter` matching `Message.NewChatMembers` to send a welcome message for each joiner
- `OnMessageFilter` matching `Message.LeftChatMember` to log departures
- `OnMyChatMember` to detect when the bot itself is added to or removed from a group

## Required bot permissions

The bot must be an **admin** in the group (or at minimum have the *"Read Messages"* permission granted to non-admin bots via `setMyDefaultAdminRights`). Without this, Telegram does not forward service messages about member joins and leaves.

## Running

```bash
export TELEGRAM_BOT_TOKEN=123456:ABC...
go run ./examples/welcome
```

Add the bot to a group, then have another user join or leave — the bot will greet joiners and log departures to stdout.
