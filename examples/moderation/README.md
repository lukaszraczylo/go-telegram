# moderation

Group moderation commands: `/kick`, `/ban`, `/mute`, `/warn`, `/unwarn`.

## What it shows

- `OnCommand` for each moderation action
- `api.BanChatMember` / `api.UnbanChatMember` for kick and ban
- `api.RestrictChatMember` with `ChatPermissions` for muting
- `errors.Is(err, client.ErrForbidden)` to surface missing-permissions errors cleanly
- In-memory warn counter via `sync.Map` (auto-bans at 3 warnings)

## Required bot permissions

The bot must be an **admin** in the group with **"can ban users"** and **"can restrict members"** permissions. Without those rights, commands will reply with a friendly error message instead of crashing.

## Usage

All commands work by **replying** to a target user's message:

```
/kick   — remove from group (can rejoin)
/ban    — permanent ban
/mute   — silence for 1 hour
/warn   — issue a warning (3 warnings = auto-ban)
/unwarn — remove the last warning
```

## Production notes

- The warn counter is in-memory and lost on restart. For production, back it with Redis or a database.
- Consider adding an admin check (see `examples/admin`) so only group admins can invoke these commands.

## Running

```bash
export TELEGRAM_BOT_TOKEN=123456:ABC...
go run ./examples/moderation
```
