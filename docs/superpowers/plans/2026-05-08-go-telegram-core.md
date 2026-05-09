# go-telegram Core Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the hand-written core of `github.com/lukaszraczylo/go-telegram` — a pluggable Bot client, long-poll + webhook transports, and a typed dispatcher — with a hand-coded `api/` slice covering the ~8 methods needed for echo and webhook example bots, fully tested via mocked HTTP transport.

**Architecture:** Three hand-written packages (`client`, `transport`, `dispatch`) plus a hand-coded slice of `api/` whose conventions match what the codegen pipeline (Plan 2) will later produce. Pluggable HTTP via `HTTPDoer` interface, pluggable JSON via `Codec` interface, all generated method calls funnel through a single generic `call[Req,Resp]` helper. Updates flow `Updater → chan Update → Router → Handler[T]`.

**Tech Stack:** Go 1.23+, standard library only for production code, `github.com/stretchr/testify` for tests. No third-party HTTP, JSON, or logging libraries in core (users plug their own).

**Reference:** [Design spec](../specs/2026-05-08-go-telegram-design.md). Read it before starting if you are unfamiliar with the project.

---

## Conventions for every task

- Work from repo root (`/Users/nvm/Documents/projects/private/go-telegram`).
- Run `go test ./...` after every implementation step that adds tests.
- Commit at the end of every task with the message shown.
- Use `gofmt` (run automatically on save). All files must `go vet` clean.
- For hand-written API types and methods (Task 11–12), follow the convention spec'd here so Plan 2 codegen output is byte-identical:
  - Optional fields: pointer (`*int64`) or slice/map with `omitempty`.
  - Required fields: bare type, no `omitempty`.
  - Doc comment on every exported symbol; verbatim Telegram doc prose preferred.
  - Field tag pattern: `json:"snake_case_name,omitempty"` (omit `omitempty` only on required scalars).
  - Method param structs named `<MethodName>Params` (e.g. `SendMessageParams`).
  - Method wrappers on `*Bot`, signature `func (b *Bot) MethodName(ctx context.Context, p *MethodNameParams) (*ReturnType, error)` (or `(ReturnType, error)` for non-pointer returns).

---

## Task 1 — Repository foundation

**Files:**
- Create: `go.mod`
- Create: `LICENSE`
- Create: `.gitignore`
- Create: `Makefile`
- Create: `README.md`
- Create: `doc.go` (package-level godoc anchor at module root)

- [ ] **Step 1: Initialise Go module**

```bash
go mod init github.com/lukaszraczylo/go-telegram
```

- [ ] **Step 2: Write `LICENSE`**

```
MIT License

Copyright (c) 2026 Lukasz Raczylo

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

- [ ] **Step 3: Write `.gitignore`**

```gitignore
# Binaries
/bin/
*.exe
*.dll
*.so
*.dylib

# Test artifacts
*.test
*.out
coverage.out
coverage.html

# IDE
.idea/
.vscode/
*.swp
*~

# OS
.DS_Store

# Local env
.env
.env.local
```

- [ ] **Step 4: Write `Makefile`** (regen targets are stubs in Plan 1; Plan 2 fills them in)

```makefile
.PHONY: test test-race lint vet integration regen snapshot regen-from-fixture test-update-golden help

GO ?= go

help:
	@echo "Targets:"
	@echo "  test                 - run unit tests"
	@echo "  test-race            - run unit tests with race detector"
	@echo "  lint                 - go vet + staticcheck"
	@echo "  integration          - run integration suite (requires TELEGRAM_BOT_TOKEN)"
	@echo "  snapshot             - capture HTML snapshot from live API (Plan 2)"
	@echo "  regen                - regenerate api/ from latest snapshot (Plan 2)"
	@echo "  regen-from-fixture   - deterministic regen from pinned fixture (Plan 2)"
	@echo "  test-update-golden   - refresh golden test fixtures (Plan 2)"

test:
	$(GO) test ./...

test-race:
	$(GO) test -race ./...

vet:
	$(GO) vet ./...

lint: vet
	@which staticcheck > /dev/null || (echo "install staticcheck: go install honnef.co/go/tools/cmd/staticcheck@latest" && exit 1)
	staticcheck ./...

integration:
	$(GO) test -tags=integration -v ./test/integration/...

snapshot:
	@echo "Plan 2 — not yet implemented"
	@exit 1

regen:
	@echo "Plan 2 — not yet implemented"
	@exit 1

regen-from-fixture:
	@echo "Plan 2 — not yet implemented"
	@exit 1

test-update-golden:
	@echo "Plan 2 — not yet implemented"
	@exit 1
```

- [ ] **Step 5: Write `README.md`** (stub — full content in Task 22)

```markdown
# go-telegram

A Go library for the Telegram Bot API with pluggable HTTP transport, pluggable JSON codec, long-poll and webhook delivery, and a typed dispatcher.

> Status: in active development. See `docs/superpowers/plans/` for the implementation roadmap.

## Install

```bash
go get github.com/lukaszraczylo/go-telegram
```

## License

MIT — see [LICENSE](LICENSE).
```

- [ ] **Step 6: Write `doc.go`**

```go
// Package gotelegram is the module root.
//
// The public API lives in the api, client, transport, and dispatch packages.
// See https://github.com/lukaszraczylo/go-telegram for documentation.
package gotelegram
```

- [ ] **Step 7: Verify build**

Run: `go build ./...`
Expected: success (no source files yet beyond `doc.go`, but module resolves cleanly).

- [ ] **Step 8: Commit**

```bash
git add -A
git commit -m "chore: scaffold repository (module, license, makefile, README stub)"
```

---

## Task 2 — CI workflow

**Files:**
- Create: `.github/workflows/ci.yml`

- [ ] **Step 1: Write the CI workflow**

```yaml
name: ci

on:
  push:
    branches: [main]
  pull_request:

permissions:
  contents: read

jobs:
  test:
    runs-on: ubuntu-latest
    strategy:
      fail-fast: false
      matrix:
        go-version: ['1.23.x', '1.24.x']
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ matrix.go-version }}
          check-latest: true

      - name: Cache modules
        uses: actions/cache@v4
        with:
          path: |
            ~/go/pkg/mod
            ~/.cache/go-build
          key: ${{ runner.os }}-go-${{ matrix.go-version }}-${{ hashFiles('**/go.sum') }}

      - name: Install staticcheck
        run: go install honnef.co/go/tools/cmd/staticcheck@latest

      - name: Vet
        run: go vet ./...

      - name: Staticcheck
        run: staticcheck ./...

      - name: Test (race + cover)
        run: go test -race -coverprofile=coverage.out ./...

      - name: Upload coverage
        if: matrix.go-version == '1.24.x'
        uses: actions/upload-artifact@v4
        with:
          name: coverage
          path: coverage.out
```

- [ ] **Step 2: Commit**

```bash
git add .github/
git commit -m "ci: add Go test + lint workflow"
```

---

## Task 3 — IR types (`internal/spec/ir.go`)

The IR is finalised now even though Plan 1 does not exercise it; defining it here means client code structures match the eventual codegen output, and Plan 2 has zero design work to do.

**Files:**
- Create: `internal/spec/ir.go`
- Create: `internal/spec/ir_test.go`

- [ ] **Step 1: Write the failing test**

`internal/spec/ir_test.go`:

```go
package spec

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAPIRoundTripJSON(t *testing.T) {
	in := API{
		Version: "7.10",
		Types: []TypeDecl{{
			Name: "User",
			Doc:  "This object represents a Telegram user or bot.",
			Fields: []Field{
				{Name: "ID", JSONName: "id", Type: TypeRef{Kind: KindPrimitive, Name: "int64"}, Required: true, Doc: "Unique identifier."},
				{Name: "Username", JSONName: "username", Type: TypeRef{Kind: KindPrimitive, Name: "string"}, Required: false, Doc: "Optional username."},
			},
		}},
		Methods: []MethodDecl{{
			Name:    "getMe",
			Doc:     "A simple method for testing your bot's authentication token.",
			Returns: TypeRef{Kind: KindNamed, Name: "User"},
		}},
	}

	data, err := json.MarshalIndent(in, "", "  ")
	require.NoError(t, err)

	var out API
	require.NoError(t, json.Unmarshal(data, &out))
	require.Equal(t, in, out)
}

func TestTypeRefKindString(t *testing.T) {
	require.Equal(t, "primitive", KindPrimitive.String())
	require.Equal(t, "named", KindNamed.String())
	require.Equal(t, "array", KindArray.String())
	require.Equal(t, "oneOf", KindOneOf.String())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/spec/...`
Expected: build failure — `spec` package does not exist.

- [ ] **Step 3: Implement `internal/spec/ir.go`**

```go
// Package spec defines the intermediate representation produced by the
// Telegram Bot API scraper (cmd/scrape) and consumed by the code generator
// (cmd/genapi). It is committed as internal/spec/api.json so PR diffs read
// as a Telegram changelog.
package spec

// API is the top-level IR document.
type API struct {
	// Version is the Telegram Bot API version parsed from the
	// "Recent changes" section of the docs page.
	Version string `json:"version"`
	// Types lists all object types in declaration order.
	Types []TypeDecl `json:"types"`
	// Methods lists all API methods in declaration order.
	Methods []MethodDecl `json:"methods"`
}

// TypeDecl describes a Telegram object type.
type TypeDecl struct {
	Name   string   `json:"name"`
	Doc    string   `json:"doc,omitempty"`
	Fields []Field  `json:"fields,omitempty"`
	// OneOf, when non-empty, indicates this type is a union and lists the
	// concrete variant type names. Variants are emitted as concrete structs
	// implementing a sealed interface.
	OneOf []string `json:"one_of,omitempty"`
}

// MethodDecl describes a Telegram API method.
type MethodDecl struct {
	Name     string  `json:"name"`
	Doc      string  `json:"doc,omitempty"`
	Params   []Field `json:"params,omitempty"`
	Returns  TypeRef `json:"returns"`
	// HasFiles is true when any parameter accepts an InputFile, requiring
	// a multipart/form-data request.
	HasFiles bool `json:"has_files,omitempty"`
}

// Field describes a single field on a type or a single parameter on a method.
type Field struct {
	// Name is the Go-style identifier (e.g. "ChatID").
	Name string `json:"name"`
	// JSONName is the wire name (e.g. "chat_id").
	JSONName string  `json:"json_name"`
	Type     TypeRef `json:"type"`
	Required bool    `json:"required,omitempty"`
	Doc      string  `json:"doc,omitempty"`
}

// Kind enumerates TypeRef shapes.
type Kind int

const (
	// KindPrimitive: int64, string, bool, float64.
	KindPrimitive Kind = iota
	// KindNamed: a TypeDecl by name.
	KindNamed
	// KindArray: ElemType is the element type.
	KindArray
	// KindOneOf: Variants lists discriminant union members.
	KindOneOf
)

// String returns a stable, lowercase representation suitable for serialisation.
func (k Kind) String() string {
	switch k {
	case KindPrimitive:
		return "primitive"
	case KindNamed:
		return "named"
	case KindArray:
		return "array"
	case KindOneOf:
		return "oneOf"
	default:
		return "unknown"
	}
}

// MarshalText / UnmarshalText keep JSON output human-readable.
func (k Kind) MarshalText() ([]byte, error) { return []byte(k.String()), nil }

func (k *Kind) UnmarshalText(b []byte) error {
	switch string(b) {
	case "primitive":
		*k = KindPrimitive
	case "named":
		*k = KindNamed
	case "array":
		*k = KindArray
	case "oneOf":
		*k = KindOneOf
	default:
		*k = -1
	}
	return nil
}

// TypeRef is a structural reference used wherever a Field type is expressed.
type TypeRef struct {
	Kind     Kind     `json:"kind"`
	Name     string   `json:"name,omitempty"`
	ElemType *TypeRef `json:"elem_type,omitempty"`
	Variants []string `json:"variants,omitempty"`
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/spec/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/
git commit -m "feat(spec): define IR types for codegen pipeline"
```

---

## Task 4 — Client: Codec interface + default

**Files:**
- Create: `client/codec.go`
- Create: `client/codec_test.go`

- [ ] **Step 1: Write the failing test**

```go
package client

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultCodec_RoundTrip(t *testing.T) {
	c := DefaultCodec{}
	type payload struct {
		Name string `json:"name"`
		N    int    `json:"n"`
	}
	in := payload{Name: "x", N: 7}
	data, err := c.Marshal(in)
	require.NoError(t, err)
	require.JSONEq(t, `{"name":"x","n":7}`, string(data))

	var out payload
	require.NoError(t, c.Unmarshal(data, &out))
	require.Equal(t, in, out)
}

func TestDefaultCodec_UnmarshalError(t *testing.T) {
	var v map[string]any
	err := DefaultCodec{}.Unmarshal([]byte(`not json`), &v)
	require.Error(t, err)
}
```

- [ ] **Step 2: Run test — expect compile failure (Codec/DefaultCodec missing)**

Run: `go test ./client/...`
Expected: FAIL — undefined `Codec`, `DefaultCodec`.

- [ ] **Step 3: Implement `client/codec.go`**

```go
package client

import "encoding/json"

// Codec encodes/decodes JSON payloads exchanged with the Telegram Bot API.
// The default implementation wraps encoding/json. Users may plug in
// goccy/go-json, bytedance/sonic, or any compatible encoder by passing
// WithCodec to New.
type Codec interface {
	Marshal(v any) ([]byte, error)
	Unmarshal(data []byte, v any) error
}

// DefaultCodec wraps encoding/json. It is the zero-value safe default.
type DefaultCodec struct{}

// Marshal calls json.Marshal.
func (DefaultCodec) Marshal(v any) ([]byte, error) { return json.Marshal(v) }

// Unmarshal calls json.Unmarshal.
func (DefaultCodec) Unmarshal(data []byte, v any) error { return json.Unmarshal(data, v) }
```

- [ ] **Step 4: Run test — expect PASS**

Run: `go test ./client/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add client/
git commit -m "feat(client): add Codec interface with encoding/json default"
```

---

## Task 5 — Client: HTTPDoer interface + default

**Files:**
- Create: `client/httpclient.go`
- Create: `client/httpclient_test.go`

- [ ] **Step 1: Write the failing test**

```go
package client

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultHTTPClient_Do(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	t.Cleanup(srv.Close)

	doer := NewDefaultHTTPDoer()
	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	require.NoError(t, err)
	resp, err := doer.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusTeapot, resp.StatusCode)
}
```

- [ ] **Step 2: Run test — expect compile failure**

Run: `go test ./client/...`
Expected: FAIL — undefined `HTTPDoer`, `NewDefaultHTTPDoer`.

- [ ] **Step 3: Implement `client/httpclient.go`**

```go
package client

import (
	"net"
	"net/http"
	"time"
)

// HTTPDoer abstracts the HTTP transport. The default is a net/http client
// tuned for Telegram's long-poll usage. Users may plug in valyala/fasthttp
// (via an adapter), or any custom retry/circuit-breaker client by passing
// WithHTTPClient to New.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// NewDefaultHTTPDoer returns an *http.Client with sensible defaults for
// Telegram Bot API usage:
//   - 60s overall timeout (longer than typical long-poll Timeout=30s).
//   - Connection pooling sized for a small number of long-lived hosts.
//   - HTTP/2 enabled (default in net/http).
func NewDefaultHTTPDoer() *http.Client {
	t := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          16,
		MaxIdleConnsPerHost:   8,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
	}
	return &http.Client{
		Transport: t,
		Timeout:   60 * time.Second,
	}
}
```

- [ ] **Step 4: Run test — expect PASS**

Run: `go test ./client/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add client/httpclient.go client/httpclient_test.go
git commit -m "feat(client): add HTTPDoer interface with tuned net/http default"
```

---

## Task 6 — Client: Logger interface + nil-safe noop default

**Files:**
- Create: `client/logger.go`
- Create: `client/logger_test.go`

- [ ] **Step 1: Write the failing test**

```go
package client

