# Benchmarks vs top 5 Go Telegram libraries

**Date:** 2026-05-10
**Environment:** Apple M4 Max · darwin/arm64 · `go1.26.2`
**Methodology:** `go test -count=10 -bench=. -benchmem`, summarised with `benchstat` (golang.org/x/perf)
**Source:** [`test/benchmarks/`](../../test/benchmarks/) · raw output: [`results/raw.txt`](../../test/benchmarks/results/raw.txt) · benchstat: [`results/benchstat.txt`](../../test/benchmarks/results/benchstat.txt)

## Libraries

| Lib | Module |
|-----|--------|
| **ours** | `github.com/lukaszraczylo/go-telegram` (this repo) |
| gotba | `github.com/go-telegram-bot-api/telegram-bot-api/v5` |
| telebot | `gopkg.in/telebot.v3` (tucnak) |
| gobot | `github.com/go-telegram/bot` |
| telego | `github.com/mymmrac/telego` |
| echotron | `github.com/NicoNex/echotron/v3` |

## TL;DR

- **Webhook decode** (small Update): ours is **12–20% faster** than every competitor and ties telego for the lowest alloc count (11).
- **Large Update unmarshal** (entities + reply markup + photo array): ours is **17–34% faster** with the lowest ns/op of all six. telego edges us on alloc count (31 vs 34) at the cost of ~17% more time.
- **API call round-trip** (mock HTTP server): telego wins (35.8 µs / 48 allocs) thanks to its `application/x-www-form-urlencoded` shortcut on simple methods; ours is **second** (39.8 µs / 102 allocs) and beats gotba, telebot, gobot.
- **Dispatcher routing** (20 handlers, last matches): ours is **2.5–2.8× faster than telebot and gobot** (98 ns vs 271 / 246 ns).

## How to read these numbers

- One machine, single workload, fixtures defined in [`shared/fixtures.go`](../../test/benchmarks/shared/fixtures.go). Re-run on your hardware before drawing conclusions.
- Codecs differ across libs (we use `goccy/go-json`; most competitors use stdlib `encoding/json`). Codec choice is part of the library's value prop, so we benchmark each library as it ships, not in some artificial common-codec mode.
- "Equivalent code path" was chosen via each library's idiomatic public API for the same logical operation. The exact code is in the bench files alongside each `BenchmarkXxx_<lib>` function — read them.

---

## 1. Webhook decode — small Update (text message)

Decode `shared.SmallUpdateJSON` into the library's typed `Update` struct.

| Lib | sec/op | B/op | allocs/op |
|-----|--------|------|-----------|
| **ours** | **1.832 µs ±4%** | 2.180 KiB | **11** |
| gotba | 2.082 µs ±0% | 1.461 KiB | 17 |
| telebot | 2.194 µs ±1% | 1.773 KiB | 17 |
| gobot | 2.082 µs ±1% | 1.789 KiB | 16 |
| telego | 2.143 µs ±2% | 3.058 KiB | **11** |
| echotron | 2.039 µs ±1% | 1.680 KiB | 16 |

**Notes.** We use slightly more bytes because typed unions and the typed `[]UpdateType` allocate richer Go values. We win on time and tie telego on alloc count.

## 2. Large Update unmarshal — entities + reply markup + photo array

Decode `shared.LargeUpdateJSON` (text + 3 entities + 2x3 inline keyboard + 3-size photo array). Stresses each library's union/discriminator decoding.

| Lib | sec/op | B/op | allocs/op |
|-----|--------|------|-----------|
| **ours** | **6.726 µs ±1%** | 5.875 KiB | 34 |
| gotba | 8.066 µs ±1% | 3.438 KiB | 56 |
| telebot | 10.190 µs ±1% | 5.594 KiB | 60 |
| gobot | 8.231 µs ±1% | 4.703 KiB | 50 |
| telego | 7.849 µs ±2% | 6.600 KiB | **31** |
| echotron | 8.123 µs ±1% | 4.219 KiB | 56 |

