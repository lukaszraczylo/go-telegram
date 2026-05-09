# Contributing

Thanks for your interest in go-telegram. The library mixes hand-written and generated code; this guide explains how to update each.

## Project layout

- **`client/`** — hand-written Bot client, generic Call helper, error taxonomy, retry middleware. Stable; rarely changes.
- **`transport/`** — long-poll and webhook updaters. Hand-written.
- **`dispatch/`** — typed router with command/text/callback matchers. Hand-written.
- **`api/`** — generated types and method wrappers (`*.gen.go`) plus runtime helpers (`runtime.go`, `download.go`, `me.go`).
- **`internal/spec/`** — IR types + the committed `api.json` snapshot of the Telegram Bot API.
- **`cmd/scrape/`** — HTML scraper that produces `internal/spec/api.json`.
- **`cmd/genapi/`** — emitter that consumes `api.json` and renders `api/*.gen.go`.

## Workflows

### Updating to a newer Telegram Bot API version

```bash
make snapshot                          # fetch latest HTML from core.telegram.org
make regen                              # scrape + emit
go test -race ./...                     # verify
```

If the live page introduces a phrasing the scraper doesn't recognise, you'll see methods falling back to `bool` returns or struct fields typed `any`. Check the audit script in `cmd/scrape/method_test.go` and add new patterns to `cmd/scrape/method.go` and / or `cmd/scrape/table.go`. Then `go test -update ./cmd/scrape/...` to refresh the small-fixture golden, and re-run `make regen`.

### Adding a new union for auto-decode

If Telegram introduces a new discriminated union type (similar to `ChatMember`):

1. Add an entry to `knownDiscriminators` in `cmd/genapi/emitter.go`.
2. Run `make regen`.
3. The emitter will produce `UnmarshalXxx` for the union and per-struct `UnmarshalJSON` for any field referencing it.

### Updating runtime helpers

Edit `api/runtime.go`, `api/download.go`, `api/me.go`, or any of `client/*.go`, `transport/*.go`, `dispatch/*.go`. Add tests for new functionality. CI runs `go test -race ./...`, `go vet`, `staticcheck`, and the codegen-clean check (which asserts the committed `api/` matches what the pipeline produces from the committed snapshot).

### Conventions

- Doc comments on every exported symbol — generated types carry verbatim Telegram prose.
- No `//nolint` directives anywhere; if the linter complains, fix the code or update `.golangci.yml`.
- No reordering of struct fields for `fieldalignment` — JSON field order tracks the spec for diff readability.
- TDD where practical: failing test, then implementation, then commit.
- Conventional Commits style for messages: `feat(...):`, `fix(...):`, `docs(...):`, `chore(...):`, `refactor(...):`, `test(...):`.

## Running locally

```bash
make test              # unit tests
make test-race         # with race detector (CI default)
make lint              # vet + staticcheck
make integration       # live API smoke tests (requires TELEGRAM_BOT_TOKEN)
```

The codegen tooling in `cmd/scrape` pulls `golang.org/x/net/html`. The runtime library packages depend only on the standard library plus `stretchr/testify` (test-only).

## Releasing

1. Bump version in `CHANGELOG.md`.
2. Tag with `git tag -a v0.x.0 -m "summary"` (no leading 'v' alone — use the full SemVer triple).
3. `git push --tags`.

(There is no GoReleaser config yet; releases are tag-only and `go install` works against tags.)

## Reporting issues

File issues on the GitHub repository with:
- The Telegram Bot API method involved (if applicable).
- A minimal reproduction (mocked HTTP transport is fine).
- The library version (`go list -m github.com/lukaszraczylo/go-telegram`).
