package main

import (
	"regexp"
	"strings"

	"github.com/lukaszraczylo/go-telegram/internal/spec"
)

// extractReturn pulls the return type from a method's description prose.
//
// Patterns we handle (in priority order):
//
//	"Returns an Array of X" / "On success, an Array of X is returned" → array of named X
//	"an array of X of the sent messages is returned"                  → array of named X
//	"the edited X is returned, otherwise True is returned"             → XOrBool
//	"Returns ... as a X object" / "Returns ... as X object"           → named X
//	"Returns ... as String on success"                                → string
//	"On success, returns a X object" / "Returns a X object"           → named X (indefinite article)
//	"On success, an? X is returned" / "On success, the X is returned" → named X
//	"Returns True" / "On success, true is returned"                   → bool
//	"Returns the verb-ed X"                                           → named X
//	"On success, X is returned"                                       → named X
//	"Returns X on success" (no article)                               → named X
//	"in form of a X"                                                  → named X
//	fallback: bool
func extractReturn(desc string) spec.TypeRef {
	// Normalise; strip *bold* markers because Telegram uses italics.
	d := strings.ReplaceAll(desc, "*", "")

	patterns := []struct {
		re *regexp.Regexp
		fn func([]string) spec.TypeRef
	}{
		// Array patterns first — most specific.
		{regexp.MustCompile(`Returns an? [Aa]rray of ([A-Z][A-Za-z0-9]+)`), func(m []string) spec.TypeRef {
			elem := primitiveOrNamed(m[1])
			return spec.TypeRef{Kind: spec.KindArray, ElemType: &elem}
		}},
		{regexp.MustCompile(`On success(?:,)?\s+(?:an?\s+)?[Aa]rray of ([A-Z][A-Za-z0-9]+)(?:\s+objects?)?\s+(?:is|are|that\s+\S+\s+\S+\s+)?(?:is |are )?returned`), func(m []string) spec.TypeRef {
			elem := primitiveOrNamed(m[1])
			return spec.TypeRef{Kind: spec.KindArray, ElemType: &elem}
		}},
		// "an array of X of the sent messages is returned" (ForwardMessages/CopyMessages shape).
		{regexp.MustCompile(`(?:[Oo]n success[,.]?\s+)?an? array of ([A-Z][A-Za-z0-9]+)(?:\s+of [^.]+?)?\s+(?:objects\s+)?(?:is|are) returned`), func(m []string) spec.TypeRef {
			elem := primitiveOrNamed(m[1])
			return spec.TypeRef{Kind: spec.KindArray, ElemType: &elem}
		}},
		// "Message or True" conditional return → XOrBool sentinel.
		{regexp.MustCompile(`the (?:edited|sent|stopped)?\s*([A-Z][A-Za-z0-9]+)\s+is returned, otherwise (?:True|true) is returned`), func(m []string) spec.TypeRef {
			return spec.TypeRef{Kind: spec.KindNamed, Name: m[1] + "OrBool"}
		}},
		// "Returns ... as a X object" / "Returns ... as X object" (with or without article).
		{regexp.MustCompile(`[Rr]eturns? (?:.+? )?as (?:an? )?([A-Z][A-Za-z0-9]+) object`), func(m []string) spec.TypeRef {
			return primitiveOrNamed(m[1])
		}},
		// "Returns ... as String on success" / "Returns ... as X on success" (named type after "as").
		{regexp.MustCompile(`[Rr]eturns? (?:.+? )?as ([A-Z][A-Za-z0-9]+) on success`), func(m []string) spec.TypeRef {
			return primitiveOrNamed(m[1])
		}},
		// Indefinite article: "On success, returns a X object" / "Returns a X object".
		{regexp.MustCompile(`(?:[Oo]n success[,.]?\s+)?[Rr]eturns? an? ([A-Z][A-Za-z0-9]+)(?:\s+object)?`), func(m []string) spec.TypeRef {
			return primitiveOrNamed(m[1])
		}},
		// "On success, an? X is returned" / "On success, the stopped X is returned".
		{regexp.MustCompile(`On success,\s+(?:an?|the)?\s*(?:[a-z]+\s+)?([A-Z][A-Za-z0-9]+)(?:\s+object)?\s+is returned`), func(m []string) spec.TypeRef {
			return primitiveOrNamed(m[1])
		}},
		// Explicit True — must come before the broad "Returns X" pattern.
		{regexp.MustCompile(`Returns True`), func(m []string) spec.TypeRef {
			return spec.TypeRef{Kind: spec.KindPrimitive, Name: "bool"}
		}},
		{regexp.MustCompile(`(?i)on success, true is returned`), func(m []string) spec.TypeRef {
			return spec.TypeRef{Kind: spec.KindPrimitive, Name: "bool"}
		}},
		// "Returns the verb-ed X" — accepts any verb prefix (uploaded, revoked, …).
		{regexp.MustCompile(`Returns (?:the|an?)\s+(?:[a-z]+ )?([A-Z][A-Za-z0-9]+)`), func(m []string) spec.TypeRef {
			return primitiveOrNamed(m[1])
		}},
		// "On success, X is returned" (no article).
		{regexp.MustCompile(`On success(?:,)?\s+(?:the\s+)?(?:newly\s+)?(?:edited\s+|sent\s+|created\s+|updated\s+)?([A-Z][A-Za-z0-9]+)\s+is returned`), func(m []string) spec.TypeRef {
			return primitiveOrNamed(m[1])
		}},
		// "Returns X on success" (no article, e.g. "Returns OwnedGifts on success").
		{regexp.MustCompile(`[Rr]eturns ([A-Z][A-Za-z0-9]+) on success`), func(m []string) spec.TypeRef {
			return primitiveOrNamed(m[1])
		}},
		// "in form of a X".
		{regexp.MustCompile(`in (?:the )?form of (?:a )?([A-Z][A-Za-z0-9]+)`), func(m []string) spec.TypeRef {
			return primitiveOrNamed(m[1])
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
		return tr.Name == "InputFile" || strings.HasPrefix(tr.Name, "InputMedia") || strings.HasPrefix(tr.Name, "InputPaidMedia")
	case spec.KindArray:
		if tr.ElemType != nil {
			return mentionsInputFile(*tr.ElemType)
		}
	case spec.KindOneOf:
		for _, v := range tr.Variants {
			if v == "InputFile" || strings.HasPrefix(v, "InputMedia") || strings.HasPrefix(v, "InputPaidMedia") {
				return true
			}
		}
	}
	return false
}

// extractVersion finds the API version string in a "Bot API X.Y[.Z]" heading.
var versionRE = regexp.MustCompile(`Bot API (\d+\.\d+(?:\.\d+)?)`)

// extractVersion finds the API version string. The live docs page emits
// the version as "<strong>Bot API X.Y</strong>" inside a paragraph below
// a date heading; the small fixture uses an h4 "Bot API X.Y" instead.
// Both shapes are handled here by also scanning section descriptions.
func extractVersion(sections []section) string {
	for _, s := range sections {
		if m := versionRE.FindStringSubmatch(s.Title); m != nil {
			return m[1]
		}
		if m := versionRE.FindStringSubmatch(s.Description); m != nil {
			return m[1]
		}
	}
	return ""
}