**Notes.** Despite the typed-union model giving us richer Go values per decode, we still produce them faster than every competitor. telego edges us by 3 allocs but pays 17% more time.

## 3. API call round-trip — `sendMessage` against a mock HTTP server

Build params → POST to local `httptest.Server` returning `{"ok":true,"result":Message}` → decode response.

| Lib | sec/op | B/op | allocs/op |
|-----|--------|------|-----------|
| ours | 39.83 µs ±4% | 11.09 KiB | 102 |
| gotba | 42.03 µs ±4% | 10.97 KiB | 125 |
| telebot | 43.41 µs ±1% | 13.15 KiB | 139 |
| gobot | 61.19 µs ±1% | 13.50 KiB | 176 |
| **telego** | **35.84 µs ±1%** | **6.547 KiB** | **48** |
| echotron | *skipped — see below* | — | — |

**Notes.**
- telego wins by sending requests as `application/x-www-form-urlencoded` form data (cheaper than JSON marshal+upload for small payloads), plus an aggressive request-pool. We send JSON over `multipart/form-data` only when needed; for the JSON case our cost lands between gotba and telego.
- Our request path runs through a manually-constructed `*http.Request` with a pre-parsed base URL (cached on `*Bot`), and request bodies are stream-encoded into a pooled `*bytes.Buffer` via the optional `BodyEncoder` codec extension. Together those skip the `url.Parse` + `*http.Request` bookkeeping that `http.NewRequestWithContext` runs on every call.
- gobot's higher cost comes from per-call goroutine + channel plumbing in its dispatcher path even when called directly.
- **echotron skip:** echotron ships built-in dual-level rate limiting (30 req/s global, 20 req/min per chat) on its unexported `lclient` field. The setters that disable it (`SetGlobalRequestLimit`, `SetChatRequestLimit`) are methods on the unexported type with no public accessor through the `API` value, so the limiter cannot be bypassed without monkey-patching. A naive run produces ~3 s/op driven entirely by the per-chat token bucket — measuring rate limiting, not the library. We skip rather than publish a misleading number. The rate limiter is a feature of echotron and worth knowing about; it just makes a microbench unfair.

## 4. Dispatcher routing — 20 handlers, last one matches

Register 20 command handlers (`/cmd0` … `/cmd19`); feed an update matching `/cmd19` so the bench measures worst-case filter chain traversal.

| Lib | sec/op | B/op | allocs/op |
|-----|--------|------|-----------|
| **ours** | **98.46 ns ±2%** | 128 B | 3 |
| telebot | 270.9 ns ±2% | 678 B | 5 |
| gobot | 246.1 ns ±1% | **48 B** | **1** |

**Notes.** We dispatch ~2.5× faster than telebot and gobot. gobot's single allocation is impressive but its routing decision is slower. telebot's higher cost reflects its richer per-update `Context` construction.

**Coverage caveats.**
- **gotba** ships no built-in dispatcher; users route via a manual `switch` on `Update` fields. Benchmarking that against framework-based dispatchers would be apples-to-oranges, so it's omitted.
- **telego** routes via a buffered channel + goroutine pool inside `telegohandler.BotHandler`. There is no public sync entry point, so the bench would conflate channel + goroutine overhead with routing cost.
- **echotron** uses a chat-ID-keyed `Dispatcher` that fans out to per-chat `Bot` instances — a different paradigm (stateful per-chat bot loop), not directly comparable to "match this update against N handlers".

---

## How to reproduce

```bash
cd test/benchmarks
go test -count=10 -bench=. -benchmem | tee results/raw.txt
benchstat results/raw.txt > results/benchstat.txt
```

Install `benchstat` if missing: `go install golang.org/x/perf/cmd/benchstat@latest`.

## Bench code

All bench source lives under [`test/benchmarks/`](../../test/benchmarks/) as a separate Go module so competitor dependencies stay out of the root `go.mod`. The fixtures (the JSON each library decodes, the mock HTTP server) are in [`shared/fixtures.go`](../../test/benchmarks/shared/fixtures.go) — every library decodes the same bytes.
