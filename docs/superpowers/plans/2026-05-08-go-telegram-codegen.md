# go-telegram Codegen Implementation Plan (Plan 2)

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development. Tasks use checkbox (`- [ ]`) syntax.

**Goal:** Build the two-stage codegen pipeline (`cmd/scrape` → `internal/spec/api.json` → `cmd/genapi` → `api/*.gen.go`), regenerate the full Telegram Bot API surface from a committed HTML snapshot, replace the hand-coded `api/` slice from Plan 1 with the generated code, and wire `make snapshot` / `make regen` / `make regen-from-fixture` / `make test-update-golden`.

**Architecture:** Stage 1 walks `<h4>`-anchored sections of the docs page and emits IR. Stage 2 reads IR and renders Go via `text/template` + `go/format`. Determinism is enforced by golden tests on both stages plus a CI clean-diff check.

**Tech Stack:** Go 1.23+, `golang.org/x/net/html` (added in P2.T1), `text/template`, `go/format`, `mime/multipart`, `encoding/json`. No new runtime deps.

**Reference:** [Spec §6, §11](../specs/2026-05-08-go-telegram-design.md). [Plan 1](2026-05-08-go-telegram-core.md) for the IR types, client, transport, dispatch packages.

---

## Conventions for every task

- Work from repo root (`/Users/nvm/Documents/projects/private/go-telegram`).
- Branch: `feat/plan-2-codegen` (forked from `feat/plan-1-core` after Plan 1).
- Run `go test -race ./...` after every task.
- Commit at end of every task with the message shown.
- DO NOT reorder struct fields. DO NOT add `//nolint` directives. `.golangci.yml` already disables fieldalignment.
- DO NOT modify Plan 1 files in `client/`, `transport/`, `dispatch/` unless a task explicitly says so.
- IR types in `internal/spec/ir.go` are stable from P1.T3 — extend only if a task explicitly says so.

---

## Task 1 — Scaffold cmd/scrape, cmd/genapi, add x/net/html

**Files:**
- Create: `cmd/scrape/main.go` (skeleton)
- Create: `cmd/genapi/main.go` (skeleton)
- Modify: `go.mod`, `go.sum` (add `golang.org/x/net/html`)
- Create: `testdata/html/.gitkeep` (placeholder; fixture lands in T2)
- Create: `testdata/golden/.gitkeep`

- [ ] **Step 1: Add dependency**

```bash
go get golang.org/x/net/html
```

- [ ] **Step 2: Scaffold `cmd/scrape/main.go`**

```go
// Command scrape parses the Telegram Bot API HTML page into the IR
// (internal/spec.API) and writes it to internal/spec/api.json.
//
// Usage:
//   scrape -input <file>            (read HTML from local file)
//   scrape -url   <url>             (fetch HTML from URL; default: live docs)
//   scrape -output <file>           (output path; default: internal/spec/api.json)
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/lukaszraczylo/go-telegram/internal/spec"
)

const defaultURL = "https://core.telegram.org/bots/api"

func main() {
	input := flag.String("input", "", "local HTML file (overrides -url)")
	url := flag.String("url", defaultURL, "URL to fetch HTML from")
	output := flag.String("output", "internal/spec/api.json", "output path")
	flag.Parse()

	if err := run(*input, *url, *output); err != nil {
		fmt.Fprintln(os.Stderr, "scrape:", err)
		os.Exit(1)
	}
}

func run(input, url, output string) error {
	htmlBytes, err := readHTML(input, url)
	if err != nil {
		return fmt.Errorf("read html: %w", err)
	}

	api, err := scrape(htmlBytes)
	if err != nil {
		return fmt.Errorf("scrape: %w", err)
	}

	return writeJSON(output, api)
}

func readHTML(input, url string) ([]byte, error) {
	if input != "" {
		return os.ReadFile(input)
	}
	c := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "go-telegram codegen scraper")
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// scrape and writeJSON are filled in by later tasks.
// scrape returns spec.API parsed from html.
// writeJSON writes the IR to path with stable formatting.
func scrape(html []byte) (*spec.API, error) {
	return nil, errors.New("scrape: not implemented (see P2.T3)")
}

func writeJSON(path string, api *spec.API) error {
	return errors.New("writeJSON: not implemented (see P2.T6)")
}
```

- [ ] **Step 3: Scaffold `cmd/genapi/main.go`**

```go
// Command genapi reads internal/spec/api.json and emits api/*.gen.go.
//
// Usage:
//   genapi -input <file>     (default: internal/spec/api.json)
//   genapi -outdir <dir>     (default: api)
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
)

func main() {
	input := flag.String("input", "internal/spec/api.json", "IR JSON path")
	outdir := flag.String("outdir", "api", "output directory")
	flag.Parse()

	if err := run(*input, *outdir); err != nil {
		fmt.Fprintln(os.Stderr, "genapi:", err)
		os.Exit(1)
	}
}

// run is filled in by P2.T8/T9/T10.
func run(input, outdir string) error {
	return errors.New("genapi: not implemented (see P2.T8)")
}
```

- [ ] **Step 4: Verify build**

```bash
go build ./cmd/...
```

Both binaries should build without error. They will exit non-zero at runtime until later tasks fill them in — that's fine.

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum cmd/ testdata/
git commit -m "chore(codegen): scaffold cmd/scrape and cmd/genapi"
```

---

## Task 2 — HTML fixture + snapshot capture script

**Files:**
- Create: `testdata/html/small_fixture.html`
- Create: `testdata/html/snapshot_2026-05-08.html` (capture from live URL)
- Create: `scripts/snapshot.sh`

The small fixture is hand-crafted, deterministic, and exercises every parser branch we care about (type with required+optional fields, type with array field, method with params + return, method with file param triggering HasFiles, oneof discriminator).

- [ ] **Step 1: Write `testdata/html/small_fixture.html`**

```html
<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>fixture</title></head><body>
<div id="dev_page_content">

<h3>Recent changes</h3>
<h4>Bot API 7.10</h4>
<p>Test fixture; not a real release.</p>

<h3>Available types</h3>

<h4><a class="anchor" href="#user" name="user"></a>User</h4>
<p>This object represents a Telegram user or bot.</p>
<table class="table">
<thead><tr><th>Field</th><th>Type</th><th>Description</th></tr></thead>
<tbody>
<tr><td>id</td><td>Integer</td><td>Unique identifier.</td></tr>
<tr><td>is_bot</td><td>Boolean</td><td>True, if this user is a bot.</td></tr>
<tr><td>first_name</td><td>String</td><td>User's or bot's first name.</td></tr>
<tr><td>last_name</td><td>String</td><td><em>Optional</em>. User's or bot's last name.</td></tr>
</tbody>
</table>

<h4><a class="anchor" href="#chatmember" name="chatmember"></a>ChatMember</h4>
<p>This object contains information about one member of a chat.
Currently, the following 6 types of chat members are supported:</p>
<ul>
<li><a href="#chatmemberowner">ChatMemberOwner</a></li>
<li><a href="#chatmemberadministrator">ChatMemberAdministrator</a></li>
</ul>

<h3>Available methods</h3>

<h4><a class="anchor" href="#getme" name="getme"></a>getMe</h4>
<p>A simple method for testing your bot's authentication token. Requires
no parameters. Returns basic information about the bot in form of a <a href="#user">User</a> object.</p>

<h4><a class="anchor" href="#sendmessage" name="sendmessage"></a>sendMessage</h4>
<p>Use this method to send text messages. On success, the sent <a href="#message">Message</a> is returned.</p>
<table class="table">
<thead><tr><th>Parameter</th><th>Type</th><th>Required</th><th>Description</th></tr></thead>
<tbody>
<tr><td>chat_id</td><td>Integer or String</td><td>Yes</td><td>Unique identifier for the target chat.</td></tr>
<tr><td>text</td><td>String</td><td>Yes</td><td>Text of the message to be sent.</td></tr>
<tr><td>parse_mode</td><td>String</td><td>Optional</td><td>Mode for parsing entities in the message text.</td></tr>
</tbody>
</table>

<h4><a class="anchor" href="#senddocument" name="senddocument"></a>sendDocument</h4>
<p>Use this method to send general files. On success, the sent <a href="#message">Message</a> is returned.</p>
<table class="table">
<thead><tr><th>Parameter</th><th>Type</th><th>Required</th><th>Description</th></tr></thead>
<tbody>
<tr><td>chat_id</td><td>Integer</td><td>Yes</td><td>Unique identifier for the target chat.</td></tr>
<tr><td>document</td><td><a href="#inputfile">InputFile</a> or String</td><td>Yes</td><td>File to send.</td></tr>
<tr><td>caption</td><td>String</td><td>Optional</td><td>Document caption.</td></tr>
</tbody>
</table>

<h4><a class="anchor" href="#getupdates" name="getupdates"></a>getUpdates</h4>
<p>Use this method to receive incoming updates using long polling. Returns an Array of <a href="#update">Update</a> objects.</p>
<table class="table">
<thead><tr><th>Parameter</th><th>Type</th><th>Required</th><th>Description</th></tr></thead>
<tbody>
<tr><td>limit</td><td>Integer</td><td>Optional</td><td>Limits the number of updates to be retrieved.</td></tr>
</tbody>
</table>