import "testing"

func TestNoopLogger_DoesNotPanic(t *testing.T) {
	var l Logger = NoopLogger{}
	l.Debug("d", "k", "v")
	l.Info("i")
	l.Warn("w")
	l.Error("e")
}

func TestNoopLogger_NilSafe(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("nil logger should be usable through helper, got panic: %v", r)
		}
	}()
	var l Logger
	logDebug(l, "x")
	logInfo(l, "y")
	logWarn(l, "z")
	logError(l, "e")
}
```

- [ ] **Step 2: Run test — expect compile failure**

Run: `go test ./client/...`

- [ ] **Step 3: Implement `client/logger.go`**

```go
package client

// Logger is a slog-shaped logging interface. Users pass any compatible
// implementation via WithLogger. The default is NoopLogger, which discards
// everything. Internal helpers (logDebug, logInfo, logWarn, logError) are
// nil-safe: passing a nil Logger is equivalent to NoopLogger.
type Logger interface {
	Debug(msg string, attrs ...any)
	Info(msg string, attrs ...any)
	Warn(msg string, attrs ...any)
	Error(msg string, attrs ...any)
}

// NoopLogger discards all log records. It is the zero-value safe default.
type NoopLogger struct{}

func (NoopLogger) Debug(string, ...any) {}
func (NoopLogger) Info(string, ...any)  {}
func (NoopLogger) Warn(string, ...any)  {}
func (NoopLogger) Error(string, ...any) {}

func logDebug(l Logger, msg string, attrs ...any) {
	if l == nil {
		return
	}
	l.Debug(msg, attrs...)
}
func logInfo(l Logger, msg string, attrs ...any) {
	if l == nil {
		return
	}
	l.Info(msg, attrs...)
}
func logWarn(l Logger, msg string, attrs ...any) {
	if l == nil {
		return
	}
	l.Warn(msg, attrs...)
}
func logError(l Logger, msg string, attrs ...any) {
	if l == nil {
		return
	}
	l.Error(msg, attrs...)
}
```

- [ ] **Step 4: Run test — PASS**

Run: `go test ./client/...`

- [ ] **Step 5: Commit**

```bash
git add client/logger.go client/logger_test.go
git commit -m "feat(client): add Logger interface with nil-safe noop default"
```

---

## Task 7 — Client: Bot + functional options + Result envelope

**Files:**
- Create: `client/client.go`
- Create: `client/options.go`
- Create: `client/result.go`
- Create: `client/client_test.go`

- [ ] **Step 1: Write the failing test**

`client/client_test.go`:

```go
package client

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNew_Defaults(t *testing.T) {
	b := New("123:abc")
	require.Equal(t, "123:abc", b.token)
	require.Equal(t, defaultBaseURL, b.base)
	require.NotNil(t, b.http)
	require.NotNil(t, b.codec)
	require.NotNil(t, b.logger)
}

func TestNew_OptionsApplied(t *testing.T) {
	custom := &http.Client{}
	type fakeCodec struct{ DefaultCodec }
	c := fakeCodec{}

	b := New("t",
		WithHTTPClient(custom),
		WithCodec(c),
		WithBaseURL("https://example.test"),
		WithLogger(NoopLogger{}),
	)
	require.Same(t, custom, b.http)
	require.Equal(t, c, b.codec)
	require.Equal(t, "https://example.test", b.base)
}

func TestResultRoundTrip(t *testing.T) {
	in := Result[int64]{OK: true, Result: 42}
	data, err := DefaultCodec{}.Marshal(in)
	require.NoError(t, err)
	var out Result[int64]
	require.NoError(t, DefaultCodec{}.Unmarshal(data, &out))
	require.Equal(t, in, out)
}
```

- [ ] **Step 2: Run test — expect compile failure**

- [ ] **Step 3: Implement `client/client.go`**

```go
package client

const defaultBaseURL = "https://api.telegram.org"

// Bot is the Telegram Bot API client. Construct via New. All API methods
// (declared in package api) hang off *Bot via thin wrappers around call.
type Bot struct {
	token  string
	base   string
	http   HTTPDoer
	codec  Codec
	logger Logger
}

// Token returns the bot token. Exposed for advanced use cases (custom
// transports, manual URL building); ordinary code does not need it.
func (b *Bot) Token() string { return b.token }

// BaseURL returns the configured Telegram API base URL.
func (b *Bot) BaseURL() string { return b.base }

// HTTP returns the underlying HTTPDoer. Exposed for adapters that need
// to share connection pools or for diagnostic checks.
func (b *Bot) HTTP() HTTPDoer { return b.http }

// Codec returns the configured Codec.
func (b *Bot) Codec() Codec { return b.codec }

// Logger returns the configured Logger.
func (b *Bot) Logger() Logger { return b.logger }

// New constructs a Bot with the given token and optional configuration.
// The default HTTP client is tuned for long-poll workloads (see
// NewDefaultHTTPDoer); the default codec wraps encoding/json; the default
// logger discards records.
func New(token string, opts ...Option) *Bot {
	b := &Bot{
		token:  token,
		base:   defaultBaseURL,
		http:   NewDefaultHTTPDoer(),
		codec:  DefaultCodec{},
		logger: NoopLogger{},
	}
	for _, o := range opts {
		o(b)
	}
	return b
}
```

- [ ] **Step 4: Implement `client/options.go`**

```go
package client

// Option configures a Bot at construction time. Per-call configuration is
// expressed via typed parameter structs (e.g. SendMessageParams), not options.
type Option func(*Bot)

// WithHTTPClient overrides the HTTP transport. Pass any HTTPDoer
// implementation (e.g. an *http.Client wrapping a custom RoundTripper, or
// a fasthttp adapter).
func WithHTTPClient(c HTTPDoer) Option { return func(b *Bot) { b.http = c } }

// WithCodec overrides the JSON codec. Pass goccy/go-json, sonic, or any
// type implementing Codec to swap out encoding/json.
func WithCodec(c Codec) Option { return func(b *Bot) { b.codec = c } }

// WithBaseURL overrides the API base URL. Useful for testing against a
// local httptest.Server, or for self-hosted Bot API servers.
func WithBaseURL(url string) Option { return func(b *Bot) { b.base = url } }

// WithLogger sets the logger used for diagnostic events. Passing nil
// silently disables logging.
func WithLogger(l Logger) Option { return func(b *Bot) { b.logger = l } }
```

- [ ] **Step 5: Implement `client/result.go`**

```go
package client

// Result is the universal Telegram API response envelope. Every successful
// response is shaped {"ok":true,"result":T,...}; failure responses set ok
// to false and populate ErrorCode / Description / Parameters.
//
// Result is generic over T so generated method wrappers can decode the
// strongly-typed payload directly. Users do not normally construct or
// inspect Result values; method wrappers unwrap them and return either
// the typed payload or a *APIError.
type Result[T any] struct {
	OK          bool                `json:"ok"`
	Result      T                   `json:"result,omitempty"`
	ErrorCode   int                 `json:"error_code,omitempty"`
	Description string              `json:"description,omitempty"`
	Parameters  *ResponseParameters `json:"parameters,omitempty"`
}

// ResponseParameters is the optional metadata Telegram includes on certain
// failures. The most common is RetryAfter (seconds) on 429 responses.
//
// This type is duplicated in package api for users; keeping a copy here
// avoids an import cycle (api imports client, not vice versa).
type ResponseParameters struct {
	MigrateToChatID int64 `json:"migrate_to_chat_id,omitempty"`
	RetryAfter      int   `json:"retry_after,omitempty"`
}
```

- [ ] **Step 6: Run tests — PASS**

Run: `go test ./client/...`

- [ ] **Step 7: Commit**

```bash
git add client/
git commit -m "feat(client): Bot constructor, options, and Result[T] envelope"
```

---

## Task 8 — Client: errors

**Files:**
- Create: `client/errors.go`
- Create: `client/errors_test.go`

- [ ] **Step 1: Write the failing test**

```go
package client

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAPIError_FieldsAndMethods(t *testing.T) {
	e := &APIError{
		Code:        429,
		Description: "Too Many Requests: retry after 5",
		Parameters:  &ResponseParameters{RetryAfter: 5},
	}
	require.Equal(t, "telegram: 429 Too Many Requests: retry after 5", e.Error())
	require.True(t, e.IsRetryable())
	require.Equal(t, 5*time.Second, e.RetryAfter())
}

func TestAPIError_Sentinels(t *testing.T) {
	cases := []struct {
		code     int
		desc     string
		sentinel error
	}{
		{401, "Unauthorized", ErrUnauthorized},
		{400, "Bad Request: chat not found", ErrChatNotFound},
		{400, "Bad Request: message is not modified", ErrMessageNotModified},
		{429, "Too Many Requests: retry after 1", ErrTooManyRequests},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			e := mapAPIError(c.code, c.desc, nil)
			require.True(t, errors.Is(e, c.sentinel), "expected %v to wrap %v", e, c.sentinel)
		})
	}
}

func TestAPIError_IsRetryable(t *testing.T) {
	require.True(t, (&APIError{Code: 500}).IsRetryable())
	require.True(t, (&APIError{Code: 502}).IsRetryable())
	require.True(t, (&APIError{Code: 429}).IsRetryable())
	require.False(t, (&APIError{Code: 400}).IsRetryable())
	require.False(t, (&APIError{Code: 401}).IsRetryable())
}

func TestNetworkAndParseErrorWrapping(t *testing.T) {
	inner := errors.New("dial tcp: timeout")
	ne := &NetworkError{Err: inner}
	require.ErrorIs(t, ne, inner)

	pe := &ParseError{Err: errors.New("unexpected EOF"), Body: []byte("garbage")}
	require.Contains(t, pe.Error(), "garbage")
}
```

- [ ] **Step 2: Run test — expect compile failure**

- [ ] **Step 3: Implement `client/errors.go`**

```go
package client

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// APIError represents a non-OK Telegram Bot API response.
// It satisfies error and unwraps to a sentinel (ErrUnauthorized, etc.)
// where the description matches a known prefix, enabling errors.Is checks.
type APIError struct {
	Code        int
	Description string
	Parameters  *ResponseParameters

	// sentinel, if non-nil, is the wrapped sentinel error returned by
	// Unwrap. It is set by mapAPIError based on Code+Description.
	sentinel error
}

