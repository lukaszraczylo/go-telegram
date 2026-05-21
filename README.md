<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="docs/logo-dark.svg">
    <img alt="go-telegram" src="docs/logo-light.svg" width="320">
  </picture>
</p>

<p align="center">
  <strong>Build Telegram bots in Go that just work.</strong><br>
  Type-safe. Batteries included. Always up to date with the latest Bot API.
</p>

<p align="center">
  <a href="https://github.com/lukaszraczylo/go-telegram/actions/workflows/ci.yml"><img src="https://github.com/lukaszraczylo/go-telegram/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://pkg.go.dev/github.com/lukaszraczylo/go-telegram"><img src="https://pkg.go.dev/badge/github.com/lukaszraczylo/go-telegram.svg" alt="Go Reference"></a>
  <a href="go.mod"><img src="https://img.shields.io/github/go-mod/go-version/lukaszraczylo/go-telegram" alt="Go Version"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License: MIT"></a>
</p>

<p align="center">
  Bot API <strong>v10.0</strong> · 176 methods · 301 types · 1428 auto-generated tests
</p>

<p align="center">
  <a href="https://go-telegram.raczylo.com/">Website</a> ·
  <a href="docs/reference/">API Reference</a> ·
  <a href="examples/">Examples</a> ·
  <a href="https://pkg.go.dev/github.com/lukaszraczylo/go-telegram">pkg.go.dev</a>
</p>

---

## Hello, Telegram 👋

```go
bot := client.New(os.Getenv("TELEGRAM_BOT_TOKEN"))
router := dispatch.New(bot)

router.OnCommand("/start", func(c *dispatch.Context, m *api.Message) error {
    _, err := api.SendMessage(c.Ctx, c.Bot, &api.SendMessageParams{
        ChatID: api.ChatIDFromInt(m.Chat.ID),
        Text:   "Hi " + m.From.FirstName + "! 👋",
    })
    return err
})

router.Run(ctx, transport.NewLongPoller(bot))
```

That's a working bot. No magic strings, no `any`, no guessing what fields exist — your editor autocompletes everything.

## Why you'll like it

- 🎯 **No `any`, anywhere.** Telegram's "Integer or String" and "one of N types" unions are real Go types you can `switch` on.
- 🔋 **Batteries included.** Long-poll, webhooks, retries on rate limits, conversation state machines, filters, handler groups — out of the box.
- 🔄 **Always current.** The whole API is generated from Telegram's live docs. New Bot API release? `make regen` and you're done.
- 🪶 **Pluggable everything.** Swap the HTTP client, JSON codec, or storage backend with a one-method interface. No forks.
- 🧪 **Already tested.** 1428 generated tests cover every method × every failure mode (success, API errors, network failures, parse errors, timeouts, missing fields, forbidden, server errors).

## Install

```bash
go get github.com/lukaszraczylo/go-telegram
```

## A complete echo bot

Long-poll, graceful shutdown, retries on Telegram's `429 retry_after`:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/lukaszraczylo/go-telegram/api"
    "github.com/lukaszraczylo/go-telegram/client"
    "github.com/lukaszraczylo/go-telegram/dispatch"
    "github.com/lukaszraczylo/go-telegram/transport"
)

