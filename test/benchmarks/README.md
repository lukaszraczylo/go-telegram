# Cross-library benchmarks

Apples-to-apples micro-benchmarks comparing `lukaszraczylo/go-telegram` against the five most-starred Go Telegram libraries:

- `github.com/go-telegram-bot-api/telegram-bot-api/v5`
- `gopkg.in/telebot.v3` (tucnak)
- `github.com/go-telegram/bot`
- `github.com/mymmrac/telego`
- `github.com/NicoNex/echotron/v3`

Lives in its own Go module so competitor dependencies don't leak into the main repo's `go.mod`.

## Run

```bash
go test -count=10 -bench=. -benchmem | tee results/raw.txt
go install golang.org/x/perf/cmd/benchstat@latest    # one-time
benchstat results/raw.txt > results/benchstat.txt
```

## Hot paths covered

| File | Path |
|------|------|
| `webhook_bench_test.go` | Decode small text-message Update from JSON |
| `unmarshal_bench_test.go` | Decode large Update (entities + reply markup + photo array) |
| `call_bench_test.go` | `sendMessage` round-trip against `httptest.Server` |
| `dispatch_bench_test.go` | Route an Update through 20 registered handlers (worst-case match) |

## Fixtures

`shared/fixtures.go` defines the JSON payloads and the mock HTTP server. Every library decodes the same bytes; every round-trip hits the same canned response.

## Latest results

See [`../../docs/benchmarks/2026-05-10-comparison.md`](../../docs/benchmarks/2026-05-10-comparison.md) for the rendered comparison.