// Error implements error.
func (e *APIError) Error() string {
	return fmt.Sprintf("telegram: %d %s", e.Code, e.Description)
}

// Unwrap returns the matched sentinel error, if any.
func (e *APIError) Unwrap() error { return e.sentinel }

// IsRetryable returns true for transient HTTP statuses (429, 5xx).
func (e *APIError) IsRetryable() bool {
	return e.Code == 429 || (e.Code >= 500 && e.Code < 600)
}

// RetryAfter returns the recommended back-off duration. It honours the
// Telegram-supplied retry_after parameter; if absent, returns 0.
func (e *APIError) RetryAfter() time.Duration {
	if e.Parameters == nil {
		return 0
	}
	return time.Duration(e.Parameters.RetryAfter) * time.Second
}

// NetworkError wraps a transport-level failure (DNS, TCP, TLS, timeout
// short of an HTTP response).
type NetworkError struct{ Err error }

func (e *NetworkError) Error() string { return "telegram: network: " + e.Err.Error() }
func (e *NetworkError) Unwrap() error { return e.Err }

// ParseError wraps a JSON decode failure on a response body. Body is
// retained (truncated to 4 KiB) for diagnostics.
type ParseError struct {
	Err  error
	Body []byte
}

func (e *ParseError) Error() string {
	body := e.Body
	if len(body) > 256 {
		body = body[:256]
	}
	return fmt.Sprintf("telegram: parse: %s (body=%q)", e.Err, body)
}
func (e *ParseError) Unwrap() error { return e.Err }

// Sentinel errors returned via APIError.Unwrap when the description matches.
// Compare with errors.Is.
var (
	ErrUnauthorized       = errors.New("telegram: unauthorized")
	ErrChatNotFound       = errors.New("telegram: chat not found")
	ErrMessageNotModified = errors.New("telegram: message is not modified")
	ErrTooManyRequests    = errors.New("telegram: too many requests")
	ErrBadRequest         = errors.New("telegram: bad request")
	ErrForbidden          = errors.New("telegram: forbidden")
)

// mapAPIError builds an *APIError and attaches the appropriate sentinel
// based on Code+Description. It is the single point where wire-level
// failures are translated into the Go error taxonomy.
func mapAPIError(code int, description string, params *ResponseParameters) *APIError {
	e := &APIError{Code: code, Description: description, Parameters: params}
	switch {
	case code == 401:
		e.sentinel = ErrUnauthorized
	case code == 403:
		e.sentinel = ErrForbidden
	case code == 429:
		e.sentinel = ErrTooManyRequests
	case code == 400 && strings.Contains(description, "chat not found"):
		e.sentinel = ErrChatNotFound
	case code == 400 && strings.Contains(description, "message is not modified"):
		e.sentinel = ErrMessageNotModified
	case code == 400:
		e.sentinel = ErrBadRequest
	}
	return e
}
```

- [ ] **Step 4: Run test — PASS**

Run: `go test ./client/...`

- [ ] **Step 5: Commit**

```bash
git add client/errors.go client/errors_test.go
git commit -m "feat(client): typed errors with sentinel mapping"
```

---

## Task 9 — Client: Call helper + multipart builder

**Files:**
- Create: `client/call.go`
- Create: `client/multipart.go`
- Create: `client/call_test.go`
- Create: `client/multipart_test.go`

This task is the heart of the library. `Call` is generic over request and response types and is the single point through which every API method is invoked. It is exported so the `api` package (Task 10–11) can call it.

- [ ] **Step 1: Write the failing test for `Call`**

`client/call_test.go`:

```go
package client

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockDoer struct{ mock.Mock }

func (m *mockDoer) Do(r *http.Request) (*http.Response, error) {
	args := m.Called(r)
	if v := args.Get(0); v != nil {
		return v.(*http.Response), args.Error(1)
	}
	return nil, args.Error(1)
}

func newResp(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}

type echoReq struct {
	ChatID int64  `json:"chat_id"`
	Text   string `json:"text"`
}
type echoResp struct {
	MessageID int64 `json:"message_id"`
}

func TestCall_Success(t *testing.T) {
	m := &mockDoer{}
	m.On("Do", mock.MatchedBy(func(r *http.Request) bool {
		if !strings.HasSuffix(r.URL.Path, "/bot123:abc/sendEcho") {
			return false
		}
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(r.Body)
		return strings.Contains(buf.String(), `"chat_id":42`)
	})).Return(newResp(200, `{"ok":true,"result":{"message_id":7}}`), nil)

	b := New("123:abc", WithHTTPClient(m))
	out, err := Call[*echoReq, *echoResp](context.Background(), b, "sendEcho", &echoReq{ChatID: 42, Text: "hi"})
	require.NoError(t, err)
	require.Equal(t, int64(7), out.MessageID)
	m.AssertExpectations(t)
}

func TestCall_APIError(t *testing.T) {
	m := &mockDoer{}
	m.On("Do", mock.Anything).Return(
		newResp(200, `{"ok":false,"error_code":429,"description":"Too Many Requests: retry after 3","parameters":{"retry_after":3}}`), nil)

	b := New("t", WithHTTPClient(m))
	_, err := Call[*echoReq, *echoResp](context.Background(), b, "x", &echoReq{})
	require.Error(t, err)
	var ae *APIError
	require.ErrorAs(t, err, &ae)
	require.Equal(t, 429, ae.Code)
	require.True(t, ae.IsRetryable())
	require.True(t, errors.Is(err, ErrTooManyRequests))
}

func TestCall_NetworkError(t *testing.T) {
	m := &mockDoer{}
	m.On("Do", mock.Anything).Return(nil, errors.New("dial timeout"))

	b := New("t", WithHTTPClient(m))
	_, err := Call[*echoReq, *echoResp](context.Background(), b, "x", &echoReq{})
	require.Error(t, err)
	var ne *NetworkError
	require.ErrorAs(t, err, &ne)
}

func TestCall_ParseError(t *testing.T) {
	m := &mockDoer{}
	m.On("Do", mock.Anything).Return(newResp(200, `not json`), nil)

	b := New("t", WithHTTPClient(m))
	_, err := Call[*echoReq, *echoResp](context.Background(), b, "x", &echoReq{})
	require.Error(t, err)
	var pe *ParseError
	require.ErrorAs(t, err, &pe)
}

func TestCall_ContextCanceled(t *testing.T) {
	m := &mockDoer{}
	m.On("Do", mock.Anything).Return(nil, context.Canceled).Maybe()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	b := New("t", WithHTTPClient(m))
	_, err := Call[*echoReq, *echoResp](ctx, b, "x", &echoReq{})
	require.ErrorIs(t, err, context.Canceled)
}

func TestCall_NilRequest(t *testing.T) {
	// Methods with no params (e.g. getMe) may pass a nil Req value.
	m := &mockDoer{}
	m.On("Do", mock.MatchedBy(func(r *http.Request) bool {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(r.Body)
		return buf.String() == "{}"
	})).Return(newResp(200, `{"ok":true,"result":{"message_id":0}}`), nil)

	b := New("t", WithHTTPClient(m))
	_, err := Call[*echoReq, *echoResp](context.Background(), b, "x", nil)
	require.NoError(t, err)
}
```

- [ ] **Step 2: Run test — expect compile failure** (`Call` is not defined yet)

- [ ] **Step 3: Implement `client/call.go`**

```go
package client

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"reflect"
)

// Call is the single point through which every Telegram Bot API method
// invocation flows. It marshals the request, signs the URL with the bot
// token, dispatches via HTTPDoer, decodes the Result envelope, and
// translates non-OK responses into typed errors.
//
// It is generic over both request and response types. Methods with no
// parameters may pass a nil Req; the helper sends "{}" in that case so
// Telegram receives a syntactically valid empty object.
//
// Call is exported because the api package (which lives outside this one)
// invokes it from generated method wrappers. User code should not normally
// call it directly — use the typed wrappers in package api instead.
func Call[Req any, Resp any](ctx context.Context, b *Bot, method string, req Req) (Resp, error) {
	var zero Resp

	if mp, ok := any(req).(multipartRequest); ok && mp != nil && mp.HasFile() {
		return callMultipart[Resp](ctx, b, method, mp)
	}

	body, err := encodeJSONBody(b.codec, req)
	if err != nil {
		return zero, err
	}

	url := b.base + "/bot" + b.token + "/" + method
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return zero, &NetworkError{Err: err}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := b.http.Do(httpReq)
	if err != nil {
		// Surface ctx errors faithfully so callers can errors.Is(err, ctx.Err()).
		if ctxErr := ctx.Err(); ctxErr != nil {
			return zero, ctxErr
		}
		return zero, &NetworkError{Err: err}
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return zero, &NetworkError{Err: err}
	}

	return decodeResult[Resp](b.codec, raw)
}

// encodeJSONBody marshals req to a JSON body. A nil interface or nil
// pointer req yields "{}" so Telegram receives a valid empty object.
func encodeJSONBody(codec Codec, req any) (io.Reader, error) {
	if req == nil || isNilPointer(req) {
		return bytes.NewBufferString("{}"), nil
	}
	data, err := codec.Marshal(req)
	if err != nil {
		return nil, &ParseError{Err: err}
	}
	return bytes.NewReader(data), nil
}

// decodeResult unmarshals raw into Result[Resp] and translates non-OK
// responses into *APIError.
func decodeResult[Resp any](codec Codec, raw []byte) (Resp, error) {
	var zero Resp
	var env Result[Resp]
	if err := codec.Unmarshal(raw, &env); err != nil {
		return zero, &ParseError{Err: err, Body: copyBody(raw)}
	}
	if !env.OK {
		return zero, mapAPIError(env.ErrorCode, env.Description, env.Parameters)
	}
	return env.Result, nil
}

// isNilPointer returns true when v is a typed nil pointer (the interface
// itself is non-nil because it carries a type, but the underlying value
// is nil). One reflect call per request; not on a hot path that demands
// allocation-freedom.
func isNilPointer(v any) bool {
	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Ptr && rv.IsNil()
}

func copyBody(b []byte) []byte {
	const max = 4096
	if len(b) > max {
		b = b[:max]
	}
	out := make([]byte, len(b))
	copy(out, b)
	return out
}
```

- [ ] **Step 4: Implement `client/multipart.go`**

```go
package client

import (
	"context"
	"io"
	"mime/multipart"
	"net/http"
)

// multipartRequest is implemented by request structs that may carry an
// InputFile. The codegen emits this interface for any method whose IR
// MethodDecl.HasFiles is true.
//
// HasFile returns true if at least one file field is set; if false, the
// request is sent as plain JSON via the regular call path.
//
// MultipartFiles returns one entry per file field that should be uploaded.
// The accompanying scalar/object fields are returned by MultipartFields.
type multipartRequest interface {
	HasFile() bool
	MultipartFiles() []MultipartFile
	MultipartFields() map[string]string
}

// MultipartFile describes a single file part in a multipart upload.
type MultipartFile struct {
	FieldName string
	Filename  string
	Reader    io.Reader
}

// callMultipart performs a multipart/form-data POST. It is invoked by call
// when the request implements multipartRequest and HasFile() is true.
func callMultipart[Resp any](ctx context.Context, b *Bot, method string, mp multipartRequest) (Resp, error) {
	var zero Resp

	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)

	// Stream-write the multipart body in a goroutine so we don't buffer
	// large files in memory.
	go func() {
		defer pw.Close()
		defer mw.Close()
		for k, v := range mp.MultipartFields() {
			if err := mw.WriteField(k, v); err != nil {
				_ = pw.CloseWithError(err)
				return
			}
		}
		for _, f := range mp.MultipartFiles() {
			part, err := mw.CreateFormFile(f.FieldName, f.Filename)
			if err != nil {
				_ = pw.CloseWithError(err)
				return
			}
			if _, err := io.Copy(part, f.Reader); err != nil {
				_ = pw.CloseWithError(err)
				return
			}
		}
	}()

	url := b.base + "/bot" + b.token + "/" + method
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, pr)
	if err != nil {
		return zero, &NetworkError{Err: err}
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Accept", "application/json")

	resp, err := b.http.Do(req)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return zero, ctxErr
		}
		return zero, &NetworkError{Err: err}
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return zero, &NetworkError{Err: err}
	}
	return decodeResult[Resp](b.codec, raw)
}

```

- [ ] **Step 5: Write the multipart test**

`client/multipart_test.go`:

```go
package client