func main() {
    token := os.Getenv("TELEGRAM_BOT_TOKEN")
    if token == "" {
        log.Fatal("TELEGRAM_BOT_TOKEN required")
    }

    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

    bot := client.New(token,
        client.WithHTTPClient(client.NewRetryDoer(client.NewDefaultHTTPDoer())),
    )

    router := dispatch.New(bot)
    router.OnCommand("/start", func(c *dispatch.Context, m *api.Message) error {
        _, err := api.SendMessage(c.Ctx, c.Bot, &api.SendMessageParams{
            ChatID: api.ChatIDFromInt(m.Chat.ID),
            Text:   fmt.Sprintf("Hello %s! Send me anything.", m.From.FirstName),
        })
        return err
    })
    router.OnText(`.+`, func(c *dispatch.Context, m *api.Message) error {
        _, err := api.SendMessage(c.Ctx, c.Bot, &api.SendMessageParams{
            ChatID:          api.ChatIDFromInt(m.Chat.ID),
            Text:            m.Text,
            ReplyParameters: &api.ReplyParameters{MessageID: m.MessageID},
        })
        return err
    })

    if err := router.Run(ctx, transport.NewLongPoller(bot)); err != nil && err != context.Canceled {
        log.Printf("router exited: %v", err)
    }
}
```

## Examples

Run any example: `TELEGRAM_BOT_TOKEN=xxx go run ./examples/<name>`

| Category | Example | What it shows |
|---|---|---|
| **Basics** | [`echo`](examples/echo) | Long-poll echo bot |
| | [`webhook`](examples/webhook) | Webhook server with secret-token verification |
| | [`files`](examples/files) | Upload and download cycle |
| | [`inline`](examples/inline) | Inline-mode results |
| **Conversations & state** | [`conversation`](examples/conversation) | Multi-step state machine with `/cancel` exit |
| | [`stateful`](examples/stateful) | Per-user state via closures |
| | [`callback`](examples/callback) | Inline keyboards and callback query handling |
| | [`pagination`](examples/pagination) | Multi-page inline keyboard |
| **Group management** | [`welcome`](examples/welcome) | Greet new chat members |
| | [`moderation`](examples/moderation) | Kick/ban/mute/warn with permission checks |
| | [`admin`](examples/admin) | Auth middleware allowlist |
| **Advanced** | [`middleware`](examples/middleware) | `Use` chains |
| | [`polls`](examples/polls) | `sendPoll` and answer tally |
| | [`payments`](examples/payments) | Invoice → pre-checkout → success |

## Optional fields

Telegram marks many fields as optional. For optional **scalars** (int, bool, float) we use pointers so you can explicitly send `false` or `0` when the wire format needs to override a chat default. The `api.Ptr` helper keeps that ergonomic:

```go
api.SendMessage(ctx, bot, &api.SendMessageParams{
    ChatID:              api.ChatIDFromInt(chatID),
    Text:                "hi",
    DisableNotification: api.Ptr(true),     // type inferred
})

api.GetUserProfilePhotos(ctx, bot, &api.GetUserProfilePhotosParams{
    UserID: userID,
    Limit:  api.Ptr[int64](5),              // explicit type for untyped literals
})
```

Optional structs and slices are already nullable in Go — no helper needed.

## Reference docs

Full API reference is auto-generated from source comments and lives in [`docs/reference/`](docs/reference/README.md) — browse package by package on GitHub, or read it rendered at [go-telegram.raczylo.com](https://go-telegram.raczylo.com/) and [pkg.go.dev](https://pkg.go.dev/github.com/lukaszraczylo/go-telegram).

## How it works

<details>
<summary>Bot client and pluggable transport</summary>

`client.New` accepts functional options:

```go
bot := client.New(token,
    client.WithHTTPClient(doer),       // any HTTPDoer (one-method interface)
    client.WithCodec(myCodec),         // any Codec (Marshal + Unmarshal)
    client.WithLogger(myLogger),
    client.WithBaseURL("https://..."), // proxy or local Bot API server
)
```

`HTTPDoer` is `Do(*http.Request) (*http.Response, error)` — a plain `*http.Client` satisfies it.
`Codec` is `Marshal(any) ([]byte, error)` + `Unmarshal([]byte, any) error` — the default wraps `goccy/go-json`.

Every API call goes through `client.Call[Req, Resp]`; per-method generated functions are thin wrappers.

</details>

<details>
<summary>Typed unions — no any</summary>

Telegram's docs describe many fields as "Integer or String" or "one of N types". go-telegram turns every one of these into a concrete Go type.

```go
// ChatID: construct from int64 or @username
chatID := api.ChatIDFromInt(123456789)
chatID := api.ChatIDFromString("@mychannel")