</div></body></html>
```

This fixture exercises:
- A type with mixed required/optional fields (User)
- A union type with no field table, prose listing variants (ChatMember)
- A method with no params returning a single object (getMe)
- A method with params returning an object (sendMessage), including a "Integer or String" union ChatID
- A method triggering HasFiles via "InputFile" mention (sendDocument)
- A method returning "Array of X" (getUpdates)
- The "Recent changes" section with version (`Bot API 7.10`)

- [ ] **Step 2: Write `scripts/snapshot.sh`**

```bash
#!/usr/bin/env bash
# Capture the live Telegram Bot API HTML to testdata/html/snapshot_<date>.html
# and update the latest.html symlink. Used by `make snapshot`.
set -euo pipefail

DATE=$(date +%Y-%m-%d)
DEST="testdata/html/snapshot_${DATE}.html"

curl -fsSL --user-agent "go-telegram codegen scraper" \
  https://core.telegram.org/bots/api > "$DEST"

ln -sf "snapshot_${DATE}.html" "testdata/html/latest.html"
echo "captured: $DEST"
```

```bash
chmod +x scripts/snapshot.sh
```

- [ ] **Step 3: Capture today's snapshot**

```bash
./scripts/snapshot.sh
```

This downloads `core.telegram.org/bots/api` to `testdata/html/snapshot_2026-05-08.html` and creates a `testdata/html/latest.html` symlink. The file will be ~1 MB; commit it.

- [ ] **Step 4: Verify both files exist**

```bash
ls -la testdata/html/
# Expect: snapshot_2026-05-08.html  small_fixture.html  latest.html -> snapshot_2026-05-08.html
```

- [ ] **Step 5: Commit**

```bash
git add scripts/ testdata/html/
git commit -m "test(codegen): add HTML fixture and snapshot capture script"
```

---

## Task 3 — Scraper section walker

**Files:**
- Create: `cmd/scrape/walker.go`
- Create: `cmd/scrape/walker_test.go`

The walker traverses the parsed HTML and produces a list of "sections", each anchored on an `<h4>` element. Lower tasks (T4, T5) consume sections.

- [ ] **Step 1: Implement `cmd/scrape/walker.go`**

```go
package main

import (
	"strings"

	"golang.org/x/net/html"
)

// section is an h4-anchored block of the docs page. Title is the
// heading text (e.g. "User" or "sendMessage"). Description is the
// concatenation of immediately-following <p> paragraphs (until the
// next h4 / h3 / table / list). Tables and Lists hold raw nodes for
// later parsing by the table/oneof extractors.
type section struct {
	Title       string
	Description string
	Tables      []*html.Node // <table> nodes
	Lists       []*html.Node // <ul> nodes (used for oneof variant lists)
}

// walk parses the page and returns sections in document order.
// Sections whose title contains a space (e.g. "Bot API 7.10") are
// included; later passes ignore them or treat them specially.
func walk(doc *html.Node) []section {
	var (
		sections []section
		current  *section
	)

	var visit func(n *html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "h4":
				if current != nil {
					sections = append(sections, *current)
				}
				current = &section{Title: textOf(n)}
				// Don't recurse into the heading; we already have its text.
				return
			case "h3":
				// h3 (e.g. "Available methods") delimits a section;
				// flush the current h4 section but do not start a new one.
				if current != nil {
					sections = append(sections, *current)
					current = nil
				}
				return
			case "p":
				if current != nil {
					if current.Description != "" {
						current.Description += "\n"
					}
					current.Description += strings.TrimSpace(textOf(n))
				}
				return
			case "table":
				if current != nil {
					current.Tables = append(current.Tables, n)
				}
				return
			case "ul":
				if current != nil {
					current.Lists = append(current.Lists, n)
				}
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(doc)
	if current != nil {
		sections = append(sections, *current)
	}
	return sections
}

// textOf returns the concatenated text content of n and descendants,
// with adjacent whitespace collapsed to single spaces.
func textOf(n *html.Node) string {
	var sb strings.Builder
	var w func(*html.Node)
	w = func(n *html.Node) {
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			w(c)
		}
	}
	w(n)
	return collapseWS(sb.String())
}

func collapseWS(s string) string {
	var b strings.Builder
	prevSpace := false
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if !prevSpace {
				b.WriteByte(' ')
			}
			prevSpace = true
			continue
		}
		prevSpace = false
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

// isMethodTitle returns true for headings that look like method names
// (camelCase starting with a lowercase letter; e.g. "sendMessage").
func isMethodTitle(s string) bool {
	if s == "" {
		return false
	}
	r := s[0]
	return r >= 'a' && r <= 'z'
}

// isTypeTitle returns true for headings that look like type names
// (PascalCase; e.g. "Message"). Allows a leading-uppercase only;
// excludes spaces (which would denote a header like "Bot API 7.10").
func isTypeTitle(s string) bool {
	if s == "" {
		return false
	}
	r := s[0]
	if r < 'A' || r > 'Z' {
		return false
	}
	return !strings.Contains(s, " ")
}
```

- [ ] **Step 2: Implement `cmd/scrape/walker_test.go`**

```go
package main

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

func parse(t *testing.T, path string) *html.Node {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()
	doc, err := html.Parse(f)
	require.NoError(t, err)
	return doc
}

func TestWalk_FixtureSections(t *testing.T) {
	doc := parse(t, "../../testdata/html/small_fixture.html")
	sections := walk(doc)

	titles := make([]string, 0, len(sections))
	for _, s := range sections {
		titles = append(titles, s.Title)
	}

	require.Contains(t, titles, "User")
	require.Contains(t, titles, "ChatMember")
	require.Contains(t, titles, "getMe")
	require.Contains(t, titles, "sendMessage")
	require.Contains(t, titles, "sendDocument")
	require.Contains(t, titles, "getUpdates")
	require.Contains(t, titles, "Bot API 7.10")
}

func TestIsMethodTitle(t *testing.T) {
	require.True(t, isMethodTitle("sendMessage"))
	require.True(t, isMethodTitle("getMe"))
	require.False(t, isMethodTitle("Message"))
	require.False(t, isMethodTitle(""))
	require.False(t, isMethodTitle("Bot API 7.10"))
}

func TestIsTypeTitle(t *testing.T) {
	require.True(t, isTypeTitle("Message"))
	require.True(t, isTypeTitle("ChatMember"))
	require.False(t, isTypeTitle("sendMessage"))
	require.False(t, isTypeTitle("Bot API 7.10"))
	require.False(t, isTypeTitle(""))
}

func TestSection_DescriptionAndTables(t *testing.T) {
	doc := parse(t, "../../testdata/html/small_fixture.html")
	sections := walk(doc)
	var sm *section
	for i, s := range sections {
		if s.Title == "sendMessage" {
			sm = &sections[i]
			break
		}
	}
	require.NotNil(t, sm)
	require.True(t, strings.Contains(sm.Description, "Use this method to send text messages"))
	require.Len(t, sm.Tables, 1)
}
```

- [ ] **Step 3: Run**

```bash
go test ./cmd/scrape/...
```
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add cmd/scrape/walker.go cmd/scrape/walker_test.go
git commit -m "feat(scrape): h4-anchored section walker"
```

---

## Task 4 — Table parser (types + methods)

**Files:**
- Create: `cmd/scrape/table.go`
- Create: `cmd/scrape/table_test.go`

This task parses HTML tables into `[]spec.Field`. Type tables have 3 columns (Field/Type/Description); method tables have 4 columns (Parameter/Type/Required/Description). The parser also decodes the type cell into a `spec.TypeRef` (handling primitives, named, arrays, and "X or Y" unions).

- [ ] **Step 1: Implement `cmd/scrape/table.go`**

