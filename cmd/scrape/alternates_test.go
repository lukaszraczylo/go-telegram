package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lukaszraczylo/go-telegram/internal/spec"
)

func TestExtractAlternates_RichTextDoc(t *testing.T) {
	// Exact doc string as scraped from the live Bot API v10 docs page
	// (internal/spec/api.json / testdata/html/latest.html).
	doc := "This object represents a rich formatted text. Currently, it can be either a String for plain text, " +
		"an Array of RichText, or any of the following types:"
	got := extractAlternates("RichText", doc)
	require.Equal(t, []spec.Alternate{
		{Shape: spec.AlternateString},
		{Shape: spec.AlternateArray},
	}, got)
}

func TestExtractAlternates_OrderIndependent(t *testing.T) {
	// Same two clauses, reversed, plus an extra clause in between —
	// order and surrounding prose must not matter.
	doc := "Currently, it can be either an Array of RichText, something else entirely, a String for plain text, " +
		"or any of the following types:"
	got := extractAlternates("RichText", doc)
	require.Equal(t, []spec.Alternate{
		{Shape: spec.AlternateString},
		{Shape: spec.AlternateArray},
	}, got)
}

func TestExtractAlternates_ObjectOnlyUnion_NoAlternates(t *testing.T) {
	// RichBlock's doc is object-only: no String/Array alternate clause.
	doc := "This object represents a block in a rich formatted message. Currently, it can be any of the following types:"
	got := extractAlternates("RichBlock", doc)
	require.Nil(t, got)
}

func TestExtractAlternates_ArrayOfDifferentType_NotSelfReferential(t *testing.T) {
	// "an Array of X" only counts when X is the union's own name — an
	// array of some other type mentioned in the doc must not match.
	doc := "This object can be either an Array of PhotoSize or any of the following types:"
	got := extractAlternates("RichText", doc)
	require.Nil(t, got)
}

func TestExtractAlternates_StringOnly(t *testing.T) {
	doc := "This object can be either a String or any of the following types:"
	got := extractAlternates("Widget", doc)
	require.Equal(t, []spec.Alternate{{Shape: spec.AlternateString}}, got)
}

func TestExtractAlternates_PlainTypeDoc_NoFalsePositive(t *testing.T) {
	// Ordinary field/type prose mentioning neither shape must not match.
	doc := "This object represents a chat photo."
	got := extractAlternates("ChatPhoto", doc)
	require.Nil(t, got)
}

// TestTypeFromSection_RichText_RealSnapshot grounds the extraction in the
// real pinned docs snapshot (not just a literal fixture string): scraping
// testdata/html/latest.html end-to-end must produce a RichText TypeDecl
// with both alternates, matching the same live text that broke decoding
// of plain-text and array-shaped rich_message payloads.
func TestTypeFromSection_RichText_RealSnapshot(t *testing.T) {
	htmlBytes, err := os.ReadFile("../../testdata/html/latest.html")
	require.NoError(t, err)

	api, err := scrape(htmlBytes)
	require.NoError(t, err)

	var richText *spec.TypeDecl
	for i := range api.Types {
		if api.Types[i].Name == "RichText" {
			richText = &api.Types[i]
			break
		}
	}
	require.NotNil(t, richText, "RichText type must be present in the real docs snapshot")
	require.Equal(t, []spec.Alternate{
		{Shape: spec.AlternateString},
		{Shape: spec.AlternateArray},
	}, richText.Alternates)

	// RichBlock is the control: object-only union, no alternates.
	var richBlock *spec.TypeDecl
	for i := range api.Types {
		if api.Types[i].Name == "RichBlock" {
			richBlock = &api.Types[i]
			break
		}
	}
	require.NotNil(t, richBlock)
	require.Nil(t, richBlock.Alternates)
}