// Discriminated unions — 13 interfaces with auto-decode via generated UnmarshalJSON
for _, u := range updates {
    if u.MyChatMember == nil {
        continue
    }
    switch v := u.MyChatMember.OldChatMember.(type) {
    case *api.ChatMemberOwner:
        log.Println("was owner")
    case *api.ChatMemberAdministrator:
        log.Printf("was admin: can_post=%v", v.CanPostMessages)
    }
}
```

Full union list: `ChatMember`, `MessageOrigin`, `ReactionType`, `PaidMedia`, `BackgroundType`, `BackgroundFill`, `ChatBoostSource`, `RevenueWithdrawalState`, `TransactionPartner`, `MenuButton`, `OwnedGift`, `StoryAreaType`, `MaybeInaccessibleMessage`, plus `ChatID`, `MessageOrBool`, and `InputFile`.

</details>

<details>
<summary>Dispatcher, filters, and conversations</summary>

The router dispatches each update in its own goroutine (semaphore-bounded, default 50):

```go
r := dispatch.New(bot, dispatch.WithMaxConcurrency(50))

r.OnCommand("/start", handler)
r.OnText(`^hi (\w+)`, handler)
r.OnCallback(`^like:\d+`, handler)
r.OnInlineQuery(handler)
r.OnMyChatMember(handler)
// + 20 more typed On* methods
```

**Composable filters** — each update type has its own filter package:

```go
import "github.com/lukaszraczylo/go-telegram/dispatch/filters/message"

r.OnMessageFilter(
    message.Command("/admin").And(message.IsReply()),
    handler,
)
```

Filter packages: `message`, `callback`, `inline`, `chatmember`, `chatjoinrequest`, `precheckoutquery`. Combinators: `And`, `Or`, `Not`, `All`, `Any`.

**Conversation state machines** — multi-step flows with pluggable storage:

```go
conv := &conversation.Conversation{
    EntryPoints: []conversation.Step{{
        Filter: dispatch.FilterFunc(func(c *dispatch.Context, u *api.Update) bool {
            return u.Message != nil && u.Message.Text == "/start"
        }),
        Handler: func(c *dispatch.Context, u *api.Update) error {
            // send prompt, advance state
            return conversation.Next("await_name")
        },
    }},
    States: map[conversation.State][]conversation.Step{
        "await_name": {{
            Handler: func(c *dispatch.Context, u *api.Update) error {
                return conversation.End()
            },
        }},
    },
}
router.Use(conv.Dispatch)
```

Key strategies: `KeyByUser`, `KeyByChat`, `KeyByUserAndChat` (default). Default storage: `MemoryStorage` (in-process, concurrency-safe). Implement the `Storage` interface for Redis or any other backend.

</details>

<details>
<summary>Errors and retry middleware</summary>

Wrap the default HTTP doer with `RetryDoer` for production:

```go
bot := client.New(token,
    client.WithHTTPClient(
        client.NewRetryDoer(
            client.NewDefaultHTTPDoer(),
            client.WithMaxAttempts(5),
            client.WithBaseBackoff(500*time.Millisecond),
        ),
    ),
)
```

`RetryDoer` retries on 429, 5xx, and transient network errors. On a 429 it reads `retry_after` from Telegram's response body and waits exactly that long — overriding any backoff calculation. Request bodies are buffered and replayed across attempts.

Sentinel errors for `errors.Is` checks: `client.ErrForbidden`, `client.ErrNotFound`, `client.ErrUnauthorized`.

</details>

<details>
<summary>Handler groups and named handlers</summary>

Priority-ordered groups with flow control signals:

```go
// Group 0 runs first — return EndGroups to stop, ContinueGroups to continue
r.Group(0).OnText(`.*`, authMiddleware)
r.Group(1).OnText(`.*`, businessHandler)
```

Named handlers — register and replace at runtime:

```go
named := dispatch.NewNamedHandlers[*api.Message]()
named.Set("main", myHandler)
r.OnCommand("/cmd", named.Handler())
// later: named.Set("main", updatedHandler)
```

</details>

## Benchmarks

Apples-to-apples micro-benchmarks against the five most-starred Go Telegram libraries (`go-telegram-bot-api`, `telebot.v3`, `go-telegram/bot`, `telego`, `echotron`) live under [`test/benchmarks/`](test/benchmarks/) as a separate Go module.

<details>
<summary>Results — Apple M4 Max · darwin/arm64 · go1.26.2</summary>

| Path | Fastest | Our position |
|------|---------|--------------|
| Webhook decode (small Update) | **ours** — 1.83 µs / 11 allocs | 1st of 6 |
| Large Update unmarshal (unions + reply markup) | **ours** — 6.73 µs / 34 allocs | 1st of 6 |
| `sendMessage` round-trip — `net/http` default | telego — 35.8 µs / 48 allocs | 2nd of 5 (102 allocs) |
| `sendMessage` round-trip — opt-in `fasthttp` | telego — 48 allocs | within 8 of telego (56 allocs) |
| Dispatcher routing (20 handlers, last matches) | **ours** — 98 ns / 3 allocs | 1st of 3 |

Opt into fasthttp for high-throughput bots: `client.WithHTTPClient(client.NewFastHTTPDoer())`. Trade-off: HTTP/1.1 only, no `RoundTripper` middleware composition.

Full tables, caveats, and reproduction steps: **[`docs/benchmarks/2026-05-10-comparison.md`](docs/benchmarks/2026-05-10-comparison.md)**.

</details>

## Keeping up with Telegram

When Telegram ships a new Bot API version, regenerating the whole library is one command:

```bash
make snapshot   # grab the latest HTML from core.telegram.org
make regen      # scrape → audit → emit Go → run tests → regenerate docs
```

The audit tool checks for `any`-typed escapes, surprise `bool` returns, and signature drift. CI runs it on every PR, and a weekly workflow opens an auto-PR with regenerated code so a new Bot API version never sits longer than a week.

If something in Telegram's docs trips up the scraper, add an override to `internal/spec/overrides.json`. The audit will tell you what to put there.

## Testing

Mock the one-method `HTTPDoer` interface to test handlers in isolation — no test server needed:

```go
type fakeDoer struct{ body string }
func (f fakeDoer) Do(*http.Request) (*http.Response, error) {
    return &http.Response{
        StatusCode: 200,
        Body:       io.NopCloser(strings.NewReader(f.body)),
    }, nil
}