import (
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type fakeMultipartReq struct {
	chatID int64
	body   string
}

func (f *fakeMultipartReq) HasFile() bool { return true }
func (f *fakeMultipartReq) MultipartFields() map[string]string {
	return map[string]string{"chat_id": "42"}
}
func (f *fakeMultipartReq) MultipartFiles() []MultipartFile {
	return []MultipartFile{{
		FieldName: "document",
		Filename:  "hello.txt",
		Reader:    strings.NewReader(f.body),
	}}
}

type fileResp struct {
	MessageID int64 `json:"message_id"`
}

func TestCallMultipart_Success(t *testing.T) {
	m := &mockDoer{}
	m.On("Do", mock.MatchedBy(func(r *http.Request) bool {
		ct := r.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "multipart/form-data") {
			return false
		}
		// Parse and verify content
		_, params, err := mime.ParseMediaType(ct)
		if err != nil {
			return false
		}
		mr := multipart.NewReader(r.Body, params["boundary"])
		seenChat := false
		seenFile := false
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return false
			}
			switch p.FormName() {
			case "chat_id":
				body, _ := io.ReadAll(p)
				seenChat = string(body) == "42"
			case "document":
				body, _ := io.ReadAll(p)
				seenFile = string(body) == "hello world"
			}
		}
		return seenChat && seenFile
	})).Return(newResp(200, `{"ok":true,"result":{"message_id":99}}`), nil)

	b := New("t", WithHTTPClient(m))
	out, err := call[*fakeMultipartReq, *fileResp](context.Background(), b, "sendDocument", &fakeMultipartReq{chatID: 42, body: "hello world"})
	require.NoError(t, err)
	require.Equal(t, int64(99), out.MessageID)
}
```

- [ ] **Step 6: Run all client tests — PASS**

Run: `go test -race ./client/...`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add client/
git commit -m "feat(client): add generic call helper, multipart builder, and tests"
```

---

## Task 10 — `api/` hand-coded core types

The convention here is critical: Plan 2 codegen will produce structs in this exact shape. The structs we emit by hand must match what the scraper would emit, so the eventual swap-over is byte-identical (or near-identical) Go.

**Files:**
- Create: `api/types.go`
- Create: `api/types_test.go`

- [ ] **Step 1: Implement `api/types.go`**

```go
// Package api contains the Telegram Bot API object types and method
// wrappers. In Plan 1 these are hand-coded for a small subset of the API;
// Plan 2 replaces them with code generated from the live documentation.
//
// The hand-coded subset covers what is needed for the echo and webhook
// example bots: Update plumbing, Message + sender objects, command parsing,
// and basic callback queries.
package api

// Update represents an incoming update from Telegram. Exactly one of the
// optional payload fields is populated per Update.
//
// https://core.telegram.org/bots/api#update
type Update struct {
	UpdateID          int64          `json:"update_id"`
	Message           *Message       `json:"message,omitempty"`
	EditedMessage     *Message       `json:"edited_message,omitempty"`
	ChannelPost       *Message       `json:"channel_post,omitempty"`
	EditedChannelPost *Message       `json:"edited_channel_post,omitempty"`
	CallbackQuery     *CallbackQuery `json:"callback_query,omitempty"`
	InlineQuery       *InlineQuery   `json:"inline_query,omitempty"`
}

// UpdateType identifies an Update payload variant. Used by allowed_updates
// in getUpdates / setWebhook.
type UpdateType string

const (
	UpdateMessage           UpdateType = "message"
	UpdateEditedMessage     UpdateType = "edited_message"
	UpdateChannelPost       UpdateType = "channel_post"
	UpdateEditedChannelPost UpdateType = "edited_channel_post"
	UpdateCallbackQuery     UpdateType = "callback_query"
	UpdateInlineQuery       UpdateType = "inline_query"
)

// User represents a Telegram user or bot.
//
// https://core.telegram.org/bots/api#user
type User struct {
	ID           int64  `json:"id"`
	IsBot        bool   `json:"is_bot"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name,omitempty"`
	Username     string `json:"username,omitempty"`
	LanguageCode string `json:"language_code,omitempty"`
}

// ChatType is the type of a Telegram chat.
type ChatType string

const (
	ChatTypePrivate    ChatType = "private"
	ChatTypeGroup      ChatType = "group"
	ChatTypeSupergroup ChatType = "supergroup"
	ChatTypeChannel    ChatType = "channel"
)

// Chat represents a chat.
//
// https://core.telegram.org/bots/api#chat
type Chat struct {
	ID       int64    `json:"id"`
	Type     ChatType `json:"type"`
	Title    string   `json:"title,omitempty"`
	Username string   `json:"username,omitempty"`
	FirstName string  `json:"first_name,omitempty"`
	LastName  string  `json:"last_name,omitempty"`
}

// MessageEntityType is the kind of an entity (mention, hashtag, command, ...).
type MessageEntityType string

const (
	EntityMention      MessageEntityType = "mention"
	EntityHashtag      MessageEntityType = "hashtag"
	EntityCashtag      MessageEntityType = "cashtag"
	EntityBotCommand   MessageEntityType = "bot_command"
	EntityURL          MessageEntityType = "url"
	EntityEmail        MessageEntityType = "email"
	EntityPhoneNumber  MessageEntityType = "phone_number"
	EntityBold         MessageEntityType = "bold"
	EntityItalic       MessageEntityType = "italic"
	EntityUnderline    MessageEntityType = "underline"
	EntityStrike       MessageEntityType = "strikethrough"
	EntitySpoiler      MessageEntityType = "spoiler"
	EntityCode         MessageEntityType = "code"
	EntityPre          MessageEntityType = "pre"
	EntityTextLink     MessageEntityType = "text_link"
	EntityTextMention  MessageEntityType = "text_mention"
	EntityCustomEmoji  MessageEntityType = "custom_emoji"
)

// MessageEntity describes one special entity in a text message.
type MessageEntity struct {
	Type     MessageEntityType `json:"type"`
	Offset   int               `json:"offset"`
	Length   int               `json:"length"`
	URL      string            `json:"url,omitempty"`
	User     *User             `json:"user,omitempty"`
	Language string            `json:"language,omitempty"`
}

// Message represents a message.
//
// https://core.telegram.org/bots/api#message
type Message struct {
	MessageID int64           `json:"message_id"`
	From      *User           `json:"from,omitempty"`
	Date      int64           `json:"date"`
	Chat      Chat            `json:"chat"`
	Text      string          `json:"text,omitempty"`
	Caption   string          `json:"caption,omitempty"`
	Entities  []MessageEntity `json:"entities,omitempty"`
	ReplyToMessage *Message   `json:"reply_to_message,omitempty"`
}

// CallbackQuery represents an incoming callback from an inline keyboard.
//
// https://core.telegram.org/bots/api#callbackquery
type CallbackQuery struct {
	ID              string   `json:"id"`
	From            User     `json:"from"`
	Message         *Message `json:"message,omitempty"`
	InlineMessageID string   `json:"inline_message_id,omitempty"`
	ChatInstance    string   `json:"chat_instance"`
	Data            string   `json:"data,omitempty"`
}

// InlineQuery represents an incoming inline query.
//
// https://core.telegram.org/bots/api#inlinequery
type InlineQuery struct {
	ID     string `json:"id"`
	From   User   `json:"from"`
	Query  string `json:"query"`
	Offset string `json:"offset"`
}

// ResponseParameters duplicates client.ResponseParameters so users
// importing only api can reference it without importing client.
//
// https://core.telegram.org/bots/api#responseparameters
type ResponseParameters struct {
	MigrateToChatID int64 `json:"migrate_to_chat_id,omitempty"`
	RetryAfter      int   `json:"retry_after,omitempty"`
}

// ParseMode controls how Telegram interprets formatting in message text.
type ParseMode string

const (
	ParseModeMarkdown   ParseMode = "Markdown"   // legacy
	ParseModeMarkdownV2 ParseMode = "MarkdownV2"
	ParseModeHTML       ParseMode = "HTML"
)

// InputFile carries either a file path (for upload) or a Telegram file_id
// / URL string (for reuse). When PathOrID names a local file, the request
// is sent as multipart/form-data; otherwise the value is sent inline.
type InputFile struct {
	// PathOrID is one of: an absolute or relative filesystem path, a
	// previously-uploaded Telegram file_id, or an HTTPS URL Telegram
	// can fetch.
	PathOrID string
	// Reader, when non-nil, is used as the file content (Filename names it).
	Reader io.Reader
	// Filename is the upload filename used when Reader is set.
	Filename string
}

// IsLocalUpload reports whether this InputFile triggers a multipart upload.
func (f *InputFile) IsLocalUpload() bool {
	if f == nil {
		return false
	}
	return f.Reader != nil
}
```

Add `import "io"` at the top of the file.

- [ ] **Step 2: Implement `api/types_test.go`** — round-trip JSON for the types

```go
package api

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdate_RoundTrip(t *testing.T) {
	in := Update{
		UpdateID: 1,
		Message: &Message{
			MessageID: 7,
			Date:      1234567890,
			Chat:      Chat{ID: 42, Type: ChatTypePrivate},
			Text:      "/start",
			Entities: []MessageEntity{{
				Type: EntityBotCommand, Offset: 0, Length: 6,
			}},
		},
	}
	data, err := json.Marshal(in)
	require.NoError(t, err)

	var out Update
	require.NoError(t, json.Unmarshal(data, &out))
	require.Equal(t, in, out)
}

func TestMessage_OmitsOptionalFields(t *testing.T) {
	m := Message{MessageID: 1, Date: 2, Chat: Chat{ID: 3, Type: ChatTypePrivate}}
	data, err := json.Marshal(m)
	require.NoError(t, err)
	require.NotContains(t, string(data), "from")
	require.NotContains(t, string(data), "text")
	require.NotContains(t, string(data), "entities")
}

func TestInputFile_IsLocalUpload(t *testing.T) {
	require.False(t, (*InputFile)(nil).IsLocalUpload())
	require.False(t, (&InputFile{PathOrID: "AgADAgADu7gxG..."}).IsLocalUpload())
	require.True(t, (&InputFile{Reader: nopReader{}}).IsLocalUpload())
}

type nopReader struct{}

func (nopReader) Read(p []byte) (int, error) { return 0, nil }
```

- [ ] **Step 3: Run tests — PASS**

Run: `go test ./api/...`

- [ ] **Step 4: Commit**

```bash
git add api/types.go api/types_test.go
git commit -m "feat(api): hand-coded core object types (Update, Message, User, ...)"
```

---

## Task 11 — `api/` hand-coded methods

**Files:**
- Create: `api/methods.go`
- Create: `api/methods_test.go`

- [ ] **Step 1: Implement `api/methods.go`** — wrappers for `getMe`, `getUpdates`, `sendMessage`, `setWebhook`, `deleteWebhook`, `answerCallbackQuery`, `sendDocument` (multipart example).

```go
package api

import (
	"context"
	"strconv"

	"github.com/lukaszraczylo/go-telegram/client"
)

// callJSON is the in-package re-export of client.Call so generated code
// (and the hand-coded methods in this file) need not import the unexported
// helper directly.
//
// We re-export here as a thin wrapper rather than exposing client.Call
// publicly because callers should only invoke methods through the
// generated wrappers; the call shape itself is an implementation detail.
type bot = client.Bot

// --- getMe ---------------------------------------------------------------

// GetMeParams is the parameter set for getMe. The method takes no parameters,
// but a typed struct is exposed for symmetry and so users can wire it into
// generic helpers (e.g. middleware that records method+params).
type GetMeParams struct{}

// GetMe returns basic information about the bot in the form of a User.
//
// https://core.telegram.org/bots/api#getme
func GetMe(ctx context.Context, b *bot) (*User, error) {
	return client.Call[*GetMeParams, *User](ctx, b, "getMe", &GetMeParams{})
}

// --- getUpdates ----------------------------------------------------------

// GetUpdatesParams is the parameter set for getUpdates.
//
// https://core.telegram.org/bots/api#getupdates
type GetUpdatesParams struct {
	Offset         int64        `json:"offset,omitempty"`
	Limit          int          `json:"limit,omitempty"`
	Timeout        int          `json:"timeout,omitempty"`
	AllowedUpdates []UpdateType `json:"allowed_updates,omitempty"`
}

// GetUpdates returns an array of incoming updates.
func GetUpdates(ctx context.Context, b *bot, p *GetUpdatesParams) ([]Update, error) {
	return client.Call[*GetUpdatesParams, []Update](ctx, b, "getUpdates", p)
}

// --- sendMessage ---------------------------------------------------------

// SendMessageParams is the parameter set for sendMessage.
//
// https://core.telegram.org/bots/api#sendmessage
type SendMessageParams struct {
	ChatID                int64       `json:"chat_id"`
	Text                  string      `json:"text"`
	ParseMode             ParseMode   `json:"parse_mode,omitempty"`
	Entities              []MessageEntity `json:"entities,omitempty"`
	DisableWebPagePreview *bool       `json:"disable_web_page_preview,omitempty"`
	DisableNotification   *bool       `json:"disable_notification,omitempty"`
	ProtectContent        *bool       `json:"protect_content,omitempty"`
	ReplyToMessageID      int64       `json:"reply_to_message_id,omitempty"`
	AllowSendingWithoutReply *bool    `json:"allow_sending_without_reply,omitempty"`
}

// SendMessage sends a text message and returns the sent Message.
func SendMessage(ctx context.Context, b *bot, p *SendMessageParams) (*Message, error) {
	return client.Call[*SendMessageParams, *Message](ctx, b, "sendMessage", p)
}

// --- setWebhook / deleteWebhook ----------------------------------------