```go
package main

import (
	"strings"

	"golang.org/x/net/html"

	"github.com/lukaszraczylo/go-telegram/internal/spec"
)

// parseFieldsTable walks a <table> for an object-type definition.
// Columns: Field, Type, Description (optional column orders are not
// supported; Telegram's docs use a stable layout).
//
// Optional fields are detected via the "Optional." prefix in the
// description text, which is the documented convention.
func parseFieldsTable(t *html.Node) []spec.Field {
	rows := tableRows(t)
	if len(rows) == 0 {
		return nil
	}
	var fields []spec.Field
	for _, row := range rows[1:] { // skip header
		cells := rowCells(row)
		if len(cells) < 3 {
			continue
		}
		jname := strings.TrimSpace(textOf(cells[0]))
		typeText := strings.TrimSpace(textOf(cells[1]))
		desc := strings.TrimSpace(textOf(cells[2]))

		required := !strings.HasPrefix(desc, "Optional.")
		fields = append(fields, spec.Field{
			Name:     goName(jname),
			JSONName: jname,
			Type:     parseTypeRef(typeText),
			Required: required,
			Doc:      desc,
		})
	}
	return fields
}

// parseParamsTable walks a <table> for a method definition.
// Columns: Parameter, Type, Required, Description.
func parseParamsTable(t *html.Node) []spec.Field {
	rows := tableRows(t)
	if len(rows) == 0 {
		return nil
	}
	var params []spec.Field
	for _, row := range rows[1:] {
		cells := rowCells(row)
		if len(cells) < 4 {
			continue
		}
		jname := strings.TrimSpace(textOf(cells[0]))
		typeText := strings.TrimSpace(textOf(cells[1]))
		req := strings.EqualFold(strings.TrimSpace(textOf(cells[2])), "Yes")
		desc := strings.TrimSpace(textOf(cells[3]))

		params = append(params, spec.Field{
			Name:     goName(jname),
			JSONName: jname,
			Type:     parseTypeRef(typeText),
			Required: req,
			Doc:      desc,
		})
	}
	return params
}

// tableRows returns the <tr> children of a <table>, skipping over
// any <thead>/<tbody> wrappers.
func tableRows(t *html.Node) []*html.Node {
	var rows []*html.Node
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			rows = append(rows, n)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(t)
	return rows
}

// rowCells returns the <td> (or <th>) children of a <tr>.
func rowCells(tr *html.Node) []*html.Node {
	var cells []*html.Node
	for c := tr.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && (c.Data == "td" || c.Data == "th") {
			cells = append(cells, c)
		}
	}
	return cells
}

// goName converts a snake_case JSON identifier to PascalCase.
// Special-cases common acronyms used in the Telegram docs.
func goName(s string) string {
	if s == "" {
		return ""
	}
	parts := strings.Split(s, "_")
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		switch p {
		case "id":
			b.WriteString("ID")
		case "url":
			b.WriteString("URL")
		case "ip":
			b.WriteString("IP")
		case "https":
			b.WriteString("HTTPS")
		case "json":
			b.WriteString("JSON")
		case "html":
			b.WriteString("HTML")
		default:
			b.WriteByte(p[0] - 'a' + 'A')
			b.WriteString(p[1:])
		}
	}
	return b.String()
}

// parseTypeRef decodes the type-cell text into a spec.TypeRef.
//
// Recognised shapes:
//   "Integer"               → primitive int64
//   "String"                → primitive string
//   "Boolean" / "True"      → primitive bool
//   "Float" / "Float number"→ primitive float64
//   "Array of X"            → array of (parseTypeRef of X)
//   "Array of Array of X"   → array of array of X
//   "Foo"                   → named Foo
//   "Foo or Bar"            → oneOf {Foo, Bar}
//   "InputFile or String"   → oneOf (caller may translate to InputFile)
func parseTypeRef(s string) spec.TypeRef {
	s = strings.TrimSpace(s)
	// Array prefix.
	if rest, ok := strings.CutPrefix(s, "Array of "); ok {
		elem := parseTypeRef(rest)
		return spec.TypeRef{Kind: spec.KindArray, ElemType: &elem}
	}
	// Union ("X or Y" / "X or Y or Z").
	if strings.Contains(s, " or ") {
		parts := strings.Split(s, " or ")
		variants := make([]string, 0, len(parts))
		for _, p := range parts {
			variants = append(variants, primitiveOrNamed(strings.TrimSpace(p)).Name)
		}
		return spec.TypeRef{Kind: spec.KindOneOf, Variants: variants}
	}
	return primitiveOrNamed(s)
}

// primitiveOrNamed maps a single-word type cell to either a primitive
// or a named TypeRef. Unrecognised words are treated as named types.
func primitiveOrNamed(s string) spec.TypeRef {
	switch s {
	case "Integer", "Int":
		return spec.TypeRef{Kind: spec.KindPrimitive, Name: "int64"}
	case "String":
		return spec.TypeRef{Kind: spec.KindPrimitive, Name: "string"}
	case "Boolean", "Bool", "True", "False":
		return spec.TypeRef{Kind: spec.KindPrimitive, Name: "bool"}
	case "Float", "Float number":
		return spec.TypeRef{Kind: spec.KindPrimitive, Name: "float64"}
	default:
		return spec.TypeRef{Kind: spec.KindNamed, Name: s}
	}
}
```

- [ ] **Step 2: Implement `cmd/scrape/table_test.go`**

```go
package main

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lukaszraczylo/go-telegram/internal/spec"
)

func TestGoName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"chat_id", "ChatID"},
		{"first_name", "FirstName"},
		{"is_bot", "IsBot"},
		{"url", "URL"},
		{"ip_address", "IPAddress"},
		{"language_code", "LanguageCode"},
	}
	for _, c := range cases {
		require.Equal(t, c.want, goName(c.in), c.in)
	}
}

func TestParseTypeRef(t *testing.T) {
	cases := []struct {
		in   string
		want spec.TypeRef
	}{
		{"Integer", spec.TypeRef{Kind: spec.KindPrimitive, Name: "int64"}},
		{"String", spec.TypeRef{Kind: spec.KindPrimitive, Name: "string"}},
		{"Boolean", spec.TypeRef{Kind: spec.KindPrimitive, Name: "bool"}},
		{"Float", spec.TypeRef{Kind: spec.KindPrimitive, Name: "float64"}},
		{"Message", spec.TypeRef{Kind: spec.KindNamed, Name: "Message"}},
		{"Array of Update", spec.TypeRef{Kind: spec.KindArray, ElemType: &spec.TypeRef{Kind: spec.KindNamed, Name: "Update"}}},
		{"Array of Array of PhotoSize", spec.TypeRef{Kind: spec.KindArray, ElemType: &spec.TypeRef{Kind: spec.KindArray, ElemType: &spec.TypeRef{Kind: spec.KindNamed, Name: "PhotoSize"}}}},
		{"Integer or String", spec.TypeRef{Kind: spec.KindOneOf, Variants: []string{"int64", "string"}}},
		{"InputFile or String", spec.TypeRef{Kind: spec.KindOneOf, Variants: []string{"InputFile", "string"}}},
	}
	for _, c := range cases {
		require.Equal(t, c.want, parseTypeRef(c.in), c.in)
	}
}

func TestParseFieldsTable_FromFixture(t *testing.T) {
	doc := parse(t, "../../testdata/html/small_fixture.html")
	sections := walk(doc)
	var user *section
	for i := range sections {
		if sections[i].Title == "User" {
			user = &sections[i]
			break
		}
	}
	require.NotNil(t, user)
	require.Len(t, user.Tables, 1)

	fields := parseFieldsTable(user.Tables[0])
	require.Len(t, fields, 4)
	require.Equal(t, "ID", fields[0].Name)
	require.Equal(t, "id", fields[0].JSONName)
	require.Equal(t, spec.KindPrimitive, fields[0].Type.Kind)
	require.True(t, fields[0].Required)

	require.Equal(t, "LastName", fields[3].Name)
	require.False(t, fields[3].Required) // "Optional." prefix
}

func TestParseParamsTable_FromFixture(t *testing.T) {
	doc := parse(t, "../../testdata/html/small_fixture.html")
	sections := walk(doc)
	var sm *section
	for i := range sections {
		if sections[i].Title == "sendMessage" {
			sm = &sections[i]
			break
		}
	}
	require.NotNil(t, sm)
	require.Len(t, sm.Tables, 1)

	params := parseParamsTable(sm.Tables[0])
	require.Len(t, params, 3)
	require.Equal(t, "ChatID", params[0].Name)
	require.True(t, params[0].Required)
	require.Equal(t, spec.KindOneOf, params[0].Type.Kind)
	require.Equal(t, []string{"int64", "string"}, params[0].Type.Variants)

	require.Equal(t, "ParseMode", params[2].Name)
	require.False(t, params[2].Required) // "Optional"
}
```

- [ ] **Step 3: Run + commit**

```bash
go test ./cmd/scrape/...
git add cmd/scrape/table.go cmd/scrape/table_test.go
git commit -m "feat(scrape): table parser for types and method params"
```

---

## Task 5 — Return type, version, HasFiles

**Files:**
- Create: `cmd/scrape/method.go`
- Create: `cmd/scrape/method_test.go`

- [ ] **Step 1: Implement `cmd/scrape/method.go`**

