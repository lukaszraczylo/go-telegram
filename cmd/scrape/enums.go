package main

import (
	"regexp"
	"strings"
)

// extractEnumValues inspects a field-description string and returns the
// list of wire-level string values when the description matches one of
// the enum-like patterns Telegram uses in its docs. Order follows doc
// order; duplicates are removed but order of first occurrence is kept.
//
// Handled patterns (curly quotes “…” are required to avoid false
// positives on free-text quoting):
//
//   - "Type of the chat, can be either “private”, “group”, … or “channel”"
//   - "Currently, can be “mention”, “hashtag”, …"
//   - "Currently, one of “XTR” … or “TON” …"
//   - "Currently, must be one of “XTR” …"
//   - "Currently, it can be one of “pending”, “approved”, “declined”."
//   - "Must be one of “danger” …, “success” …"
//   - "Must be one of “image/jpeg”, “image/gif”, or “video/mp4”"
//   - "Format … must be one of “static” …, “animated” …, “video” …"
//   - "Currently, either “upgrade” …, “transfer” …, “resale” …"
//   - "..., always “creator”"
//   - parse_mode parameter special case ("Mode for parsing entities …")
//     emits the canonical Markdown / MarkdownV2 / HTML triple.
//
// Returns nil when the description does not look like an enum.
func extractEnumValues(jsonName, desc string) []string {
	if values := parseModeEnumValues(jsonName, desc); values != nil {
		return values
	}

	trigger, triggerEnd, isAlways := findEnumTrigger(desc)
	if trigger < 0 {
		return nil
	}
	tail := desc[trigger:]

	values := collectQuotedValues(tail)
	if len(values) == 0 {
		return nil
	}
	// First quoted value must sit close to the trigger phrase (e.g.
	// "can be “private”…"). Phrasings like "can be available only for
	// “invoice_payment”…" introduce a referenced value, not an enum,
	// and the gap between trigger end and first quote rules them out.
	firstQuote := strings.Index(desc[triggerEnd:], "“")
	if firstQuote < 0 {
		return nil
	}
	gap := desc[triggerEnd : triggerEnd+firstQuote]
	// Allow "always " as a permitted bridge (e.g. "Currently, always
	// “XTR”") and promote the match to single-value form.
	if strings.Contains(strings.ToLower(gap), "always ") {
		isAlways = true
	} else if firstQuote > 8 {
		return nil
	}
	// Single-value matches are only credible after "always". Multi-
	// value matches are credible after any trigger; the trigger phrase
	// already constrained the context.
	if !isAlways && len(values) < 2 {
		return nil
	}
	for _, v := range values {
		if !looksLikeEnumValue(v) {
			return nil
		}
	}
	return dedupeStrings(values)
}

// parseMode parameters do not list values inline — Telegram links to a
// separate "formatting options" section. We hardcode the canonical set
// here so callers get a typed ParseMode without writing magic strings.
func parseModeEnumValues(jsonName, desc string) []string {
	if !strings.HasSuffix(jsonName, "parse_mode") {
		return nil
	}
	if !strings.Contains(desc, "Mode for parsing entities") {
		return nil
	}
	return []string{"Markdown", "MarkdownV2", "HTML"}
}

// enumTriggers are anchor phrases that introduce a list of valid wire
// values. Order matches longest-prefix priority; the matcher uses the
// earliest match in the description.
var enumTriggers = []string{
	"can be either ",
	"can be one of ",
	"can be ",
	"must be one of ",
	"must be ",
	"currently one of ",
	"currently, one of ",
	"currently, either ",
	"currently, must be one of ",
	"currently, can be ",
	"currently, it can be one of ",
	"currently, ",
	"one of ",
	"either ",
	"always ",
}

// findEnumTrigger returns the byte offset where the first enum trigger
// phrase begins, the offset just past the phrase, and whether the
// trigger is the single-value "always" form. Returns (-1, -1, false)
// when no trigger matches. Matching is case-insensitive so "Currently"
// and "currently" both fire.
func findEnumTrigger(desc string) (int, int, bool) {
	lower := strings.ToLower(desc)
	bestStart := -1
	bestEnd := -1
	bestAlways := false
	for _, t := range enumTriggers {
		i := strings.Index(lower, t)
		if i < 0 {
			continue
		}
		if bestStart != -1 && i >= bestStart {
			// Earlier-trigger wins outright; on a tie, the longer trigger
			// (which we visit first) already populated bestEnd.
			continue
		}
		bestStart = i
		bestEnd = i + len(t)
		bestAlways = t == "always "
	}
	return bestStart, bestEnd, bestAlways
}

// quotedRE matches a curly-quoted token: “value”.
var quotedRE = regexp.MustCompile(`“([^”]*)”`)

// collectQuotedValues returns the contents of every “…” pair in s in
// order. Multi-line is fine; the docs use single-paragraph cells.
func collectQuotedValues(s string) []string {
	matches := quotedRE.FindAllStringSubmatch(s, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		out = append(out, m[1])
	}
	return out
}

// looksLikeEnumValue returns true for short identifiers that fit the
// shape of a Telegram wire enum. This rules out values like
// "attach://…", "h264", arbitrary URLs, and stylised punctuation.
//
// Permitted shapes:
//
//	a-z0-9_                            (e.g. "private", "bot_command")
//	A-Z0-9_                            (e.g. "XTR", "TON", "MarkdownV2")
//	mixed case incl. "/" once          (e.g. "image/jpeg", "video/mp4")
func looksLikeEnumValue(v string) bool {
	if v == "" || len(v) > 64 {
		return false
	}
	if strings.Contains(v, "://") || strings.Contains(v, " ") {
		return false
	}
	// "image/jpeg"-style mime types: at most one slash, both halves alnum.
	if i := strings.Index(v, "/"); i >= 0 {
		if strings.Count(v, "/") > 1 {
			return false
		}
		left, right := v[:i], v[i+1:]
		return isIdent(left) && isIdent(right)
	}
	return isIdent(v)
}

func isIdent(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '_' || r == '-' || r == '.':
		default:
			return false
		}
	}
	return true
}

func dedupeStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