// SetWebhookParams is the parameter set for setWebhook.
//
// https://core.telegram.org/bots/api#setwebhook
type SetWebhookParams struct {
	URL                string       `json:"url"`
	Certificate        *InputFile   `json:"certificate,omitempty"`
	IPAddress          string       `json:"ip_address,omitempty"`
	MaxConnections     int          `json:"max_connections,omitempty"`
	AllowedUpdates     []UpdateType `json:"allowed_updates,omitempty"`
	DropPendingUpdates *bool        `json:"drop_pending_updates,omitempty"`
	SecretToken        string       `json:"secret_token,omitempty"`
}

// SetWebhook configures a webhook URL for incoming updates.
func SetWebhook(ctx context.Context, b *bot, p *SetWebhookParams) (bool, error) {
	return client.Call[*SetWebhookParams, bool](ctx, b, "setWebhook", p)
}

// DeleteWebhookParams is the parameter set for deleteWebhook.
type DeleteWebhookParams struct {
	DropPendingUpdates *bool `json:"drop_pending_updates,omitempty"`
}

// DeleteWebhook removes the webhook configuration.
func DeleteWebhook(ctx context.Context, b *bot, p *DeleteWebhookParams) (bool, error) {
	return client.Call[*DeleteWebhookParams, bool](ctx, b, "deleteWebhook", p)
}

// --- answerCallbackQuery ----------------------------------------------

// AnswerCallbackQueryParams is the parameter set for answerCallbackQuery.
//
// https://core.telegram.org/bots/api#answercallbackquery
type AnswerCallbackQueryParams struct {
	CallbackQueryID string `json:"callback_query_id"`
	Text            string `json:"text,omitempty"`
	ShowAlert       *bool  `json:"show_alert,omitempty"`
	URL             string `json:"url,omitempty"`
	CacheTime       int    `json:"cache_time,omitempty"`
}

// AnswerCallbackQuery acknowledges a CallbackQuery from an inline keyboard.
func AnswerCallbackQuery(ctx context.Context, b *bot, p *AnswerCallbackQueryParams) (bool, error) {
	return client.Call[*AnswerCallbackQueryParams, bool](ctx, b, "answerCallbackQuery", p)
}

// --- sendDocument (multipart sample) -----------------------------------

// SendDocumentParams is the parameter set for sendDocument.
//
// https://core.telegram.org/bots/api#senddocument
type SendDocumentParams struct {
	ChatID    int64      `json:"chat_id"`
	Document  *InputFile `json:"document"`
	Caption   string     `json:"caption,omitempty"`
	ParseMode ParseMode  `json:"parse_mode,omitempty"`
}

// HasFile reports whether a multipart upload is required.
func (p *SendDocumentParams) HasFile() bool { return p.Document.IsLocalUpload() }

// MultipartFields returns the non-file fields used in the multipart body.
func (p *SendDocumentParams) MultipartFields() map[string]string {
	out := map[string]string{
		"chat_id": strconv.FormatInt(p.ChatID, 10),
	}
	if p.Caption != "" {
		out["caption"] = p.Caption
	}
	if p.ParseMode != "" {
		out["parse_mode"] = string(p.ParseMode)
	}
	return out
}

// MultipartFiles returns the file parts.
func (p *SendDocumentParams) MultipartFiles() []client.MultipartFile {
	if !p.HasFile() {
		return nil
	}
	name := p.Document.Filename
	if name == "" {
		name = "file"
	}
	return []client.MultipartFile{{
		FieldName: "document",
		Filename:  name,
		Reader:    p.Document.Reader,
	}}
}

// SendDocument sends a generic document and returns the sent Message.
func SendDocument(ctx context.Context, b *bot, p *SendDocumentParams) (*Message, error) {
	return client.Call[*SendDocumentParams, *Message](ctx, b, "sendDocument", p)
}

// methodNames is a debugging helper exposing the wired method names; useful
// in tests to assert that nothing has been forgotten when extending coverage.
func methodNames() []string {
	return []string{"getMe", "getUpdates", "sendMessage", "setWebhook", "deleteWebhook", "answerCallbackQuery", "sendDocument"}
}
```

- [ ] **Step 2: Implement `api/methods_test.go`**

```go
package api

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/lukaszraczylo/go-telegram/client"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockDoer struct{ mock.Mock }

func (m *mockDoer) Do(r *http.Request) (*http.Response, error) {
	args := m.Called(r)
	if v := args.Get(0); v != nil {
		return v.(*http.Response), args.Error(1)
	}
	return nil, args.Error(1)
}

func newJSONResp(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}

func TestGetMe_Wraps(t *testing.T) {
	m := &mockDoer{}
	m.On("Do", mock.MatchedBy(func(r *http.Request) bool {
		return strings.HasSuffix(r.URL.Path, "/getMe")
	})).Return(newJSONResp(200, `{"ok":true,"result":{"id":1,"is_bot":true,"first_name":"echo"}}`), nil)

	b := client.New("123:abc", client.WithHTTPClient(m))
	u, err := GetMe(context.Background(), b)
	require.NoError(t, err)
	require.Equal(t, int64(1), u.ID)
}

func TestSendMessage_Wraps(t *testing.T) {
	m := &mockDoer{}
	m.On("Do", mock.MatchedBy(func(r *http.Request) bool {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(r.Body)
		return strings.HasSuffix(r.URL.Path, "/sendMessage") &&
			strings.Contains(buf.String(), `"chat_id":42`) &&
			strings.Contains(buf.String(), `"text":"hi"`)
	})).Return(newJSONResp(200, `{"ok":true,"result":{"message_id":7,"date":0,"chat":{"id":42,"type":"private"},"text":"hi"}}`), nil)

	b := client.New("t", client.WithHTTPClient(m))
	msg, err := SendMessage(context.Background(), b, &SendMessageParams{ChatID: 42, Text: "hi"})
	require.NoError(t, err)
	require.Equal(t, int64(7), msg.MessageID)
}

func TestGetUpdates_Empty(t *testing.T) {
	m := &mockDoer{}
	m.On("Do", mock.Anything).Return(newJSONResp(200, `{"ok":true,"result":[]}`), nil)

	b := client.New("t", client.WithHTTPClient(m))
	ups, err := GetUpdates(context.Background(), b, &GetUpdatesParams{Limit: 10, Timeout: 0})
	require.NoError(t, err)
	require.Empty(t, ups)
}

func TestMethodNames_Coverage(t *testing.T) {
	require.ElementsMatch(t,
		[]string{"getMe", "getUpdates", "sendMessage", "setWebhook", "deleteWebhook", "answerCallbackQuery", "sendDocument"},
		methodNames(),
	)
}
```

- [ ] **Step 3: Run tests — PASS**

Run: `go test -race ./...`

- [ ] **Step 4: Commit**

```bash
git add api/ client/
git commit -m "feat(api): hand-coded method wrappers (getMe, sendMessage, getUpdates, ...)"
```

---

## Task 12 — Transport: `Updater` interface

**Files:**
- Create: `transport/updater.go`

- [ ] **Step 1: Implement**

```go
// Package transport provides update delivery mechanisms (long-poll and
// webhook) that feed updates into the dispatch package's Router.
//
// All implementations satisfy the Updater interface so user code can
// swap one for the other without touching handler logic.
package transport

import (
	"context"

	"github.com/lukaszraczylo/go-telegram/api"
)

// Updater is the abstraction over update sources. Implementations must:
//   - return a channel from Updates() that receives every Update they read.
//   - close the channel after Run returns.
//   - honour ctx cancellation in Run.
type Updater interface {
	// Updates returns the channel updates flow into. Multiple readers
	// is implementation-defined; users should treat it as single-reader.
	Updates() <-chan api.Update
	// Run blocks until ctx is cancelled or a fatal error occurs. It is
	// the user's responsibility to call Run in a goroutine if needed.
	Run(ctx context.Context) error
	// Stop signals Run to exit and waits for the channel to drain.
	// Implementations must be idempotent.
	Stop(ctx context.Context) error
}
```

- [ ] **Step 2: Commit**

```bash
git add transport/updater.go
git commit -m "feat(transport): define Updater interface"
```

---

## Task 13 — Transport: `LongPoller`

**Files:**
- Create: `transport/longpoll.go`
- Create: `transport/longpoll_test.go`
- Create: `transport/backoff.go`

- [ ] **Step 1: Implement `transport/backoff.go`**

```go
package transport

import (
	"math"
	"math/rand/v2"
	"time"
)

// BackoffStrategy returns the duration to wait before the next attempt
// after `attempt` consecutive failures (1-based). Implementations must
// be safe to call from a single goroutine.
type BackoffStrategy interface {
	NextDelay(attempt int) time.Duration
}

// ExponentialBackoff implements capped exponential back-off with jitter.
// Defaults: Base=500ms, Max=30s, Factor=2.0, Jitter=0.2.
type ExponentialBackoff struct {
	Base   time.Duration
	Max    time.Duration
	Factor float64
	Jitter float64 // 0..1; fraction of computed delay added/subtracted at random
}

// DefaultBackoff returns an ExponentialBackoff with library defaults.
func DefaultBackoff() *ExponentialBackoff {
	return &ExponentialBackoff{
		Base:   500 * time.Millisecond,
		Max:    30 * time.Second,
		Factor: 2.0,
		Jitter: 0.2,
	}
}

// NextDelay implements BackoffStrategy.
func (b *ExponentialBackoff) NextDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	d := float64(b.Base) * math.Pow(b.Factor, float64(attempt-1))
	if d > float64(b.Max) {
		d = float64(b.Max)
	}
	if b.Jitter > 0 {
		d *= 1 + (rand.Float64()*2-1)*b.Jitter
	}
	if d < 0 {
		d = 0
	}
	return time.Duration(d)
}
```

- [ ] **Step 2: Implement `transport/longpoll.go`**

```go
package transport

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/lukaszraczylo/go-telegram/api"
	"github.com/lukaszraczylo/go-telegram/client"
)

// LongPoller pulls updates via Bot.GetUpdates in a loop, advancing the
// offset cursor after each batch. It applies BackoffStrategy on transient
// errors (network failures, 5xx, 429).
type LongPoller struct {
	Bot          *client.Bot
	Timeout      int            // seconds, default 30
	Limit        int            // 1..100, default 100
	AllowedTypes []api.UpdateType
	Backoff      BackoffStrategy

	out  chan api.Update
	once sync.Once
	stop chan struct{}
}

// NewLongPoller constructs a LongPoller with sensible defaults.
func NewLongPoller(b *client.Bot) *LongPoller {
	return &LongPoller{
		Bot:     b,
		Timeout: 30,
		Limit:   100,
		Backoff: DefaultBackoff(),
		out:     make(chan api.Update, 64),
		stop:    make(chan struct{}),
	}
}

// Updates implements Updater.
func (p *LongPoller) Updates() <-chan api.Update { return p.out }

// Run implements Updater. It blocks until ctx is cancelled, Stop is
// called, or a fatal error occurs (e.g. unauthorized).
func (p *LongPoller) Run(ctx context.Context) error {
	defer close(p.out)

	var offset int64
	failures := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-p.stop:
			return nil
		default:
		}

		ups, err := api.GetUpdates(ctx, p.Bot, &api.GetUpdatesParams{
			Offset:         offset,
			Limit:          p.Limit,
			Timeout:        p.Timeout,
			AllowedUpdates: p.AllowedTypes,
		})
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return err
			}
			// Fatal: unauthorized -> bail.
			if errors.Is(err, client.ErrUnauthorized) {
				return err
			}
			failures++
			delay := p.Backoff.NextDelay(failures)
			select {
			case <-time.After(delay):
				continue
			case <-ctx.Done():
				return ctx.Err()
			case <-p.stop:
				return nil
			}
		}
		failures = 0

		for _, u := range ups {
			select {
			case p.out <- u:
				if u.UpdateID >= offset {
					offset = u.UpdateID + 1
				}
			case <-ctx.Done():
				return ctx.Err()
			case <-p.stop:
				return nil
			}
		}
	}
}

// Stop implements Updater.
func (p *LongPoller) Stop(ctx context.Context) error {
	p.once.Do(func() { close(p.stop) })
	return nil
}
```

- [ ] **Step 3: Write `transport/longpoll_test.go`**

```go
package transport

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lukaszraczylo/go-telegram/api"
	"github.com/lukaszraczylo/go-telegram/client"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockDoer struct{ mock.Mock }

func (m *mockDoer) Do(r *http.Request) (*http.Response, error) {
	args := m.Called(r)
	if v := args.Get(0); v != nil {
		return v.(*http.Response), args.Error(1)
	}
	return nil, args.Error(1)
}

func resp(body string) *http.Response {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}

func TestLongPoller_DeliversUpdatesAndAdvancesOffset(t *testing.T) {
	m := &mockDoer{}
	var calls atomic.Int32
	m.On("Do", mock.Anything).Return(func(r *http.Request) *http.Response {
		switch calls.Add(1) {
		case 1:
			return resp(`{"ok":true,"result":[{"update_id":10,"message":{"message_id":1,"date":0,"chat":{"id":1,"type":"private"},"text":"hi"}}]}`)
		case 2:
			return resp(`{"ok":true,"result":[{"update_id":11,"message":{"message_id":2,"date":0,"chat":{"id":1,"type":"private"},"text":"there"}}]}`)
		default:
			return resp(`{"ok":true,"result":[]}`)
		}
	}, nil)

	b := client.New("t", client.WithHTTPClient(m))
	p := NewLongPoller(b)
	p.Timeout = 0

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	go func() { _ = p.Run(ctx) }()

	u1 := <-p.Updates()
	require.Equal(t, int64(10), u1.UpdateID)
	u2 := <-p.Updates()
	require.Equal(t, int64(11), u2.UpdateID)
}