```go
package main

import (
	"regexp"
	"strings"

	"github.com/lukaszraczylo/go-telegram/internal/spec"
)

// extractReturn pulls the return type from a method's description prose.
//
// Patterns we handle (in priority order):
//   "Returns the *X*"        → named X
//   "Returns *X*"            → named X
//   "Returns an Array of X"  → array of named X
//   "Returns Array of X"     → array of named X
//   "Returns True"           → primitive bool
//   "On success, the sent X is returned" / "On success, X is returned" → named X
//   fallback: bool
func extractReturn(desc string) spec.TypeRef {
	// Normalise; strip *bold* markers because Telegram uses italics.
	d := strings.ReplaceAll(desc, "*", "")

	patterns := []struct {
		re *regexp.Regexp
		fn func([]string) spec.TypeRef
	}{
		{regexp.MustCompile(`Returns an? Array of ([A-Z][A-Za-z0-9]+)`), func(m []string) spec.TypeRef {
			return spec.TypeRef{Kind: spec.KindArray, ElemType: &spec.TypeRef{Kind: spec.KindNamed, Name: m[1]}}
		}},
		{regexp.MustCompile(`On success(?:,)? (?:an? )?Array of ([A-Z][A-Za-z0-9]+) (?:objects )?(?:is|are) returned`), func(m []string) spec.TypeRef {
			return spec.TypeRef{Kind: spec.KindArray, ElemType: &spec.TypeRef{Kind: spec.KindNamed, Name: m[1]}}
		}},
		{regexp.MustCompile(`Returns (?:the )?(?:newly )?(?:edited |sent |created |updated )?([A-Z][A-Za-z0-9]+)`), func(m []string) spec.TypeRef {
			if m[1] == "True" || m[1] == "False" {
				return spec.TypeRef{Kind: spec.KindPrimitive, Name: "bool"}
			}
			return spec.TypeRef{Kind: spec.KindNamed, Name: m[1]}
		}},
		{regexp.MustCompile(`On success(?:,)? (?:the )?(?:newly )?(?:edited |sent |created |updated )?([A-Z][A-Za-z0-9]+) is returned`), func(m []string) spec.TypeRef {
			return spec.TypeRef{Kind: spec.KindNamed, Name: m[1]}
		}},
		{regexp.MustCompile(`Returns True`), func(m []string) spec.TypeRef {
			return spec.TypeRef{Kind: spec.KindPrimitive, Name: "bool"}
		}},
		{regexp.MustCompile(`(?i)on success, true is returned`), func(m []string) spec.TypeRef {
			return spec.TypeRef{Kind: spec.KindPrimitive, Name: "bool"}
		}},
	}
	for _, p := range patterns {
		if m := p.re.FindStringSubmatch(d); m != nil {
			return p.fn(m)
		}
	}
	// Fallback: bool. Better than panic; method-by-method tests would
	// catch any regression.
	return spec.TypeRef{Kind: spec.KindPrimitive, Name: "bool"}
}

// hasFilesParams returns true if any param mentions InputFile (the
// scraper convention triggering multipart/form-data).
func hasFilesParams(params []spec.Field) bool {
	for _, p := range params {
		if mentionsInputFile(p.Type) {
			return true
		}
	}
	return false
}

func mentionsInputFile(tr spec.TypeRef) bool {
	switch tr.Kind {
	case spec.KindNamed:
		return tr.Name == "InputFile"
	case spec.KindArray:
		if tr.ElemType != nil {
			return mentionsInputFile(*tr.ElemType)
		}
	case spec.KindOneOf:
		for _, v := range tr.Variants {
			if v == "InputFile" {
				return true
			}
		}
	}
	return false
}

// extractVersion finds the API version string in a "Bot API X.Y" heading.
var versionRE = regexp.MustCompile(`Bot API (\d+\.\d+)`)

func extractVersion(sections []section) string {
	for _, s := range sections {
		if m := versionRE.FindStringSubmatch(s.Title); m != nil {
			return m[1]
		}
	}
	return ""
}
```

- [ ] **Step 2: Implement `cmd/scrape/method_test.go`**

```go
package main

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lukaszraczylo/go-telegram/internal/spec"
)

func TestExtractReturn(t *testing.T) {
	cases := []struct {
		in   string
		want spec.TypeRef
	}{
		{"Returns basic information about the bot in form of a User object.", spec.TypeRef{Kind: spec.KindNamed, Name: "User"}},
		{"On success, the sent Message is returned.", spec.TypeRef{Kind: spec.KindNamed, Name: "Message"}},
		{"Returns an Array of Update objects.", spec.TypeRef{Kind: spec.KindArray, ElemType: &spec.TypeRef{Kind: spec.KindNamed, Name: "Update"}}},
		{"Returns True on success.", spec.TypeRef{Kind: spec.KindPrimitive, Name: "bool"}},
		{"On success, True is returned.", spec.TypeRef{Kind: spec.KindPrimitive, Name: "bool"}},
	}
	for _, c := range cases {
		require.Equal(t, c.want, extractReturn(c.in), c.in)
	}
}

func TestHasFilesParams(t *testing.T) {
	require.True(t, hasFilesParams([]spec.Field{
		{Type: spec.TypeRef{Kind: spec.KindNamed, Name: "InputFile"}},
	}))
	require.True(t, hasFilesParams([]spec.Field{
		{Type: spec.TypeRef{Kind: spec.KindOneOf, Variants: []string{"InputFile", "string"}}},
	}))
	require.False(t, hasFilesParams([]spec.Field{
		{Type: spec.TypeRef{Kind: spec.KindPrimitive, Name: "string"}},
	}))
}

func TestExtractVersion(t *testing.T) {
	sections := []section{{Title: "Recent changes"}, {Title: "Bot API 7.10"}, {Title: "Available types"}}
	require.Equal(t, "7.10", extractVersion(sections))
}
```

- [ ] **Step 3: Run + commit**

```bash
go test ./cmd/scrape/...
git add cmd/scrape/method.go cmd/scrape/method_test.go
git commit -m "feat(scrape): return-type, HasFiles, and version extraction"
```

---

## Task 6 — Wire scraper, stable JSON output, golden test

**Files:**
- Modify: `cmd/scrape/main.go` (replace `scrape` and `writeJSON` stubs with implementations)
- Create: `cmd/scrape/scrape.go` (top-level scrape function)
- Create: `cmd/scrape/scrape_test.go` (golden test against fixture)
- Create: `testdata/golden/api_small_fixture.json` (committed)

- [ ] **Step 1: Implement `cmd/scrape/scrape.go`**

```go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"golang.org/x/net/html"

	"github.com/lukaszraczylo/go-telegram/internal/spec"
)

// scrape (the package-level implementation overriding the stub in main.go;
// remove the stub from main.go in this task) parses the docs HTML into IR.
func scrape(htmlBytes []byte) (*spec.API, error) {
	doc, err := html.Parse(bytes.NewReader(htmlBytes))
	if err != nil {
		return nil, fmt.Errorf("html parse: %w", err)
	}
	sections := walk(doc)
	api := &spec.API{Version: extractVersion(sections)}
	for _, s := range sections {
		switch {
		case isMethodTitle(s.Title):
			api.Methods = append(api.Methods, methodFromSection(s))
		case isTypeTitle(s.Title):
			api.Types = append(api.Types, typeFromSection(s))
		}
	}
	return api, nil
}

func typeFromSection(s section) spec.TypeDecl {
	td := spec.TypeDecl{Name: s.Title, Doc: s.Description}
	if len(s.Tables) > 0 {
		td.Fields = parseFieldsTable(s.Tables[0])
	} else if len(s.Lists) > 0 {
		// Union: extract variant names from <li><a>...</a></li>.
		td.OneOf = extractListLinks(s.Lists[0])
	}
	return td
}

func methodFromSection(s section) spec.MethodDecl {
	md := spec.MethodDecl{Name: s.Title, Doc: s.Description, Returns: extractReturn(s.Description)}
	if len(s.Tables) > 0 {
		md.Params = parseParamsTable(s.Tables[0])
	}
	md.HasFiles = hasFilesParams(md.Params)
	return md
}

// extractListLinks pulls anchor texts out of a <ul>: each <li><a>X</a></li>
// contributes "X" to the result. Used for union variant lists.
func extractListLinks(ul *html.Node) []string {
	var names []string
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			names = append(names, textOf(n))
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(ul)
	return names
}

// writeJSON marshals the IR with stable, human-readable formatting and
// writes it to path. Marshalling is deterministic: types and methods
// preserve scrape order; struct fields use IR-defined order.
func writeJSON(path string, api *spec.API) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(api); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}
```

- [ ] **Step 2: Remove `scrape` and `writeJSON` stubs from `cmd/scrape/main.go`**

In `cmd/scrape/main.go`, delete the two stub functions at the bottom (the ones returning `errors.New("... not implemented")`). The implementations now live in `scrape.go`.

- [ ] **Step 3: Implement `cmd/scrape/scrape_test.go`**

```go
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "update golden files")

func TestScrape_Golden_SmallFixture(t *testing.T) {
	htmlBytes, err := os.ReadFile("../../testdata/html/small_fixture.html")
	require.NoError(t, err)

	api, err := scrape(htmlBytes)
	require.NoError(t, err)

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	require.NoError(t, enc.Encode(api))

	goldenPath := "../../testdata/golden/api_small_fixture.json"
	if *update {
		require.NoError(t, os.WriteFile(goldenPath, buf.Bytes(), 0o644))
		return
	}
	expected, err := os.ReadFile(goldenPath)
	require.NoError(t, err, "missing golden; run `go test -update ./cmd/scrape/...` to create")
	require.Equal(t, string(expected), buf.String())
}
```

- [ ] **Step 4: Generate the golden file**

```bash
go test -update ./cmd/scrape/...
```

Inspect `testdata/golden/api_small_fixture.json`. It should reflect the fixture: 2 types (User with 4 fields, ChatMember with OneOf), 4 methods (getMe, sendMessage, sendDocument with HasFiles=true, getUpdates with array return), version "7.10".

- [ ] **Step 5: Run again without -update; must pass**

```bash
go test ./cmd/scrape/...
```

- [ ] **Step 6: Commit**

```bash
git add cmd/scrape/scrape.go cmd/scrape/scrape_test.go cmd/scrape/main.go testdata/golden/api_small_fixture.json
git commit -m "feat(scrape): wire scraper end-to-end with golden test"
```

---

## Task 7 — Capture full snapshot, generate api.json

**Files:**
- Create: `internal/spec/api.json` (committed; the IR for the snapshot)

- [ ] **Step 1: Run the scraper against the snapshot**

```bash
go run ./cmd/scrape -input testdata/html/snapshot_2026-05-08.html -output internal/spec/api.json
```

