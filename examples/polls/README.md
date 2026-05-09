# polls

Create polls and tally answers in real time via `OnPollAnswer`.

## What it shows

- `api.SendPoll` with `[]api.InputPollOption` and `IsAnonymous: false`
- `router.OnPollAnswer` to receive vote updates (`PollAnswer.OptionIds`, `PollAnswer.User`)
- Concurrent-safe in-memory tally with `sync.Mutex`

## Commands

| Command | Description |
|---|---|
| `/poll <question>` | Creates a poll with four preset options (A/B/C/D) |
| `/tally <poll_id>` | Shows current vote counts for a poll |

## Notes

- `OnPollAnswer` only fires for **non-anonymous** polls. For anonymous polls, Telegram does not send user identifiers.
- The poll ID is logged when the poll is created; copy it to use with `/tally`.
- Vote tallies are in-memory and reset on restart.

## Running

```bash
export TELEGRAM_BOT_TOKEN=123456:ABC...
go run ./examples/polls
```