func TestLongPoller_BackoffOnNetworkError(t *testing.T) {
	m := &mockDoer{}
	var attempts atomic.Int32
	m.On("Do", mock.Anything).Return(func(r *http.Request) *http.Response {
		attempts.Add(1)
		return nil
	}, error(io.ErrUnexpectedEOF)).Maybe()

	b := client.New("t", client.WithHTTPClient(m))
	p := NewLongPoller(b)
	p.Timeout = 0
	p.Backoff = &ExponentialBackoff{Base: 5 * time.Millisecond, Max: 5 * time.Millisecond}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_ = p.Run(ctx)
	require.GreaterOrEqual(t, attempts.Load(), int32(2), "should retry at least once")
}

func TestLongPoller_StopCloses(t *testing.T) {
	m := &mockDoer{}
	m.On("Do", mock.Anything).Return(resp(`{"ok":true,"result":[]}`), nil).Maybe()

	b := client.New("t", client.WithHTTPClient(m))
	p := NewLongPoller(b)
	p.Timeout = 0

	ctx := context.Background()
	done := make(chan struct{})
	go func() { _ = p.Run(ctx); close(done) }()

	require.NoError(t, p.Stop(ctx))
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not exit after Stop")
	}

	// Channel must be closed.
	_, ok := <-p.Updates()
	require.False(t, ok, "expected closed channel after Stop")
}
```

- [ ] **Step 4: Run tests — PASS**

Run: `go test -race ./transport/...`

- [ ] **Step 5: Commit**

```bash
git add transport/
git commit -m "feat(transport): LongPoller with exponential backoff"
```

---

## Task 14 — Transport: `WebhookServer`

**Files:**
- Create: `transport/webhook.go`
- Create: `transport/webhook_test.go`

- [ ] **Step 1: Implement `transport/webhook.go`**

```go
package transport

import (
	"context"
	"crypto/subtle"
	"errors"
	"net"
	"net/http"
	"sync"

	"github.com/lukaszraczylo/go-telegram/api"
	"github.com/lukaszraczylo/go-telegram/client"
)

// WebhookServer implements Updater by exposing an http.Handler that
// receives updates from Telegram. It can be mounted on the user's own
// HTTP server (via ServeHTTP) or run standalone (via ListenAndServe).
type WebhookServer struct {
	Bot         *client.Bot
	SecretToken string // verify X-Telegram-Bot-Api-Secret-Token; empty disables
	BufferSize  int    // updates channel buffer; default 64

	out  chan api.Update
	once sync.Once
	stop chan struct{}

	srv *http.Server
}

// NewWebhookServer constructs a WebhookServer with default buffer size.
func NewWebhookServer(b *client.Bot) *WebhookServer {
	return &WebhookServer{
		Bot:        b,
		BufferSize: 64,
		out:        make(chan api.Update, 64),
		stop:       make(chan struct{}),
	}
}

// Updates implements Updater.
func (w *WebhookServer) Updates() <-chan api.Update { return w.out }

// Run implements Updater. It blocks until Stop is called or ctx is
// cancelled. If the server has not been started via ListenAndServe, Run
// only watches for shutdown — the user is expected to mount ServeHTTP
// on their own router.
func (w *WebhookServer) Run(ctx context.Context) error {
	defer close(w.out)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-w.stop:
		return nil
	}
}

// Stop implements Updater.
func (w *WebhookServer) Stop(ctx context.Context) error {
	w.once.Do(func() { close(w.stop) })
	if w.srv != nil {
		return w.srv.Shutdown(ctx)
	}
	return nil
}

// ServeHTTP implements http.Handler. Telegram POSTs each update as JSON
// to this endpoint. Non-POST requests get 405; bad bodies get 400; secret
// token mismatches get 401.
func (w *WebhookServer) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		rw.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if w.SecretToken != "" {
		got := r.Header.Get("X-Telegram-Bot-Api-Secret-Token")
		if subtle.ConstantTimeCompare([]byte(got), []byte(w.SecretToken)) != 1 {
			rw.WriteHeader(http.StatusUnauthorized)
			return
		}
	}
	defer r.Body.Close()

	var u api.Update
	codec := w.Bot.Codec()
	const max = 1 << 20 // 1 MiB cap on body
	buf := make([]byte, 0, 1024)
	tmp := make([]byte, 4096)
	for {
		n, err := r.Body.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
			if len(buf) > max {
				rw.WriteHeader(http.StatusRequestEntityTooLarge)
				return
			}
		}
		if errors.Is(err, http.ErrBodyReadAfterClose) || err != nil {
			break
		}
	}
	if err := codec.Unmarshal(buf, &u); err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	select {
	case w.out <- u:
	case <-w.stop:
	}

	rw.WriteHeader(http.StatusOK)
}

// ListenAndServe starts an HTTP server on addr and blocks until ctx is
// cancelled, Stop is called, or the server returns an error other than
// http.ErrServerClosed.
func (w *WebhookServer) ListenAndServe(ctx context.Context, addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/", w)
	w.srv = &http.Server{
		Addr:    addr,
		Handler: mux,
		BaseContext: func(net.Listener) context.Context { return ctx },
	}
	go func() {
		<-ctx.Done()
		_ = w.srv.Shutdown(context.Background())
	}()
	err := w.srv.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
```

- [ ] **Step 2: Write `transport/webhook_test.go`**

```go
package transport

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lukaszraczylo/go-telegram/client"
	"github.com/stretchr/testify/require"
)

func TestWebhook_DeliversUpdate(t *testing.T) {
	b := client.New("t")
	w := NewWebhookServer(b)
	w.SecretToken = "secret"

	srv := httptest.NewServer(w)
	t.Cleanup(srv.Close)

	body := `{"update_id":1,"message":{"message_id":1,"date":0,"chat":{"id":1,"type":"private"},"text":"hi"}}`
	req, _ := http.NewRequest(http.MethodPost, srv.URL, strings.NewReader(body))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "secret")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	select {
	case u := <-w.Updates():
		require.Equal(t, int64(1), u.UpdateID)
	case <-time.After(time.Second):
		t.Fatal("update not delivered")
	}
}

func TestWebhook_RejectsBadSecret(t *testing.T) {
	b := client.New("t")
	w := NewWebhookServer(b)
	w.SecretToken = "secret"

	srv := httptest.NewServer(w)
	t.Cleanup(srv.Close)

	req, _ := http.NewRequest(http.MethodPost, srv.URL, strings.NewReader(`{}`))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "wrong")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestWebhook_RejectsNonPOST(t *testing.T) {
	w := NewWebhookServer(client.New("t"))
	srv := httptest.NewServer(w)
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
}

func TestWebhook_RejectsBadJSON(t *testing.T) {
	w := NewWebhookServer(client.New("t"))
	srv := httptest.NewServer(w)
	t.Cleanup(srv.Close)

	resp, err := http.Post(srv.URL, "application/json", bytes.NewBufferString("not json"))
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestWebhook_StopExitsRun(t *testing.T) {
	w := NewWebhookServer(client.New("t"))

	done := make(chan struct{})
	go func() { _ = w.Run(context.Background()); close(done) }()

	require.NoError(t, w.Stop(context.Background()))
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not exit after Stop")
	}
}
```

- [ ] **Step 3: Run tests — PASS**

Run: `go test -race ./transport/...`

- [ ] **Step 4: Commit**

```bash
git add transport/webhook.go transport/webhook_test.go
git commit -m "feat(transport): WebhookServer with secret-token verification"
```

---

## Task 15 — Dispatcher: Context, Handler, Middleware

**Files:**
- Create: `dispatch/context.go`
- Create: `dispatch/handler.go`

- [ ] **Step 1: Implement `dispatch/context.go`**

```go
// Package dispatch provides a typed router for Telegram updates. It
// consumes any transport.Updater and dispatches updates to handlers
// registered by command, regex, or update-payload kind.
package dispatch

import (
	"context"

	"github.com/lukaszraczylo/go-telegram/api"
	"github.com/lukaszraczylo/go-telegram/client"
)

// Context bundles the per-update state every handler receives.
//
// Ctx is the request context propagated from Router.Run; cancelling the
// run cancels every handler.
//
// Bot is the API client. Handlers reply by calling api.SendMessage(c.Ctx,
// c.Bot, ...) etc.
//
// Update is the raw update; payload-typed handlers also receive a
// narrowed pointer to one of its sub-fields.
//
// Values is a per-update bag matchers populate. Conventional keys:
//   "command":      string, the matched bot command (e.g. "/start")
//   "command_args": string, everything after the command
//   "regex_match":  []string, regex sub-matches when OnText matches
type Context struct {
	Ctx    context.Context
	Bot    *client.Bot
	Update *api.Update
	Values map[string]any
}

// NewContext constructs a Context. Used by Router internally; exposed for
// custom test harnesses.
func NewContext(ctx context.Context, b *client.Bot, u *api.Update) *Context {
	return &Context{Ctx: ctx, Bot: b, Update: u, Values: map[string]any{}}
}
```

- [ ] **Step 2: Implement `dispatch/handler.go`**

```go
package dispatch

// Handler is a generic handler over update payload type T. T is typically
// *api.Message, *api.CallbackQuery, *api.InlineQuery, or *api.Update for
// global middleware.
type Handler[T any] func(ctx *Context, payload T) error

// Middleware wraps a Handler[T] with cross-cutting behaviour (logging,
// recovery, auth). Middleware composition is left-to-right: Use(a,b,c)
// runs as a(b(c(handler))).
type Middleware[T any] func(Handler[T]) Handler[T]

// Chain composes a slice of middleware into a single Middleware[T].
func Chain[T any](mws ...Middleware[T]) Middleware[T] {
	return func(h Handler[T]) Handler[T] {
		for i := len(mws) - 1; i >= 0; i-- {
			h = mws[i](h)
		}
		return h
	}
}
```

- [ ] **Step 3: Commit**

```bash
git add dispatch/context.go dispatch/handler.go
git commit -m "feat(dispatch): generic Handler[T] + Middleware[T] + Context"
```

---

## Task 16 — Dispatcher: Router with command/text/callback/inline matchers

**Files:**
- Create: `dispatch/router.go`
- Create: `dispatch/middleware.go` (panic recovery)
- Create: `dispatch/router_test.go`

- [ ] **Step 1: Implement `dispatch/middleware.go`**

```go
package dispatch

import (
	"fmt"
	"runtime/debug"

	"github.com/lukaszraczylo/go-telegram/api"
)

// Recovery returns middleware that recovers from panics in downstream
// handlers, converting them into a returned error and logging via the
// bot's configured logger. Registered automatically by NewRouter.
func Recovery() Middleware[*api.Update] {
	return func(next Handler[*api.Update]) Handler[*api.Update] {
		return func(c *Context, u *api.Update) (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("panic in handler: %v\n%s", r, debug.Stack())
					if c.Bot != nil {
						c.Bot.Logger().Error("dispatch recovered panic", "err", err)
					}
				}
			}()
			return next(c, u)
		}
	}
}
```

- [ ] **Step 2: Implement `dispatch/router.go`**

```go
package dispatch

import (
	"context"
	"regexp"
	"strings"

	"github.com/lukaszraczylo/go-telegram/api"
	"github.com/lukaszraczylo/go-telegram/client"
	"github.com/lukaszraczylo/go-telegram/transport"
)

// Router dispatches updates from any Updater to typed handlers.
//
// Matchers run in registration order; first match wins. A panic-recovery
// middleware is attached automatically and runs around every dispatch.
type Router struct {
	bot *client.Bot

	commands  []commandRoute
	texts     []textRoute
	callbacks []callbackRoute
	inlines   []Handler[*api.InlineQuery]
	editedMsg []Handler[*api.Message]

	globalMW []Middleware[*api.Update]
}

type commandRoute struct {
	cmd     string
	handler Handler[*api.Message]
}

type textRoute struct {
	re      *regexp.Regexp
	handler Handler[*api.Message]
}

type callbackRoute struct {
	re      *regexp.Regexp
	handler Handler[*api.CallbackQuery]
}

// New constructs a Router. Recovery middleware is added by default; users
// can disable it by passing WithoutRecovery (not implemented here, but
// the hook is in place via Use).
func New(b *client.Bot) *Router {
	r := &Router{bot: b}
	r.Use(Recovery())
	return r
}

// Use registers a global middleware applied to every Update dispatch.
func (r *Router) Use(mw Middleware[*api.Update]) { r.globalMW = append(r.globalMW, mw) }

// OnCommand registers a handler for a slash command. The command string
// includes the leading slash (e.g. "/start"). Matching strips an optional
// "@BotName" suffix.
func (r *Router) OnCommand(cmd string, h Handler[*api.Message]) {
	r.commands = append(r.commands, commandRoute{cmd: cmd, handler: h})
}