This should produce a ~50–200 KB JSON file with all current Telegram API types and methods.

- [ ] **Step 2: Sanity check**

```bash
# Roughly how many entities?
go run -tags ignore_autogenerated ./cmd/scrape -input testdata/html/snapshot_2026-05-08.html -output /tmp/api.json
python3 -c 'import json; d=json.load(open("internal/spec/api.json")); print("version:", d["version"]); print("types:", len(d.get("types",[]))); print("methods:", len(d.get("methods",[])))'
```

Should report `version: <something like 7.10>`, `types: ~140`, `methods: ~110`. If any counts are wildly wrong (e.g. 0 methods), STOP and investigate the scraper before committing.

- [ ] **Step 3: Spot-check critical entries**

```bash
python3 -c 'import json; d=json.load(open("internal/spec/api.json")); names=[m["name"] for m in d["methods"]]; assert "getMe" in names; assert "sendMessage" in names; assert "sendDocument" in names; assert "getUpdates" in names; print("ok")'
```

- [ ] **Step 4: Commit**

```bash
git add internal/spec/api.json
git commit -m "chore(spec): regenerate api.json from snapshot 2026-05-08"
```

---

## Task 8 — Emitter: types template + driver

**Files:**
- Create: `cmd/genapi/emitter.go`
- Create: `cmd/genapi/types.tmpl`
- Create: `cmd/genapi/emitter_test.go`
- Create: `testdata/golden/types.gen.go`
- Modify: `cmd/genapi/main.go` (replace `run` stub)

The emitter loads the IR, runs templates, formats output, and writes files. We start with `types.gen.go` only; methods/enums/oneOf land in T9/T10.

- [ ] **Step 1: Implement `cmd/genapi/emitter.go`**

```go
package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"text/template"

	"github.com/lukaszraczylo/go-telegram/internal/spec"
)

//go:embed types.tmpl
var typesTmpl string

// emitter renders Go source from a spec.API IR.
type emitter struct {
	api    *spec.API
	outDir string
}

func newEmitter(api *spec.API, outDir string) *emitter {
	return &emitter{api: api, outDir: outDir}
}

// emitTypes renders types.gen.go.
func (e *emitter) emitTypes() error {
	t, err := template.New("types").Funcs(funcs()).Parse(typesTmpl)
	if err != nil {
		return fmt.Errorf("parse types.tmpl: %w", err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, e.api); err != nil {
		return fmt.Errorf("execute types.tmpl: %w", err)
	}
	src, err := format.Source(buf.Bytes())
	if err != nil {
		// Surface the unformatted output so debugging is possible.
		return fmt.Errorf("gofmt types.gen.go: %w\n--- unformatted ---\n%s", err, buf.String())
	}
	return os.WriteFile(filepath.Join(e.outDir, "types.gen.go"), src, 0o644)
}

// loadAPI reads and decodes the IR JSON.
func loadAPI(path string) (*spec.API, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var api spec.API
	if err := json.Unmarshal(data, &api); err != nil {
		return nil, err
	}
	return &api, nil
}

// funcs is the FuncMap shared across templates.
func funcs() template.FuncMap {
	return template.FuncMap{
		"goType":     goType,
		"goField":    goField,
		"docComment": docComment,
		"isOptional": func(f spec.Field) bool { return !f.Required },
	}
}

// goType returns the Go type expression for a TypeRef.
// Optional fields use pointer types for primitives and named types,
// or rely on omitempty for slices and maps. parameter `optional` controls
// whether to wrap pointer-style.
func goType(tr spec.TypeRef, optional bool) string {
	switch tr.Kind {
	case spec.KindPrimitive:
		if optional && (tr.Name == "bool" || tr.Name == "int64" || tr.Name == "float64") {
			return "*" + tr.Name
		}
		return tr.Name
	case spec.KindNamed:
		// Named types are always pointer-optional when optional, except
		// for unions (which are interface types, naturally nil-able).
		if optional {
			return "*" + tr.Name
		}
		return tr.Name
	case spec.KindArray:
		if tr.ElemType == nil {
			return "[]any"
		}
		// Inside slices, the element shape is its own thing — never wrap
		// the element in a pointer just because the field is optional.
		return "[]" + goType(*tr.ElemType, false)
	case spec.KindOneOf:
		// Unions render as interface types named "<concat of variants>";
		// for unsupported corner cases, fall back to interface{}.
		return "any"
	}
	return "any"
}

// goField returns the Go struct-field declaration for a Field.
func goField(f spec.Field) string {
	tag := fmt.Sprintf("`json:%q`", f.JSONName+omitempty(f))
	return fmt.Sprintf("%s %s %s", f.Name, goType(f.Type, !f.Required), tag)
}

func omitempty(f spec.Field) string {
	if f.Required {
		return ""
	}
	return ",omitempty"
}

// docComment converts a doc string into a Go-style block comment with
// a leading "// " on each line.
func docComment(s string) string {
	if s == "" {
		return ""
	}
	var buf bytes.Buffer
	for _, line := range splitLines(s) {
		buf.WriteString("// ")
		buf.WriteString(line)
		buf.WriteByte('\n')
	}
	return buf.String()
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}
```

- [ ] **Step 2: Implement `cmd/genapi/types.tmpl`**

```text
// Code generated by cmd/genapi. DO NOT EDIT.

//go:build !ignore_autogenerated

// Package api contains the Telegram Bot API object types and method
// wrappers, generated from the live documentation by cmd/genapi.
package api

import "io"

var _ = io.Discard // keep import even if no fields use io

{{range .Types}}
{{if .OneOf}}
// {{.Name}} is a union type. The following concrete variants implement
// it:
{{range .OneOf}}//   - {{.}}
{{end}}//
{{docComment .Doc -}}
type {{.Name}} interface{ is{{.Name}}() }
{{else}}
{{docComment .Doc -}}
type {{.Name}} struct {
{{range .Fields}}{{docComment .Doc}}{{goField .}}
{{end}}}
{{end}}
{{end}}
```

- [ ] **Step 3: Replace `run` in `cmd/genapi/main.go`**

```go
func run(input, outdir string) error {
	api, err := loadAPI(input)
	if err != nil {
		return fmt.Errorf("load api: %w", err)
	}
	if err := os.MkdirAll(outdir, 0o755); err != nil {
		return err
	}
	e := newEmitter(api, outdir)
	if err := e.emitTypes(); err != nil {
		return err
	}
	// methods/enums/oneof emitters land in P2.T9/T10; for now types only.
	return nil
}
```

Add `"os"` to the imports if not already present, and `"fmt"`.

- [ ] **Step 4: Implement `cmd/genapi/emitter_test.go`**

```go
package main

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

var updateGolden = flag.Bool("update", false, "update golden files")

func TestEmit_Types_FixtureGolden(t *testing.T) {
	api, err := loadAPI("../../testdata/golden/api_small_fixture.json")
	require.NoError(t, err)

	tmp := t.TempDir()
	e := newEmitter(api, tmp)
	require.NoError(t, e.emitTypes())

	got, err := os.ReadFile(filepath.Join(tmp, "types.gen.go"))
	require.NoError(t, err)

	goldenPath := "../../testdata/golden/types.gen.go"
	if *updateGolden {
		require.NoError(t, os.WriteFile(goldenPath, got, 0o644))
		return
	}
	expected, err := os.ReadFile(goldenPath)
	require.NoError(t, err, "missing golden; run `go test -update ./cmd/genapi/...`")
	require.Equal(t, string(expected), string(got))
}
```

- [ ] **Step 5: Generate golden + verify**

```bash
go test -update ./cmd/genapi/...
go test ./cmd/genapi/...
```

Inspect `testdata/golden/types.gen.go`. It should be valid Go with `User` struct (4 fields) and `ChatMember` interface. Confirm `gofmt -l testdata/golden/types.gen.go` produces no output (file is already gofmt-clean).

- [ ] **Step 6: Commit**

```bash
git add cmd/genapi/ testdata/golden/types.gen.go
git commit -m "feat(genapi): types template + emitter driver with golden test"
```

---

## Task 9 — Emitter: methods + multipart

**Files:**
- Create: `cmd/genapi/methods.tmpl`
- Modify: `cmd/genapi/emitter.go` (add `emitMethods`)
- Modify: `cmd/genapi/main.go` (call emitMethods)
- Create: `testdata/golden/methods.gen.go` (golden)
- Update: `cmd/genapi/emitter_test.go` (add methods test)

- [ ] **Step 1: `cmd/genapi/methods.tmpl`**