bot := client.New("token", client.WithHTTPClient(fakeDoer{
    body: `{"ok":true,"result":{"message_id":1,"date":0,"chat":{"id":1,"type":"private"}}}`,
}))
```

The library's own generated test suite (`api/methods_gen_test.go`) covers 176 methods × 8 scenarios each: Success, APIError, NetworkError, ParseError, ContextCanceled, MissingRequiredFields, Forbidden, ServerError.

## Telemetry

On the **first call to `client.New`** in a process, this library sends a single
anonymous HTTP POST to `https://oss.raczylo.com/v1/ping` containing exactly
this body:

```json
{ "project": "go-telegram", "version": "0.7.11", "ts": 1747782200 }
```

This helps us see approximate adoption and version spread. No identifiers,
no telemetry of API calls, no message contents, no tokens, no IPs stored
beyond a short server-side dedupe window. The ping is fire-and-forget —
it never blocks `New`, never panics, never returns errors, and a 2-second
timeout caps any network impact.

Telemetry source: [`client/telemetry.go`](client/telemetry.go) and the
upstream library [`github.com/lukaszraczylo/oss-telemetry`](https://github.com/lukaszraczylo/oss-telemetry).

### Opting out

Any one of these turns it off (case-insensitive truthy values
`1`, `true`, `yes`, `on`):

| Mechanism                | How                                          |
| ------------------------ | -------------------------------------------- |
| Universal opt-out        | `DO_NOT_TRACK=1`                             |
| Library-wide opt-out     | `OSS_TELEMETRY_DISABLED=1`                   |
| Per-library opt-out      | `GO_TELEGRAM_DISABLE_TELEMETRY=1`            |
| Programmatic             | `osstelemetry.Disable()` before `client.New` |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

MIT
