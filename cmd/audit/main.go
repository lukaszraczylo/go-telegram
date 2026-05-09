// Command audit reports IR-level codegen fallbacks and signature drift.
//
// Usage:
//
//	audit -ir <path>             (default internal/spec/api.json)
//	audit -overrides <path>      (default internal/spec/overrides.json)
//	audit -drift                 (compare against -against ref's IR; off by default)
//	audit -against <ref>         (git ref to diff drift against; default HEAD~1)
//
// Exit codes:
//
//	0 — clean
//	1 — unaccounted bool fallbacks or any-typed fields
//	2 — drift detected (signature changed)
//	3 — invalid IR or overrides
package main

import (
	"flag"
	"fmt"
	"github.com/goccy/go-json"
	"os"
	"os/exec"
	"strings"

	"github.com/lukaszraczylo/go-telegram/internal/spec"
)

const (
	exitClean    = 0
	exitFallback = 1
	exitDrift    = 2
	exitInvalid  = 3
)

func main() {
	irPath := flag.String("ir", "internal/spec/api.json", "path to IR JSON")
	ovPath := flag.String("overrides", "internal/spec/overrides.json", "path to overrides JSON")
	checkDrift := flag.Bool("drift", false, "compare against -against ref's IR for signature changes")
	againstRef := flag.String("against", "HEAD~1", "git ref to diff drift against (e.g. origin/main, HEAD~1)")
	flag.Parse()

	api, err := loadIR(*irPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "audit:", err)
		os.Exit(exitInvalid)
	}
	overrides, err := spec.LoadOverrides(*ovPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "audit:", err)
		os.Exit(exitInvalid)
	}

	var problems []string

	problems = append(problems, auditBool(api, overrides)...)
	problems = append(problems, auditAny(api)...)

	driftFound := false
	if *checkDrift {
		if d, err := auditDrift(*irPath, *againstRef, api); err != nil {
			fmt.Fprintln(os.Stderr, "audit: drift check skipped:", err)
		} else if len(d) > 0 {
			fmt.Println("Drift detected (signatures changed since HEAD):")
			for _, p := range d {
				fmt.Println("  ", p)
			}
			driftFound = true
		}
	}

	if len(problems) == 0 && !driftFound {
		fmt.Println("audit: clean")
		os.Exit(exitClean)
	}

	if len(problems) > 0 {
		fmt.Println("Codegen fallbacks requiring action:")
		for _, p := range problems {
			fmt.Println("  ", p)
		}
		fmt.Println()
		fmt.Println("To resolve: extend cmd/scrape/method.go regex patterns OR")
		fmt.Println("add an entry to internal/spec/overrides.json.")
		os.Exit(exitFallback)
	}

	// drift only, no fallbacks
	os.Exit(exitDrift)
}

func loadIR(path string) (*spec.API, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open IR: %w", err)
	}
	defer func() { _ = f.Close() }()
	var api spec.API
	if err := json.NewDecoder(f).Decode(&api); err != nil {
		return nil, fmt.Errorf("decode IR: %w", err)
	}
	return &api, nil
}

// auditBool returns problems for methods returning bool whose docs don't
// actually say "Returns True" / etc. and which aren't in the approved list.
func auditBool(api *spec.API, ov *spec.Overrides) []string {
	var out []string
	for _, m := range api.Methods {
		if m.Returns.Kind != spec.KindPrimitive || m.Returns.Name != "bool" {
			continue
		}
		if ov.IsBoolApproved(m.Name) {
			continue
		}
		if looksGenuinelyBool(m.Doc) {
			continue
		}
		snippet := m.Doc
		if len(snippet) > 120 {
			snippet = snippet[:120] + "…"
		}
		out = append(out, fmt.Sprintf("bool fallback: %s — doc: %q", m.Name, snippet))
	}
	return out
}

func looksGenuinelyBool(doc string) bool {
	for _, p := range []string{
		"Returns True", "Returns true",
		"True is returned", "true is returned",
		"Returns Boolean", "Returns Bool",
	} {
		if strings.Contains(doc, p) {
			return true
		}
	}
	return false
}

