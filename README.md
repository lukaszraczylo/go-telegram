# go-telegram

> A fully-generated, strongly-typed Go client for the Telegram Bot API — no `any`, no guessing.

[![CI](https://github.com/lukaszraczylo/go-telegram/actions/workflows/ci.yml/badge.svg)](https://github.com/lukaszraczylo/go-telegram/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/lukaszraczylo/go-telegram.svg)](https://pkg.go.dev/github.com/lukaszraczylo/go-telegram)
[![Go Version](https://img.shields.io/github/go-mod/go-version/lukaszraczylo/go-telegram)](go.mod)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

> Bot API **v10.0** · 176 methods · 301 types · 1428 auto-generated tests

Most Telegram bot libraries expose Telegram's "Integer or String" fields as `interface{}` or `any`. Every union type in go-telegram is a real Go type with compile-time safety and auto-decoding. The entire API surface is code-generated from a committed HTML snapshot of the live Telegram docs — regenerating picks up new Bot API versions in one command, with a self-verifying pipeline that catches regressions before they ship.

```go
bot := client.New(os.Getenv("TELEGRAM_BOT_TOKEN"),
    client.WithHTTPClient(client.NewRetryDoer(client.NewDefaultHTTPDoer())),
)

router := dispatch.New(bot)
router.OnCommand("/start", func(c *dispatch.Context, m *api.Message) error {
    _, err := api.SendMessage(c.Ctx, c.Bot, &api.SendMessageParams{
        ChatID: api.ChatIDFromInt(m.Chat.ID),
        Text:   "Hello! Send me anything to echo.",
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

ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()
router.Run(ctx, transport.NewLongPoller(bot))
```

## Why go-telegram

| Feature | What it means for you |
|---|---|
| **Typed unions** | `ChatID`, `MessageOrBool`, `InputFile`, and 13 discriminated-union interfaces give you `switch v.(type)` instead of runtime panics |
| **Full Bot API v10.0** | 176 methods and 301 types — all generated, none hand-written, nothing missing |
| **Self-verifying codegen** | `make snapshot && make regen` regenerates everything and runs 1428 tests; any regression fails the pipeline |
| **Pluggable transport + codec** | `HTTPDoer` and `Codec` are one-method interfaces — swap in fasthttp, sonic, or your test fake without forking |
| **Retry middleware** | `RetryDoer` honours Telegram's `retry_after`, backs off on 5xx, replays request bodies |
| **Composable dispatcher** | Per-update goroutine pool (default 50), filter combinators (`And`/`Or`/`Not`), conversation state machines, named handlers |

## Quickstart

```bash
go get github.com/lukaszraczylo/go-telegram
```

Full echo bot — long-poll, graceful shutdown, retry on 429:

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

## Concepts

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

## Codegen pipeline

The full API surface in `api/*.gen.go` is generated from a committed HTML snapshot of `core.telegram.org/bots/api`:

```bash
make snapshot   # fetch and commit latest HTML from core.telegram.org
make regen      # scrape → audit → emit Go code → run generated tests
go test -race ./...
```

`make regen` is self-verifying. The audit tool (`cmd/audit`) checks:

- `any`-typed fields or returns that escaped the union machinery
- Methods returning `bool` not on the approved list (`internal/spec/overrides.json`)
- Signature drift vs HEAD's IR (added/removed/changed return types)

Exit codes: 0 clean · 1 fallback · 2 drift · 3 invalid. CI runs the audit on every PR. A weekly `regen.yml` workflow opens a PR with regenerated code and the audit summary in the body.

To track a new Bot API release: run `make snapshot && make regen`, review the audit output, update `internal/spec/overrides.json` for any newly unparseable methods, and submit a PR.

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

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

MIT