// OnText registers a handler for messages whose Text matches the regex.
func (r *Router) OnText(pattern string, h Handler[*api.Message]) {
	r.texts = append(r.texts, textRoute{re: regexp.MustCompile(pattern), handler: h})
}

// OnCallback registers a handler for callback queries whose Data matches
// the regex.
func (r *Router) OnCallback(pattern string, h Handler[*api.CallbackQuery]) {
	r.callbacks = append(r.callbacks, callbackRoute{re: regexp.MustCompile(pattern), handler: h})
}

// OnInlineQuery registers a handler for inline queries (one matcher only;
// inline queries are not partitioned by content here).
func (r *Router) OnInlineQuery(h Handler[*api.InlineQuery]) {
	r.inlines = append(r.inlines, h)
}

// OnEditedMessage registers a handler for edited message updates.
func (r *Router) OnEditedMessage(h Handler[*api.Message]) {
	r.editedMsg = append(r.editedMsg, h)
}

// Run consumes the Updater and dispatches each update. It blocks until
// the Updater's channel is closed (i.e. the underlying Run returned).
func (r *Router) Run(ctx context.Context, u transport.Updater) error {
	go func() { _ = u.Run(ctx) }()

	root := r.dispatch
	for i := len(r.globalMW) - 1; i >= 0; i-- {
		root = r.globalMW[i](root)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case up, ok := <-u.Updates():
			if !ok {
				return nil
			}
			c := NewContext(ctx, r.bot, &up)
			_ = root(c, &up)
		}
	}
}

func (r *Router) dispatch(c *Context, u *api.Update) error {
	switch {
	case u.Message != nil:
		return r.handleMessage(c, u.Message)
	case u.EditedMessage != nil:
		for _, h := range r.editedMsg {
			if err := h(c, u.EditedMessage); err != nil {
				return err
			}
			break
		}
	case u.CallbackQuery != nil:
		return r.handleCallback(c, u.CallbackQuery)
	case u.InlineQuery != nil:
		for _, h := range r.inlines {
			if err := h(c, u.InlineQuery); err != nil {
				return err
			}
			break
		}
	}
	return nil
}

func (r *Router) handleMessage(c *Context, m *api.Message) error {
	// Try command first (entity-aware).
	if cmd, args, ok := extractCommand(m); ok {
		c.Values["command"] = cmd
		c.Values["command_args"] = args
		for _, route := range r.commands {
			if route.cmd == cmd {
				return route.handler(c, m)
			}
		}
	}
	// Then text regex matchers.
	if m.Text != "" {
		for _, route := range r.texts {
			if subs := route.re.FindStringSubmatch(m.Text); subs != nil {
				c.Values["regex_match"] = subs
				return route.handler(c, m)
			}
		}
	}
	return nil
}

func (r *Router) handleCallback(c *Context, q *api.CallbackQuery) error {
	for _, route := range r.callbacks {
		if subs := route.re.FindStringSubmatch(q.Data); subs != nil {
			c.Values["regex_match"] = subs
			return route.handler(c, q)
		}
	}
	return nil
}

// extractCommand returns the command (e.g. "/start") and the remaining
// argument string, when m carries a leading bot_command entity. It strips
// optional "@BotName" suffix on the command itself.
func extractCommand(m *api.Message) (cmd, args string, ok bool) {
	if len(m.Entities) == 0 || m.Text == "" {
		return "", "", false
	}
	first := m.Entities[0]
	if first.Type != api.EntityBotCommand || first.Offset != 0 {
		return "", "", false
	}
	end := first.Offset + first.Length
	cmd = m.Text[first.Offset:end]
	if i := strings.Index(cmd, "@"); i >= 0 {
		cmd = cmd[:i]
	}
	args = strings.TrimSpace(m.Text[end:])
	return cmd, args, true
}
```

- [ ] **Step 3: Implement `dispatch/router_test.go`**

```go
package dispatch

import (
	"context"
	"testing"
	"time"

	"github.com/lukaszraczylo/go-telegram/api"
	"github.com/lukaszraczylo/go-telegram/client"
	"github.com/stretchr/testify/require"
)

// fakeUpdater feeds a fixed slice of updates then closes.
type fakeUpdater struct{ ch chan api.Update }

func newFake(ups ...api.Update) *fakeUpdater {
	ch := make(chan api.Update, len(ups))
	for _, u := range ups {
		ch <- u
	}
	close(ch)
	return &fakeUpdater{ch: ch}
}

func (f *fakeUpdater) Updates() <-chan api.Update           { return f.ch }
func (f *fakeUpdater) Run(ctx context.Context) error        { <-ctx.Done(); return ctx.Err() }
func (f *fakeUpdater) Stop(ctx context.Context) error       { return nil }

func cmdMessage(text string) api.Update {
	return api.Update{
		UpdateID: 1,
		Message: &api.Message{
			MessageID: 1, Date: 0, Chat: api.Chat{ID: 1, Type: api.ChatTypePrivate},
			Text:     text,
			Entities: []api.MessageEntity{{Type: api.EntityBotCommand, Offset: 0, Length: indexEnd(text)}},
		},
	}
}

func indexEnd(text string) int {
	for i, r := range text {
		if r == ' ' {
			return i
		}
	}
	return len(text)
}