```text
// Code generated by cmd/genapi. DO NOT EDIT.

//go:build !ignore_autogenerated

package api

import (
	"context"
	"strconv"

	"github.com/lukaszraczylo/go-telegram/client"
)

var _ = strconv.Itoa // keep import for multipart helpers

{{range .Methods}}
// {{.Name}}Params is the parameter set for {{.Name}}.
//
{{docComment .Doc -}}
type {{title .Name}}Params struct {
{{range .Params}}{{docComment .Doc}}{{goField .}}
{{end}}}
{{if .HasFiles}}
// HasFile reports whether a multipart upload is required.
func (p *{{title .Name}}Params) HasFile() bool {
{{range .Params}}{{if isFileField .}}	if p.{{.Name}} != nil && p.{{.Name}}.IsLocalUpload() { return true }
{{end}}{{end}}	return false
}

// MultipartFields returns the non-file fields used in the multipart body.
func (p *{{title .Name}}Params) MultipartFields() map[string]string {
	out := map[string]string{}
{{range .Params}}{{if not (isFileField .)}}{{multipartFieldEntry .}}{{end}}{{end}}	return out
}

// MultipartFiles returns the file parts.
func (p *{{title .Name}}Params) MultipartFiles() []client.MultipartFile {
	var files []client.MultipartFile
{{range .Params}}{{if isFileField .}}	if p.{{.Name}} != nil && p.{{.Name}}.IsLocalUpload() {
		name := p.{{.Name}}.Filename
		if name == "" { name = "{{.JSONName}}" }
		files = append(files, client.MultipartFile{FieldName: "{{.JSONName}}", Filename: name, Reader: p.{{.Name}}.Reader})
	}
{{end}}{{end}}	return files
}
{{end}}

{{docComment .Doc -}}
func {{title .Name}}(ctx context.Context, b *client.Bot, p *{{title .Name}}Params) ({{returnGoType .Returns}}, error) {
	return client.Call[*{{title .Name}}Params, {{returnGoType .Returns}}](ctx, b, "{{.Name}}", p)
}
{{end}}
```

- [ ] **Step 2: Extend `cmd/genapi/emitter.go`**

Add (alongside `typesTmpl`):

```go
//go:embed methods.tmpl
var methodsTmpl string
```

Add to the FuncMap:

```go
"title":              title,
"isFileField":        isFileField,
"multipartFieldEntry": multipartFieldEntry,
"returnGoType":       returnGoType,
```

Add helper funcs:

```go
import "strings"

func title(s string) string {
	if s == "" {
		return ""
	}
	r := s[0]
	if r >= 'a' && r <= 'z' {
		r = r - 'a' + 'A'
	}
	return string(r) + s[1:]
}

func isFileField(f spec.Field) bool {
	return mentionsInputFileTr(f.Type)
}

func mentionsInputFileTr(tr spec.TypeRef) bool {
	switch tr.Kind {
	case spec.KindNamed:
		return tr.Name == "InputFile"
	case spec.KindArray:
		if tr.ElemType != nil {
			return mentionsInputFileTr(*tr.ElemType)
		}
	case spec.KindOneOf:
		for _, v := range tr.Variants {
			if v == "InputFile" {
				return true
			}
		}
	}
	return false
}

// multipartFieldEntry generates the line that adds f to the multipart map.
// Required scalar fields go in unconditionally; optional ones go in only
// when non-zero/non-empty.
func multipartFieldEntry(f spec.Field) string {
	switch f.Type.Kind {
	case spec.KindPrimitive:
		switch f.Type.Name {
		case "int64":
			if f.Required {
				return fmt.Sprintf("\tout[%q] = strconv.FormatInt(p.%s, 10)\n", f.JSONName, f.Name)
			}
			return fmt.Sprintf("\tif p.%s != 0 { out[%q] = strconv.FormatInt(p.%s, 10) }\n", f.Name, f.JSONName, f.Name)
		case "string":
			if f.Required {
				return fmt.Sprintf("\tout[%q] = p.%s\n", f.JSONName, f.Name)
			}
			return fmt.Sprintf("\tif p.%s != \"\" { out[%q] = p.%s }\n", f.Name, f.JSONName, f.Name)
		case "bool":
			if f.Required {
				return fmt.Sprintf("\tif p.%s { out[%q] = \"true\" } else { out[%q] = \"false\" }\n", f.Name, f.JSONName, f.JSONName)
			}
			return fmt.Sprintf("\tif p.%s != nil { if *p.%s { out[%q] = \"true\" } else { out[%q] = \"false\" } }\n", f.Name, f.Name, f.JSONName, f.JSONName)
		case "float64":
			if f.Required {
				return fmt.Sprintf("\tout[%q] = strconv.FormatFloat(p.%s, 'f', -1, 64)\n", f.JSONName, f.Name)
			}
			return fmt.Sprintf("\tif p.%s != nil { out[%q] = strconv.FormatFloat(*p.%s, 'f', -1, 64) }\n", f.Name, f.JSONName, f.Name)
		}
	case spec.KindOneOf:
		// For unions like "InputFile or String" the non-file branch is
		// stringified by the user; we send empty rather than panic if
		// the field is unset. For complex unions, fall back to JSON-marshal.
		if f.Required {
			return fmt.Sprintf("\tif s, ok := p.%s.(string); ok { out[%q] = s }\n", f.Name, f.JSONName)
		}
		return ""
	}
	// Complex types (struct, array, map): marshal to JSON string.
	if f.Required {
		return fmt.Sprintf("\tif b, _ := json.Marshal(p.%s); len(b) > 0 { out[%q] = string(b) }\n", f.Name, f.JSONName)
	}
	return fmt.Sprintf("\tif p.%s != nil { if b, _ := json.Marshal(p.%s); len(b) > 0 { out[%q] = string(b) } }\n", f.Name, f.Name, f.JSONName)
}

func returnGoType(tr spec.TypeRef) string {
	switch tr.Kind {
	case spec.KindPrimitive:
		return tr.Name
	case spec.KindNamed:
		return "*" + tr.Name
	case spec.KindArray:
		if tr.ElemType == nil {
			return "[]any"
		}
		return "[]" + returnGoElem(*tr.ElemType)
	}
	return "any"
}

func returnGoElem(tr spec.TypeRef) string {
	switch tr.Kind {
	case spec.KindPrimitive:
		return tr.Name
	case spec.KindNamed:
		return tr.Name
	case spec.KindArray:
		if tr.ElemType == nil {
			return "any"
		}
		return "[]" + returnGoElem(*tr.ElemType)
	}
	return "any"
}
```

Add an `emitMethods` method on `emitter`:

```go
func (e *emitter) emitMethods() error {
	t, err := template.New("methods").Funcs(funcs()).Parse(methodsTmpl)
	if err != nil {
		return fmt.Errorf("parse methods.tmpl: %w", err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, e.api); err != nil {
		return fmt.Errorf("execute methods.tmpl: %w", err)
	}
	src, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("gofmt methods.gen.go: %w\n--- unformatted ---\n%s", err, buf.String())
	}
	return os.WriteFile(filepath.Join(e.outDir, "methods.gen.go"), src, 0o644)
}
```

Update `run` in `main.go` to call `emitMethods` after `emitTypes`.

If the methods template needs `encoding/json` (for complex multipart fallback), add an `import "encoding/json"` line in the template's imports block:

```text
import (
	"context"
	"encoding/json"
	"strconv"
	...
)
var _ = json.Marshal // keep
```

- [ ] **Step 3: Add to `emitter_test.go`**

```go
func TestEmit_Methods_FixtureGolden(t *testing.T) {
	api, err := loadAPI("../../testdata/golden/api_small_fixture.json")
	require.NoError(t, err)

	tmp := t.TempDir()
	e := newEmitter(api, tmp)
	require.NoError(t, e.emitTypes())   // some methods reference types
	require.NoError(t, e.emitMethods())

	got, err := os.ReadFile(filepath.Join(tmp, "methods.gen.go"))
	require.NoError(t, err)

	goldenPath := "../../testdata/golden/methods.gen.go"
	if *updateGolden {
		require.NoError(t, os.WriteFile(goldenPath, got, 0o644))
		return
	}
	expected, err := os.ReadFile(goldenPath)
	require.NoError(t, err, "missing golden; run `go test -update ./cmd/genapi/...`")
	require.Equal(t, string(expected), string(got))
}
```

- [ ] **Step 4: Generate golden + verify build**

```bash
go test -update ./cmd/genapi/...
go test ./cmd/genapi/...

# Verify the generated methods.gen.go compiles in isolation by writing it
# into a tmpdir alongside types.gen.go and a minimal main:
mkdir -p /tmp/genapi-smoke && cp testdata/golden/{types,methods}.gen.go /tmp/genapi-smoke/
```

