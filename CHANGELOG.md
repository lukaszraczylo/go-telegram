# Changelog

All notable changes to this project will be documented in this file. The format follows [Keep a Changelog](https://keepachangelog.com/) and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [1.0.0] - 2026-05-09

Initial public release. Built against Telegram Bot API v10.0 (176 methods, 301 types).

### Library surface
- `client.Bot` with pluggable `HTTPDoer`, `Codec` (default `github.com/goccy/go-json`), and `Logger` interfaces.
- Generic `client.Call[Req, Resp]` and `client.CallRaw[Req]` helpers — every API method funnels through one of these.
- `client.RetryDoer` middleware — exponential backoff with jitter; honours Telegram's `retry_after`; replays request bodies across attempts. Configurable via `RetryOption` functions (`WithMaxAttempts`, `WithBase`, `WithMax`, `WithFactor`, `WithJitter`).
- Typed errors: `*APIError` (sentinel-mapped: `ErrUnauthorized`, `ErrForbidden`, `ErrTooManyRequests`, `ErrChatNotFound`, `ErrMessageNotModified`, `ErrBadRequest`, `ErrUserNotFound`, `ErrMessageNotFound`), `*NetworkError`, `*ParseError`. Bot tokens redacted from error messages.

### Update delivery
- `transport.LongPoller` — getUpdates loop with `ExponentialBackoff`, retry-after honouring, graceful shutdown via `Stop`.
- `transport.WebhookServer` — `http.Handler` + `ListenAndServe`; secret-token verification (constant-time); 1 MiB body cap; in-flight handler tracking via `WaitGroup`.

### Dispatcher
- Generic `dispatch.Handler[T]` and `dispatch.Middleware[T]`.
- 21 typed handler registrations: `OnCommand`, `OnText`, `OnCallback`, `OnInlineQuery`, `OnEditedMessage`, `OnChannelPost`, `OnEditedChannelPost`, `OnMyChatMember`, `OnChatMember`, `OnChatJoinRequest`, `OnPreCheckoutQuery`, `OnShippingQuery`, `OnPoll`, `OnPollAnswer`, `OnChosenInlineResult`, `OnMessageReaction`, `OnMessageReactionCount`, `OnChatBoost`, `OnRemovedChatBoost`, `OnBusinessConnection`, `OnPurchasedPaidMedia`. Filter variants available for message, callback, inline-query, chat-member, chat-join-request, and pre-checkout-query types.
- Composable filters in `dispatch/filters/{message,callback,inline,chatmember,chatjoinrequest,precheckoutquery}` packages with `And`/`Or`/`Not`/`All`/`Any` combinators and 20+ filter helpers.
- `dispatch/conversation` — multi-step state machines with `Storage` interface (`MemoryStorage` default), pluggable `KeyStrategy` (`KeyByUser`, `KeyByChat`, `KeyByUserAndChat`), entry/state/exit/fallback steps, `AllowReEntry`. `Next(state)` and `End()` sentinel errors drive transitions. `Conversation.Dispatch` integrates as middleware via `router.Use`.
- Per-update goroutine pool (default 50; configurable via `WithMaxConcurrency`). Pass `0` for serial dispatch.
- Handler groups with `EndGroups`/`ContinueGroups` flow control.
- `NamedHandlers[T]` for runtime registration and replacement.
- Panic-recovery middleware + automatic handler-error logging registered by default.

### Generated API
- Full Bot API v10.0 surface in `api/*.gen.go` — 176 methods, 301 types, regenerated from a committed HTML snapshot of `core.telegram.org/bots/api`.
- Strongly typed unions: `ChatID` (Integer-or-String), `*InputFile` (InputFile-or-String), `MessageOrBool` (Message-or-True returns).
- 13 discriminated unions with auto-decode: `ChatMember`, `MessageOrigin`, `ReactionType`, `PaidMedia`, `BackgroundType`, `BackgroundFill`, `ChatBoostSource`, `RevenueWithdrawalState`, `TransactionPartner`, `MenuButton`, `OwnedGift`, `StoryAreaType`, `MaybeInaccessibleMessage`. Parent structs containing union-typed fields auto-decode via generated `UnmarshalJSON`.
- Sealed-interface return types use `client.CallRaw` + `Unmarshal<Name>` dispatch (e.g., `GetChatMember` returns `ChatMember` interface, decoded into the correct concrete variant).

### Runtime helpers
- `api.MeCache` — concurrent-safe `getMe` cache; zero-value safe.
- `api.DownloadFile` / `api.DownloadFileByPath` — fetch file contents from the Telegram CDN.
- `(*Message).GetSender()` — unifies `From`, `SenderChat`, and anonymous-admin fields into a `*Sender`.

### Codegen pipeline (`cmd/scrape`, `cmd/genapi`, `cmd/audit`)
- Two-stage codegen: HTML → IR (`internal/spec/api.json`) → Go.
- `internal/spec/overrides.json` pins specific method returns/field types when scraper regex does not match a particular doc phrasing.
- `cmd/audit` reports any-typed fields, fallback `bool` returns, and signature drift vs HEAD's IR. Exit codes: 0 clean, 1 fallback, 2 drift, 3 invalid.
- `make regen` is self-verifying: clean → scrape → audit → emit (code + 1428 tests) → run tests.
- Auto-generated tests cover 8 scenarios per method (1428 total): Success, APIError 429, NetworkError, ParseError, ContextCanceled, MissingRequiredFields (400 + ErrBadRequest), Forbidden (403 + ErrForbidden), ServerError (500 + IsRetryable).

### Tooling
- `.github/workflows/ci.yml` — Go matrix 1.23/1.24, vet, staticcheck, govulncheck, gosec, race tests, codegen-clean check, audit + drift detection.
- `.github/workflows/regen.yml` — weekly cron + workflow_dispatch; scrapes live API, regenerates, runs tests, opens auto-PR with audit summary in body.
- `.github/workflows/release.yml` — semver-generator-driven version bump; dual tagging (library SemVer + `bot-api-v<X.Y>` marker); GoReleaser.
- `.goreleaser.yaml` — ships `tg-scrape`, `tg-genapi`, `tg-audit` binaries for Linux/macOS/Windows × amd64/arm64.

### Examples (14)
echo, webhook, callback, conversation, files, inline, middleware, stateful, welcome, moderation, polls, payments, pagination, admin.