func TestRouter_OnCommandMatches(t *testing.T) {
	b := client.New("t")
	r := New(b)
	hit := make(chan string, 1)
	r.OnCommand("/start", func(c *Context, m *api.Message) error {
		hit <- c.Values["command"].(string)
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	go func() { _ = r.Run(ctx, newFake(cmdMessage("/start"))) }()

	require.Equal(t, "/start", <-hit)
}

func TestRouter_OnCommandStripsBotName(t *testing.T) {
	r := New(client.New("t"))
	hit := make(chan string, 1)
	r.OnCommand("/start", func(c *Context, m *api.Message) error {
		hit <- "matched"
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	go func() { _ = r.Run(ctx, newFake(cmdMessage("/start@MyBot hello"))) }()

	require.Equal(t, "matched", <-hit)
}

func TestRouter_OnText(t *testing.T) {
	r := New(client.New("t"))
	hit := make(chan []string, 1)
	r.OnText(`^hello (\w+)$`, func(c *Context, m *api.Message) error {
		hit <- c.Values["regex_match"].([]string)
		return nil
	})

	u := api.Update{UpdateID: 1, Message: &api.Message{
		MessageID: 1, Chat: api.Chat{ID: 1, Type: "private"}, Text: "hello world",
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	go func() { _ = r.Run(ctx, newFake(u)) }()

	subs := <-hit
	require.Equal(t, "world", subs[1])
}

func TestRouter_OnCallback(t *testing.T) {
	r := New(client.New("t"))
	hit := make(chan string, 1)
	r.OnCallback(`^like:(\d+)$`, func(c *Context, q *api.CallbackQuery) error {
		hit <- q.Data
		return nil
	})

	u := api.Update{UpdateID: 1, CallbackQuery: &api.CallbackQuery{
		ID: "x", From: api.User{ID: 1}, ChatInstance: "y", Data: "like:42",
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	go func() { _ = r.Run(ctx, newFake(u)) }()

	require.Equal(t, "like:42", <-hit)
}

func TestRouter_NoMatch(t *testing.T) {
	r := New(client.New("t"))
	called := false
	r.OnCommand("/start", func(c *Context, m *api.Message) error {
		called = true
		return nil
	})
	u := api.Update{UpdateID: 1, Message: &api.Message{Text: "no command"}}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_ = r.Run(ctx, newFake(u))
	require.False(t, called)
}

func TestRouter_PanicRecovery(t *testing.T) {
	r := New(client.New("t"))
	r.OnCommand("/boom", func(c *Context, m *api.Message) error {
		panic("kaboom")
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	// Should not propagate panic to Run.
	require.NotPanics(t, func() { _ = r.Run(ctx, newFake(cmdMessage("/boom"))) })
}

func TestRouter_MiddlewareOrder(t *testing.T) {
	r := New(client.New("t"))
	var order []string
	r.Use(func(next Handler[*api.Update]) Handler[*api.Update] {
		return func(c *Context, u *api.Update) error {
			order = append(order, "before-1")
			err := next(c, u)
			order = append(order, "after-1")
			return err
		}
	})
	r.Use(func(next Handler[*api.Update]) Handler[*api.Update] {
		return func(c *Context, u *api.Update) error {
			order = append(order, "before-2")
			err := next(c, u)
			order = append(order, "after-2")
			return err
		}
	})
	r.OnCommand("/x", func(c *Context, m *api.Message) error {
		order = append(order, "handler")
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_ = r.Run(ctx, newFake(cmdMessage("/x")))

	require.Equal(t,
		[]string{"before-1", "before-2", "handler", "after-2", "after-1"},
		order)
}
```

- [ ] **Step 4: Run tests — PASS**

Run: `go test -race ./dispatch/...`

- [ ] **Step 5: Commit**

```bash
git add dispatch/
git commit -m "feat(dispatch): Router with command/text/callback/inline matchers"
```

---

## Task 17 — Echo example

**Files:**
- Create: `examples/echo/main.go`
- Create: `examples/echo/README.md`

- [ ] **Step 1: Implement `examples/echo/main.go`**

```go
// Package main is a long-poll echo bot. Run with:
//
//	TELEGRAM_BOT_TOKEN=xxx go run ./examples/echo
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

	bot := client.New(token)
	me, err := api.GetMe(ctx, bot)
	if err != nil {
		log.Fatalf("getMe: %v", err)
	}
	log.Printf("running as @%s", me.Username)

	router := dispatch.New(bot)
	router.OnCommand("/start", func(c *dispatch.Context, m *api.Message) error {
		_, err := api.SendMessage(c.Ctx, c.Bot, &api.SendMessageParams{
			ChatID: m.Chat.ID,
			Text:   fmt.Sprintf("hello %s, send me anything to echo", m.From.FirstName),
		})
		return err
	})
	router.OnText(`.+`, func(c *dispatch.Context, m *api.Message) error {
		_, err := api.SendMessage(c.Ctx, c.Bot, &api.SendMessageParams{
			ChatID: m.Chat.ID,
			Text:   m.Text,
			ReplyToMessageID: m.MessageID,
		})
		return err
	})

	poller := transport.NewLongPoller(bot)
	if err := router.Run(ctx, poller); err != nil && err != context.Canceled {
		log.Printf("router exited: %v", err)
	}
}
```

- [ ] **Step 2: Add example README**

`examples/echo/README.md`:

```markdown
# echo

Long-poll echo bot. Replies to `/start` with a greeting and echoes any other text.

## Run

```bash
export TELEGRAM_BOT_TOKEN=123456:ABC...
go run ./examples/echo
```
```

- [ ] **Step 3: Verify build**

Run: `go build ./examples/echo/...`
Expected: success.

- [ ] **Step 4: Commit**

```bash
git add examples/echo/
git commit -m "docs(examples): add long-poll echo bot"
```

---

## Task 18 — Webhook example

**Files:**
- Create: `examples/webhook/main.go`
- Create: `examples/webhook/README.md`

- [ ] **Step 1: Implement `examples/webhook/main.go`**

```go
// Package main is a webhook bot. Run with:
//
//	TELEGRAM_BOT_TOKEN=xxx \
//	WEBHOOK_URL=https://example.com/bot \
//	WEBHOOK_SECRET=somethingrandom \
//	go run ./examples/webhook
//
// The bot sets its webhook to WEBHOOK_URL on startup, listens on :8080,
// and clears the webhook on shutdown.
package main

import (
	"context"
	"log"
	"net/http"
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
	url := os.Getenv("WEBHOOK_URL")
	secret := os.Getenv("WEBHOOK_SECRET")
	if token == "" || url == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN and WEBHOOK_URL required")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	bot := client.New(token)

	if _, err := api.SetWebhook(ctx, bot, &api.SetWebhookParams{
		URL:         url,
		SecretToken: secret,
	}); err != nil {
		log.Fatalf("setWebhook: %v", err)
	}
	defer func() {
		_, _ = api.DeleteWebhook(context.Background(), bot, &api.DeleteWebhookParams{})
	}()

	wh := transport.NewWebhookServer(bot)
	wh.SecretToken = secret

	router := dispatch.New(bot)
	router.OnCommand("/ping", func(c *dispatch.Context, m *api.Message) error {
		_, err := api.SendMessage(c.Ctx, c.Bot, &api.SendMessageParams{
			ChatID: m.Chat.ID,
			Text:   "pong",
		})
		return err
	})

	mux := http.NewServeMux()
	mux.Handle("/bot", wh)
	srv := &http.Server{Addr: ":8080", Handler: mux}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("http server exited: %v", err)
			stop()
		}
	}()

	if err := router.Run(ctx, wh); err != nil && err != context.Canceled {
		log.Printf("router exited: %v", err)
	}
	_ = srv.Shutdown(context.Background())
}
```

- [ ] **Step 2: Add example README**

`examples/webhook/README.md`:

```markdown
# webhook

Bot using HTTPS webhooks. Replies to `/ping` with `pong`.

## Run

You need a public HTTPS endpoint pointed at port 8080. For local development use a tunnel like Cloudflare Tunnel or ngrok.

```bash
export TELEGRAM_BOT_TOKEN=123456:ABC...
export WEBHOOK_URL=https://your.tunnel.example/bot
export WEBHOOK_SECRET=randomsecret123
go run ./examples/webhook
```
```

- [ ] **Step 3: Verify build**

Run: `go build ./examples/webhook/...`

- [ ] **Step 4: Commit**

```bash
git add examples/webhook/
git commit -m "docs(examples): add webhook bot"
```

---

## Task 19 — Integration test suite (gated)

**Files:**
- Create: `test/integration/integration_test.go`
- Create: `test/integration/README.md`

- [ ] **Step 1: Implement `test/integration/integration_test.go`**

```go
//go:build integration

// Package integration_test contains tests that hit the live Telegram Bot
// API. These tests are gated behind the "integration" build tag and the
// TELEGRAM_BOT_TOKEN environment variable; they do not run on default
// `go test ./...`.
package integration_test

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/lukaszraczylo/go-telegram/api"
	"github.com/lukaszraczylo/go-telegram/client"
	"github.com/stretchr/testify/require"
)

func botFromEnv(t *testing.T) *client.Bot {
	tok := os.Getenv("TELEGRAM_BOT_TOKEN")
	if tok == "" {
		t.Skip("TELEGRAM_BOT_TOKEN not set")
	}
	return client.New(tok)
}

func TestGetMe(t *testing.T) {
	b := botFromEnv(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	u, err := api.GetMe(ctx, b)
	require.NoError(t, err)
	require.True(t, u.IsBot)
}

func TestSendMessage(t *testing.T) {
	b := botFromEnv(t)
	chatRaw := os.Getenv("TELEGRAM_TEST_CHAT_ID")
	if chatRaw == "" {
		t.Skip("TELEGRAM_TEST_CHAT_ID not set")
	}
	chatID, err := strconv.ParseInt(chatRaw, 10, 64)
	require.NoError(t, err, "TELEGRAM_TEST_CHAT_ID must be an integer")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	msg, err := api.SendMessage(ctx, b, &api.SendMessageParams{
		ChatID: chatID,
		Text:   "integration test " + time.Now().UTC().Format(time.RFC3339),
	})
	require.NoError(t, err)
	require.NotZero(t, msg.MessageID)
}

func TestWebhookCycle(t *testing.T) {
	b := botFromEnv(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Make sure no webhook is set first (long-poll mode).
	_, _ = api.DeleteWebhook(ctx, b, &api.DeleteWebhookParams{})

	ok, err := api.SetWebhook(ctx, b, &api.SetWebhookParams{URL: "https://example.invalid/no-receive"})
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = api.DeleteWebhook(ctx, b, &api.DeleteWebhookParams{})
	require.NoError(t, err)
	require.True(t, ok)
}
```

- [ ] **Step 2: Add `test/integration/README.md`**

```markdown
# Integration tests

Live tests against the real Telegram Bot API. Skipped by default.

## Run

```bash
export TELEGRAM_BOT_TOKEN=test_bot_token_here
export TELEGRAM_TEST_CHAT_ID=123456789  # a chat the bot can post in
make integration
```

The suite covers `getMe`, `sendMessage`, and the `setWebhook`/`deleteWebhook` cycle.
```

- [ ] **Step 3: Verify build with tag**

Run: `go vet -tags=integration ./test/integration/...` and `go build -tags=integration ./test/integration/...`
Expected: success.

- [ ] **Step 4: Verify default suite still passes (without tag) and skips this file**

Run: `go test ./...`
Expected: PASS, no integration tests run.

- [ ] **Step 5: Commit**

```bash
git add test/
git commit -m "test(integration): add gated suite for live API smoke checks"
```

---

## Task 20 — README expansion

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Replace stub README with full content**

Overwrite `README.md` with:

```markdown
# go-telegram

A Go library for the Telegram Bot API with pluggable HTTP transport, pluggable JSON codec, long-poll and webhook delivery, and a typed dispatcher.

[![CI](https://github.com/lukaszraczylo/go-telegram/actions/workflows/ci.yml/badge.svg)](https://github.com/lukaszraczylo/go-telegram/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/lukaszraczylo/go-telegram.svg)](https://pkg.go.dev/github.com/lukaszraczylo/go-telegram)

## Why

Most existing Go Telegram libraries either bring large transitive dependency trees or hard-wire `encoding/json` and `net/http`. This library:

- Uses the standard library only in production code paths.
- Treats the HTTP transport (`HTTPDoer`) and JSON codec (`Codec`) as plug points so you can swap in `valyala/fasthttp`, `goccy/go-json`, `bytedance/sonic`, etc., without forking the library.
- Funnels every API call through one generic helper (`client.Call[Req, Resp]`), keeping the per-method surface tiny and consistent.
- Ships a typed dispatcher (`dispatch.Router`) with command, regex, callback, and inline-query matchers.
- Plans to regenerate its API surface from the live Telegram docs (see `docs/superpowers/specs/`); in v1 a curated subset is hand-coded.

## Install

```bash
go get github.com/lukaszraczylo/go-telegram
```

## Quick start (long-poll echo bot)

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/lukaszraczylo/go-telegram/api"
    "github.com/lukaszraczylo/go-telegram/client"
    "github.com/lukaszraczylo/go-telegram/dispatch"
    "github.com/lukaszraczylo/go-telegram/transport"
)

func main() {
    bot := client.New(os.Getenv("TELEGRAM_BOT_TOKEN"))

    r := dispatch.New(bot)
    r.OnCommand("/start", func(c *dispatch.Context, m *api.Message) error {
        _, err := api.SendMessage(c.Ctx, c.Bot, &api.SendMessageParams{
            ChatID: m.Chat.ID, Text: "hello",
        })
        return err
    })

    if err := r.Run(context.Background(), transport.NewLongPoller(bot)); err != nil {
        log.Fatal(err)
    }
}
```

See [`examples/echo`](examples/echo) and [`examples/webhook`](examples/webhook) for full programs.

## Custom HTTP and JSON

```go
import (
    "github.com/goccy/go-json"
    "github.com/lukaszraczylo/go-telegram/client"
)

type goccyCodec struct{}
func (goccyCodec) Marshal(v any) ([]byte, error)        { return json.Marshal(v) }
func (goccyCodec) Unmarshal(data []byte, v any) error    { return json.Unmarshal(data, v) }

bot := client.New(token,
    client.WithCodec(goccyCodec{}),
    client.WithHTTPClient(myFasthttpAdapter{}),
)
```

`WithLogger` accepts any `Logger` — `slog.Logger` satisfies it via a thin shim.

## Webhooks

```go
wh := transport.NewWebhookServer(bot)
wh.SecretToken = "the-secret-from-setWebhook"

mux := http.NewServeMux()
mux.Handle("/bot", wh)
go http.ListenAndServe(":8080", mux)

router := dispatch.New(bot)
// ... register handlers ...
router.Run(ctx, wh)
```

## Dispatcher

- `OnCommand("/start", h)` — matches messages whose first entity is a `bot_command`. Strips `@BotName` suffix.
- `OnText("^hello (\\w+)$", h)` — regex on `Message.Text`. Captures available via `c.Values["regex_match"]`.
- `OnCallback("^like:(\\d+)$", h)` — regex on `CallbackQuery.Data`.
- `OnInlineQuery(h)`, `OnEditedMessage(h)`.
- `Use(mw)` — typed `Middleware[*api.Update]` chain. Panic-recovery middleware is registered automatically.

Handlers receive a `*dispatch.Context` (carrying `Ctx`, `Bot`, `Update`, and a `Values` bag) and a typed payload.

## Error handling

```go
msg, err := api.SendMessage(ctx, bot, ...)
if err != nil {
    var ae *client.APIError
    if errors.As(err, &ae) {
        if ae.IsRetryable() {
            time.Sleep(ae.RetryAfter())
        }
    }
    if errors.Is(err, client.ErrChatNotFound) { /* ... */ }
}
```

Sentinels: `ErrUnauthorized`, `ErrForbidden`, `ErrTooManyRequests`, `ErrChatNotFound`, `ErrMessageNotModified`, `ErrBadRequest`. Network failures wrap as `*client.NetworkError`; JSON decode failures wrap as `*client.ParseError`.

## Testing your bot

`client.HTTPDoer` is the only thing you need to mock:

```go
type fakeDoer struct{ ... }
func (f *fakeDoer) Do(*http.Request) (*http.Response, error) { ... }

bot := client.New("token", client.WithHTTPClient(&fakeDoer{}))
```

This library's own tests use `testify/mock` on this exact interface; see `client/call_test.go`.

## Updating

The hand-coded API slice in `api/` covers the ~8 methods needed for the example bots. Codegen (Plan 2) will replace it with the full Telegram surface generated from `https://core.telegram.org/bots/api`. Until then, contributions adding methods follow the conventions documented in `docs/superpowers/plans/2026-05-08-go-telegram-core.md`.

## Contributing

```bash
make test          # unit tests
make test-race     # with race detector
make lint          # vet + staticcheck
make integration   # live tests (TELEGRAM_BOT_TOKEN required)
```

## License

MIT — see [LICENSE](LICENSE).
```

- [ ] **Step 2: Commit**

```bash
git add README.md
git commit -m "docs(readme): full README with quick start, errors, dispatcher, testing"
```

---

## Task 21 — godoc audit + final lint pass

**Files:**
- Modify: any package with missing doc comments.

- [ ] **Step 1: Run lint and audit doc comments**

Run: `go vet ./... && staticcheck ./... && go test -race ./...`
Expected: all clean.

- [ ] **Step 2: Verify every exported symbol has a doc comment**

Run:

```bash
go doc -all ./client | grep -B1 '^func\|^type' | grep -v '^--$' || true
```

Visually inspect for any exported symbol whose preceding line is empty. Add a doc comment for any that are missing one. Common omissions to check:

- `client.Logger` methods — already documented (interface).
- `transport.Updater` — documented.
- `dispatch.Handler[T]` / `Middleware[T]` — documented.

- [ ] **Step 3: Add `CHANGELOG.md`**

Create `CHANGELOG.md`:

```markdown
# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Added
- Pluggable HTTP transport (`client.HTTPDoer`) and JSON codec (`client.Codec`).
- Generic call helper `client.Call[Req, Resp]` with multipart support.
- Typed errors (`*APIError` with sentinel unwrapping, `*NetworkError`, `*ParseError`).
- Long-poll (`transport.LongPoller`) and webhook (`transport.WebhookServer`) updaters.
- Generic dispatcher (`dispatch.Router`) with command, regex, callback, inline-query matchers, panic recovery.
- Hand-coded `api/` slice covering `getMe`, `getUpdates`, `sendMessage`, `setWebhook`, `deleteWebhook`, `answerCallbackQuery`, `sendDocument`.
- Echo and webhook example bots.
- Integration test suite gated by `integration` build tag.
```

- [ ] **Step 4: Commit**

```bash
git add CHANGELOG.md
git commit -m "docs: add CHANGELOG (unreleased section)"
```

---

## Task 22 — Final verification + tag

**Files:** none.

- [ ] **Step 1: Full clean build + test pass**

```bash
go mod tidy
go vet ./...
staticcheck ./...
go test -race -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | tail -n 1
```

Expected: green. Coverage of hand-written packages (`client`, `transport`, `dispatch`, `api`) should be ≥ 80%.

- [ ] **Step 2: Build all examples**

```bash
go build ./examples/echo
go build ./examples/webhook
```

Expected: success.

- [ ] **Step 3: Commit any final tidy changes**

```bash
git add -A
git diff --cached --stat
git commit -m "chore: go mod tidy" || true
```

- [ ] **Step 4: Tag v0.1.0-core**

```bash
git tag -a v0.1.0-core -m "Plan 1 complete: hand-written client + transport + dispatcher + curated api slice"
```

---

## Self-review notes

This plan covers spec §§1–10 and §§13–14 (acceptance criteria for v1 minus codegen). Plan 2 covers spec §§6 (codegen pipeline) and §11 (handling API changes via shape tests + golden refresh).

**Spec coverage check:**
- §4 layout — Tasks 1, 3 (foundation), 4–10 (`client/`), 12–14 (`transport/`), 15–16 (`dispatch/`).
- §5 client — Tasks 4–9.
- §6 codegen — deferred to Plan 2; `internal/spec/ir.go` defined now (Task 3) so Plan 2 has nothing to design.
- §7 transport — Tasks 12–14.
- §8 dispatcher — Tasks 15–16.
- §9 testing — Tasks 4–16 (per-package), Task 19 (integration), shape tests deferred to Plan 2 (§9.2 in spec is about codegen).
- §10 CI — Task 2.
- §11 handling API changes — deferred to Plan 2.
- §12 future work — out of scope.
- §13 dependency policy — enforced by Tasks 4–9 (stdlib only) and Task 22 (`go mod tidy`).
- §14 acceptance — covered by Tasks 17–22.

**Plan 2 entry point:** with `internal/spec/ir.go` already defined, Plan 2 begins by writing `cmd/scrape` against a tiny HTML fixture (TDD), then expands to the live snapshot, then writes `cmd/genapi` templates, then re-runs against live to replace the hand-coded `api/`.