(You don't need to actually compile the dump — the next task generates against the full IR, which is the real verification.)

- [ ] **Step 5: Commit**

```bash
git add cmd/genapi/ testdata/golden/methods.gen.go
git commit -m "feat(genapi): methods template with multipart helpers"
```

---

## Task 10 — Emitter: enums + oneof

**Files:**
- Create: `cmd/genapi/enums.tmpl`
- Create: `cmd/genapi/oneof.tmpl`
- Modify: `cmd/genapi/emitter.go` (add `emitEnums`, `emitOneOf`, enum extraction)
- Modify: `cmd/genapi/main.go` (call new emitters)
- Update: golden tests

Enums are extracted heuristically: when a field's doc text says "must be one of: a, b, c" or similar, those values become const declarations. For the v1 surface we only emit a known short list (parse_mode, chat type, message entity type) hard-coded into the emitter to avoid false positives. Plan 3 (future) can expand this.

OneOf types are non-trivial: each variant must be emitted as a concrete struct (which would normally come from the union member's own TypeDecl) and a marker method, plus an `UnmarshalJSON` on a wrapper interface that switches on a discriminator field (typically `type` or `source`).

For Plan 2 we keep oneOf simple: emit the interface only (already done in types.tmpl) and provide a `Raw json.RawMessage` fallback. Concrete variant decoding is left as a known limitation for v1; users can manually re-decode via `json.RawMessage` if they need a specific variant. Plan 3 (deferred) can implement full discriminator switching.

- [ ] **Step 1: `cmd/genapi/enums.tmpl`**

```text
// Code generated by cmd/genapi. DO NOT EDIT.

//go:build !ignore_autogenerated

package api

// ParseMode controls how Telegram interprets formatting in message text.
type ParseMode string

const (
	ParseModeMarkdown   ParseMode = "Markdown"   // legacy
	ParseModeMarkdownV2 ParseMode = "MarkdownV2"
	ParseModeHTML       ParseMode = "HTML"
)

// ChatType is the type of a Telegram chat.
type ChatType string

const (
	ChatTypePrivate    ChatType = "private"
	ChatTypeGroup      ChatType = "group"
	ChatTypeSupergroup ChatType = "supergroup"
	ChatTypeChannel    ChatType = "channel"
)

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

// MessageEntityType is the kind of an entity (mention, hashtag, command, ...).
type MessageEntityType string

const (
	EntityMention     MessageEntityType = "mention"
	EntityHashtag     MessageEntityType = "hashtag"
	EntityCashtag     MessageEntityType = "cashtag"
	EntityBotCommand  MessageEntityType = "bot_command"
	EntityURL         MessageEntityType = "url"
	EntityEmail       MessageEntityType = "email"
	EntityPhoneNumber MessageEntityType = "phone_number"
	EntityBold        MessageEntityType = "bold"
	EntityItalic      MessageEntityType = "italic"
	EntityUnderline   MessageEntityType = "underline"
	EntityStrike      MessageEntityType = "strikethrough"
	EntitySpoiler     MessageEntityType = "spoiler"
	EntityCode        MessageEntityType = "code"
	EntityPre         MessageEntityType = "pre"
	EntityTextLink    MessageEntityType = "text_link"
	EntityTextMention MessageEntityType = "text_mention"
	EntityCustomEmoji MessageEntityType = "custom_emoji"
)
```

This template is static (no `{{...}}` variables) — it's hand-curated content emitted verbatim to ensure stable enum names.

- [ ] **Step 2: `cmd/genapi/oneof.tmpl` — STAYS EMPTY for Plan 2**

Plan 2 keeps `oneof` types as bare interfaces (already emitted by `types.tmpl` when `OneOf` is non-empty). Concrete variant decoding is deferred. Skip this file in Plan 2; do not create it.

- [ ] **Step 3: Add `emitEnums` to `emitter.go`**

```go
//go:embed enums.tmpl
var enumsTmpl string

func (e *emitter) emitEnums() error {
	t, err := template.New("enums").Funcs(funcs()).Parse(enumsTmpl)
	if err != nil {
		return fmt.Errorf("parse enums.tmpl: %w", err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, e.api); err != nil {
		return fmt.Errorf("execute enums.tmpl: %w", err)
	}
	src, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("gofmt enums.gen.go: %w\n--- unformatted ---\n%s", err, buf.String())
	}
	return os.WriteFile(filepath.Join(e.outDir, "enums.gen.go"), src, 0o644)
}
```

Update `run` in `main.go` to call `emitEnums` after the others.

- [ ] **Step 4: Add golden test for enums**

```go
func TestEmit_Enums_FixtureGolden(t *testing.T) {
	api, err := loadAPI("../../testdata/golden/api_small_fixture.json")
	require.NoError(t, err)

	tmp := t.TempDir()
	e := newEmitter(api, tmp)
	require.NoError(t, e.emitEnums())

	got, err := os.ReadFile(filepath.Join(tmp, "enums.gen.go"))
	require.NoError(t, err)

	goldenPath := "../../testdata/golden/enums.gen.go"
	if *updateGolden {
		require.NoError(t, os.WriteFile(goldenPath, got, 0o644))
		return
	}
	expected, err := os.ReadFile(goldenPath)
	require.NoError(t, err)
	require.Equal(t, string(expected), string(got))
}
```

- [ ] **Step 5: Generate golden + commit**

```bash
go test -update ./cmd/genapi/...
go test ./cmd/genapi/...
git add cmd/genapi/ testdata/golden/enums.gen.go
git commit -m "feat(genapi): static enums template (parse_mode, chat type, ...)"
```

---

## Task 11 — Generate full api/ from snapshot

**Files:**
- Modify (replace): `api/types.gen.go`, `api/methods.gen.go`, `api/enums.gen.go` (committed)

This task runs the full pipeline against the committed snapshot and commits the generated code.

- [ ] **Step 1: Run the emitter**

```bash
go run ./cmd/genapi -input internal/spec/api.json -outdir api
```

This writes (or overwrites) `api/types.gen.go`, `api/methods.gen.go`, `api/enums.gen.go`.

- [ ] **Step 2: Confirm the package compiles**

```bash
go build ./api/...
```

If it does NOT compile, the most likely causes are:
- A method has an unsupported return-type pattern (regex didn't match) → fallback to bool will mostly work, but specific failures may surface as "undefined" if the regex picked up a non-existent name. Inspect the error and either fix `extractReturn` or skip the offending method by hand-patching the IR for now.
- A field type cell uses a shape `parseTypeRef` doesn't recognise. Inspect the unformatted output (the emitter prints it on gofmt failure).

If you have to hand-patch a few rows in `internal/spec/api.json` to get past unimplemented edge cases, do so and document it in the commit message; create a known-issues note at the top of `CHANGELOG.md` "Unreleased" section listing the limitations.

- [ ] **Step 3: Run the cross-package tests**

```bash
go test ./...
```

Existing hand-coded tests in `api/methods_test.go` and `api/types_test.go` may now reference types/methods that exist in both hand-coded and generated form, causing duplicate declarations. T12 handles this. For now, only confirm `./client/`, `./transport/`, `./dispatch/`, `./internal/spec/`, `./cmd/...` are green.

```bash
go test ./client/... ./transport/... ./dispatch/... ./internal/spec/... ./cmd/...
```

If `./api/...` fails due to duplicates, that's expected; T12 fixes it.

- [ ] **Step 4: Commit**

```bash
git add api/
git commit -m "feat(api): regenerate from Telegram Bot API snapshot 2026-05-08"
```

---

## Task 12 — Replace hand-coded api/

**Files:**
- Delete: `api/types.go`
- Delete: `api/methods.go`
- Possibly modify: `api/types_test.go` and `api/methods_test.go` to reference only generated types

The hand-coded files now duplicate names with the generated `*.gen.go`. Delete them. Keep the test files as smoke tests against the generated code; adjust if they reference renamed symbols.

- [ ] **Step 1: Inspect for symbol drift**

```bash
diff <(go doc -all ./api 2>&1 | head -200) /dev/null  # just see what's exported
```

Compare hand-coded names to generated names. If the scraper produced different field/struct names for the 8 hand-coded methods, decide:
- If diff is purely cosmetic (doc text), no action needed.
- If a struct field has been renamed (e.g. `ChatID int64` → `ChatID any` because of "Integer or String"), update the existing tests in `api/methods_test.go` that reference these fields.

The expected drift is `SendMessageParams.ChatID` going from `int64` to `any` (oneOf union). Update the test:

```go
// In api/methods_test.go, change:
//   &SendMessageParams{ChatID: 42, Text: "hi"}
// to:
//   &SendMessageParams{ChatID: int64(42), Text: "hi"}
```

(`int64(42)` satisfies `any`.) Same for `examples/echo/main.go`:

```go
// In examples/echo/main.go, change:
//   ChatID: m.Chat.ID
// to:
//   ChatID: m.Chat.ID  // Chat.ID is int64; satisfies any
```

If `m.Chat.ID` is itself `int64`, no test change needed; the assignment to `any` is automatic.

If Test code references `ParseMode` constants or other top-level names that the generator now puts in `enums.gen.go`, no change is needed since they're in the same `api` package.

- [ ] **Step 2: Delete hand-coded files**

```bash
git rm api/types.go api/methods.go
```

- [ ] **Step 3: Run tests**

```bash
go test -race ./...
```

If anything fails, fix the failing test or example file (NOT the generated code). The generated files are the source of truth.

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "refactor(api): replace hand-coded slice with generated code"
```

---

## Task 13 — Shape smoke tests

**Files:**
- Create: `api/shape_test.go`

Per spec §11.2: one test per code-generation pattern, hitting a representative method through a mocked HTTPDoer.

- [ ] **Step 1: Implement `api/shape_test.go`**

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

type shapeMockDoer struct{ mock.Mock }

func (m *shapeMockDoer) Do(r *http.Request) (*http.Response, error) {
	args := m.Called(r)
	if v := args.Get(0); v != nil {
		return v.(*http.Response), args.Error(1)
	}
	return nil, args.Error(1)
}

func shapeJSONResp(body string) *http.Response {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}

func newShapeBot(t *testing.T, m *shapeMockDoer) *client.Bot {
	t.Helper()
	return client.New("t", client.WithHTTPClient(m))
}

// TestShape_Simple: getMe — no params, scalar object response.
func TestShape_Simple(t *testing.T) {
	m := &shapeMockDoer{}
	m.On("Do", mock.MatchedBy(func(r *http.Request) bool {
		return strings.HasSuffix(r.URL.Path, "/getMe")
	})).Return(shapeJSONResp(`{"ok":true,"result":{"id":1,"is_bot":true,"first_name":"x"}}`), nil)

	u, err := GetMe(context.Background(), newShapeBot(t, m), &GetMeParams{})
	require.NoError(t, err)
	require.Equal(t, int64(1), u.ID)
}

// TestShape_TypedStructParam: sendMessage — object param, object response.
func TestShape_TypedStructParam(t *testing.T) {
	m := &shapeMockDoer{}
	m.On("Do", mock.MatchedBy(func(r *http.Request) bool {
		return strings.HasSuffix(r.URL.Path, "/sendMessage")
	})).Return(shapeJSONResp(`{"ok":true,"result":{"message_id":7,"date":0,"chat":{"id":42,"type":"private"},"text":"hi"}}`), nil)

	msg, err := SendMessage(context.Background(), newShapeBot(t, m), &SendMessageParams{ChatID: int64(42), Text: "hi"})
	require.NoError(t, err)
	require.Equal(t, int64(7), msg.MessageID)
}

// TestShape_OptionalFields: SendMessage with only required fields. Verify
// optional fields are omitted from the request body.
func TestShape_OptionalFields(t *testing.T) {
	m := &shapeMockDoer{}
	m.On("Do", mock.MatchedBy(func(r *http.Request) bool {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(r.Body)
		body := buf.String()
		return strings.Contains(body, `"chat_id"`) && strings.Contains(body, `"text"`) &&
			!strings.Contains(body, `"parse_mode"`) // parse_mode should be omitted
	})).Return(shapeJSONResp(`{"ok":true,"result":{"message_id":1,"date":0,"chat":{"id":1,"type":"private"},"text":"hi"}}`), nil)

	_, err := SendMessage(context.Background(), newShapeBot(t, m), &SendMessageParams{ChatID: int64(1), Text: "hi"})
	require.NoError(t, err)
}

// TestShape_ArrayResult: getUpdates — empty array.
func TestShape_ArrayResult(t *testing.T) {
	m := &shapeMockDoer{}
	m.On("Do", mock.Anything).Return(shapeJSONResp(`{"ok":true,"result":[]}`), nil)

	ups, err := GetUpdates(context.Background(), newShapeBot(t, m), &GetUpdatesParams{})
	require.NoError(t, err)
	require.Empty(t, ups)
}

// TestShape_BoolResult: setWebhook — returns bool.
func TestShape_BoolResult(t *testing.T) {
	m := &shapeMockDoer{}
	m.On("Do", mock.Anything).Return(shapeJSONResp(`{"ok":true,"result":true}`), nil)

	ok, err := SetWebhook(context.Background(), newShapeBot(t, m), &SetWebhookParams{URL: "https://example.com"})
	require.NoError(t, err)
	require.True(t, ok)
}

// TestShape_MultipartUpload: sendDocument with InputFile streaming.
func TestShape_MultipartUpload(t *testing.T) {
	m := &shapeMockDoer{}
	m.On("Do", mock.MatchedBy(func(r *http.Request) bool {
		return strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data")
	})).Return(shapeJSONResp(`{"ok":true,"result":{"message_id":1,"date":0,"chat":{"id":1,"type":"private"}}}`), nil)

	_, err := SendDocument(context.Background(), newShapeBot(t, m), &SendDocumentParams{
		ChatID:   int64(1),
		Document: &InputFile{Reader: strings.NewReader("hello"), Filename: "x.txt"},
	})
	require.NoError(t, err)
}
```

- [ ] **Step 2: Run**

```bash
go test -race ./api/...
```

If any test fails because the generated method signature differs from what I wrote here (e.g. `GetMeParams` doesn't exist as a struct because the generator decided no-param methods don't get one), inspect `api/methods.gen.go` and adjust either the generator or the test. Prefer adjusting the generator: every method should have a `*Params` struct for symmetry, even if empty.

- [ ] **Step 3: Commit**

```bash
git add api/shape_test.go
git commit -m "test(api): shape smoke tests covering each codegen pattern"
```

---

## Task 14 — Wire Makefile + CI deterministic check

**Files:**
- Modify: `Makefile` (replace stubs with real targets)
- Modify: `.github/workflows/ci.yml` (add codegen-clean check)

- [ ] **Step 1: Replace Makefile stubs**

In `Makefile`, replace the four `Plan 2 — not yet implemented` stub targets with:

```makefile
SCRAPE_INPUT ?= testdata/html/snapshot_2026-05-08.html
SCRAPE_OUTPUT ?= internal/spec/api.json

snapshot:
	./scripts/snapshot.sh

regen:
	$(GO) run ./cmd/scrape -input testdata/html/latest.html -output $(SCRAPE_OUTPUT)
	$(GO) run ./cmd/genapi -input $(SCRAPE_OUTPUT) -outdir api

regen-from-fixture:
	$(GO) run ./cmd/scrape -input $(SCRAPE_INPUT) -output $(SCRAPE_OUTPUT)
	$(GO) run ./cmd/genapi -input $(SCRAPE_OUTPUT) -outdir api

test-update-golden:
	$(GO) test -run TestEmit -update ./cmd/genapi/...
	$(GO) test -run TestScrape -update ./cmd/scrape/...
```

- [ ] **Step 2: Verify make targets**

```bash
make regen-from-fixture
git diff --exit-code internal/spec/api.json api/   # must be clean — no drift
```

If the diff is non-empty, the previous run produced different output than what was committed. Investigate and fix the determinism (most likely cause: a `map` iteration in the emitter; use sorted iteration or move to slice-based config).

- [ ] **Step 3: Add CI codegen-clean step**

In `.github/workflows/ci.yml`, after `Test (race + cover)`, add a step:

```yaml
      - name: Codegen-clean check
        run: |
          make regen-from-fixture
          git diff --exit-code internal/spec/api.json api/
```

This proves the generated code committed matches what the pipeline produces from the committed snapshot.

- [ ] **Step 4: Commit**

```bash
git add Makefile .github/workflows/ci.yml
git commit -m "build(make): wire snapshot/regen/test-update-golden targets + CI clean check"
```

---

## Task 15 — Final verify, CHANGELOG, tag v0.2.0

**Files:**
- Modify: `CHANGELOG.md`
- Tag: `v0.2.0-codegen`

- [ ] **Step 1: Full verification**

```bash
go mod tidy
go vet ./...
go test -race ./...
go build ./...
make regen-from-fixture
git diff --exit-code
```

All must pass clean. If `git diff --exit-code` is non-zero after `regen-from-fixture`, fix and commit before tagging.

- [ ] **Step 2: Update CHANGELOG.md**

Insert a new section above the existing `## [Unreleased]`:

```markdown
## [0.2.0] - 2026-05-08

### Added
- Two-stage codegen pipeline (`cmd/scrape` → `internal/spec/api.json` → `cmd/genapi` → `api/*.gen.go`).
- Full Telegram Bot API surface generated from the committed `testdata/html/snapshot_2026-05-08.html`.
- Shape smoke tests covering each codegen pattern (simple, struct-param, optional-omit, array, bool, multipart).
- `make snapshot` / `make regen` / `make regen-from-fixture` / `make test-update-golden` targets.
- CI codegen-clean check asserting committed `api/` matches what the pipeline produces from the snapshot.

### Changed
- `api/types.go` and `api/methods.go` (hand-coded in v0.1.0) replaced by generated `api/*.gen.go`.

### Known limitations
- OneOf union types are emitted as bare interfaces; concrete variant decoding via discriminator is deferred.
- `ChatID` and similar `Integer or String` unions render as `any` at the source level. Callers pass `int64(N)` or `string(s)`.
- Enum constants (`ParseMode`, `ChatType`, `MessageEntityType`, `UpdateType`) are emitted from a curated static list, not extracted from prose.
```

- [ ] **Step 3: Commit + tag**

```bash
git add CHANGELOG.md
git commit -m "docs: CHANGELOG for v0.2.0-codegen"
git tag -a v0.2.0-codegen -m "Plan 2 complete: two-stage codegen pipeline + full generated api surface"
```

- [ ] **Step 4: Summary**

Report to controller: branch `feat/plan-2-codegen` HEAD, total commits, pass/fail status of all gates, tag created.

---

## Self-review notes

Spec coverage check:
- §6 codegen pipeline → Tasks 3–10 (scraper) + 8–10 (emitter) + 11 (full regen).
- §6.4 Makefile contract → Task 14.
- §11.2 shape smoke tests → Task 13.
- §11.4 change procedure → Tasks 7, 14 (procedure ops via make targets).
- §11.5 versioning → Task 15 (CHANGELOG + tag).

Known limitations spelled out in CHANGELOG (OneOf concrete decoding, `int64|string` rendered as `any`, enums hand-curated). These are acceptable for v0.2.0 and can be addressed in Plan 3 (deferred).

Subagents executing this plan should:
- Use `general-purpose` agent type with `haiku` for plumbing tasks (1, 2, 7, 11, 14, 15)
- Use `sonnet` for the heart of the codegen (Tasks 3, 4, 5, 6, 8, 9, 10, 12, 13) where reasoning quality matters
- Each task ends with a clean `go test -race ./...` and a single git commit
