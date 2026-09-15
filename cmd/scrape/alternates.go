package main

import (
	"regexp"

	"github.com/lukaszraczylo/go-telegram/internal/spec"
)

// altStringRE matches the scalar-string alternate clause Telegram uses
// for union types that can also decode as plain text on the wire, e.g.
// "a String for plain text". The trailing "for ..." clause is optional
// and, when present, is not captured — only the shape matters. "an"
// (rather than "a") is tolerated so a future rephrasing ("an String")
// doesn't silently stop matching, and the word boundary keeps "Strings"
// or "SubString" from misfiring.
var altStringRE = regexp.MustCompile(`(?i)\ban?\s+String\b`)

// altArrayRE builds the pattern that matches the self-referential array
// alternate clause Telegram uses for recursive union types, e.g. "an
// Array of RichText". Matching is anchored to the union's own name so a
// doc mentioning "an Array of" some other type is never mistaken for a
// shape alternate.
func altArrayRE(unionName string) *regexp.Regexp {
	return regexp.MustCompile(`(?i)\ban?\s+Array\s+of\s+` + regexp.QuoteMeta(unionName) + `\b`)
}

// extractAlternates detects Telegram's "can be either a String for
// plain text, an Array of <Union>, or any of the following types:"
// phrasing on a union type's doc string and returns the non-object
// shapes it declares, in a fixed (string, then array) order regardless
// of how the sentence orders its clauses.
//
// Conservative by design: callers invoke this only for types that
// already have OneOf populated (see typeFromSection in scrape.go), and
// it only recognises the two shapes Telegram's docs currently use for
// this ("a String" and "an Array of <the union's own name>") — anything
// else in the doc is ignored, so free-text mentioning "String" or
// "Array" elsewhere in a union's description can't produce a false
// positive for a type this pattern wasn't written for. As of Bot API
// v10, RichText is the only union whose doc matches either pattern.
func extractAlternates(unionName, doc string) []spec.Alternate {
	var alts []spec.Alternate
	if altStringRE.MatchString(doc) {
		alts = append(alts, spec.Alternate{Shape: spec.AlternateString})
	}
	if altArrayRE(unionName).MatchString(doc) {
		alts = append(alts, spec.Alternate{Shape: spec.AlternateArray})
	}
	return alts
}