// auditAny scans the IR for any KindOneOf TypeRef that would render as
// `any` in generated code (not matched by ChatID/InputFile-or-string/known
// union heuristics). Reports each occurrence with location.
func auditAny(api *spec.API) []string {
	var out []string
	isKnownUnion := func(variants []string) bool {
		if hasVariants(variants, "int64", "string") {
			return true // ChatID
		}
		if hasVariants(variants, "InputFile", "string") {
			return true // *InputFile
		}
		// ReplyMarkup union: all four keyboard types — emitter renders as `any` intentionally
		if hasVariants(variants, "InlineKeyboardMarkup", "ReplyKeyboardMarkup", "ReplyKeyboardRemove", "ForceReply") {
			return true
		}
		for _, t := range api.Types {
			if len(t.OneOf) > 0 && sameSet(variants, t.OneOf) {
				return true
			}
		}
		return false
	}
	isAny := func(tr spec.TypeRef) bool {
		return tr.Kind == spec.KindOneOf && !isKnownUnion(tr.Variants)
	}
	for _, t := range api.Types {
		for _, f := range t.Fields {
			if isAny(f.Type) {
				out = append(out, fmt.Sprintf("any field: %s.%s (variants=%v)", t.Name, f.Name, f.Type.Variants))
			}
		}
	}
	for _, m := range api.Methods {
		if isAny(m.Returns) {
			out = append(out, fmt.Sprintf("any return: %s (variants=%v)", m.Name, m.Returns.Variants))
		}
		for _, p := range m.Params {
			if isAny(p.Type) {
				out = append(out, fmt.Sprintf("any param: %s.%s (variants=%v)", m.Name, p.Name, p.Type.Variants))
			}
		}
	}
	return out
}

func hasVariants(got []string, want ...string) bool {
	if len(got) != len(want) {
		return false
	}
	seen := map[string]int{}
	for _, g := range got {
		seen[g]++
	}
	for _, w := range want {
		seen[w]--
	}
	for _, v := range seen {
		if v != 0 {
			return false
		}
	}
	return true
}

func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	return hasVariants(a, b...)
}

// auditDrift compares method/return signatures between the given git ref's
// version of irPath and the in-memory current IR. Returns a list of
// human-readable change descriptions.
func auditDrift(irPath, againstRef string, current *spec.API) ([]string, error) {
	cmd := exec.Command("git", "show", againstRef+":"+irPath) // #nosec G204 - operator tool, ref controlled by caller
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git show %s: %w", againstRef, err)
	}
	var prev spec.API
	if err := json.Unmarshal(out, &prev); err != nil {
		return nil, fmt.Errorf("decode %s IR: %w", againstRef, err)
	}
	return diffSignatures(&prev, current), nil
}

func diffSignatures(prev, cur *spec.API) []string {
	var changes []string

	pmeth := indexByName(prev.Methods, func(m spec.MethodDecl) string { return m.Name })
	cmeth := indexByName(cur.Methods, func(m spec.MethodDecl) string { return m.Name })

	for name, p := range pmeth {
		c, ok := cmeth[name]
		if !ok {
			changes = append(changes, fmt.Sprintf("removed method: %s", name))
			continue
		}
		if !typeRefEqual(p.Returns, c.Returns) {
			changes = append(changes, fmt.Sprintf(
				"method %s return changed: %s → %s",
				name, formatTypeRef(p.Returns), formatTypeRef(c.Returns)))
		}
	}
	for name := range cmeth {
		if _, ok := pmeth[name]; !ok {
			changes = append(changes, fmt.Sprintf("added method: %s", name))
		}
	}
	return changes
}

func indexByName[T any](xs []T, f func(T) string) map[string]T {
	out := map[string]T{}
	for _, x := range xs {
		out[f(x)] = x
	}
	return out
}

func typeRefEqual(a, b spec.TypeRef) bool {
	if a.Kind != b.Kind || a.Name != b.Name {
		return false
	}
	if (a.ElemType == nil) != (b.ElemType == nil) {
		return false
	}
	if a.ElemType != nil && !typeRefEqual(*a.ElemType, *b.ElemType) {
		return false
	}
	return sameSet(a.Variants, b.Variants)
}

func formatTypeRef(t spec.TypeRef) string {
	switch t.Kind {
	case spec.KindPrimitive:
		return t.Name
	case spec.KindNamed:
		return t.Name
	case spec.KindArray:
		if t.ElemType != nil {
			return "[]" + formatTypeRef(*t.ElemType)
		}
		return "[]any"
	case spec.KindOneOf:
		return "(" + strings.Join(t.Variants, " | ") + ")"
	}
	return "?"
}
