package main

import (
	"testing"

	"github.com/lukaszraczylo/go-telegram/internal/spec"
	"github.com/stretchr/testify/require"
)

// TestDeriveDiscriminators_RichBlockAndRichText_FromRealSpec is a
// regression test for the Bot API v10 rich_message decode failure:
// RichBlock and RichText had no knownDiscriminators entry, so genapi
// never emitted UnmarshalRichBlock / UnmarshalRichText and any update
// whose message carried rich_message failed to decode. Derivation must
// pick "type" for both and cover every variant the spec declares.
//
// The expected variant counts are read from the spec rather than pinned
// as literals. Telegram adds variants in most releases - v10.3 took
// RichBlock from 21 to 24 and RichText from 25 to 26 - so a frozen
// number turns a routine regeneration into a spurious failure while
// testing nothing the coverage assertion does not already cover.
func TestDeriveDiscriminators_RichBlockAndRichText_FromRealSpec(t *testing.T) {
	api, err := loadAPI("../../internal/spec/api.json")
	require.NoError(t, err)

	// declaredVariants returns how many concrete variants the spec lists
	// for a union, so the assertions below track the spec instead of a
	// hardcoded count.
	declaredVariants := func(union string) int {
		for i := range api.Types {
			if api.Types[i].Name == union {
				return len(api.Types[i].OneOf)
			}
		}
		t.Fatalf("union %q not found in spec", union)
		return 0
	}

	derived, _ := deriveDiscriminators(api)

	rb, ok := derived["RichBlock"]
	require.True(t, ok, "RichBlock must derive a discriminator")
	require.Equal(t, "type", rb.Field)
	require.Len(t, rb.Variants, declaredVariants("RichBlock"),
		"every RichBlock variant declared in the spec must be covered")
	require.Equal(t, "RichBlockParagraph", rb.Variants["paragraph"])
	require.Equal(t, "RichBlockTable", rb.Variants["table"])
	require.Equal(t, "RichBlockThinking", rb.Variants["thinking"])

	rt, ok := derived["RichText"]
	require.True(t, ok, "RichText must derive a discriminator")
	require.Equal(t, "type", rt.Field)
	require.Len(t, rt.Variants, declaredVariants("RichText"),
		"every RichText variant declared in the spec must be covered")
	require.Equal(t, "RichTextBold", rt.Variants["bold"])
	require.Equal(t, "RichTextAnchorLink", rt.Variants["anchor_link"])
	require.Equal(t, "RichTextReferenceLink", rt.Variants["reference_link"])
}

// TestMergeDiscriminators_HandCuratedOverridesDerived confirms a
// knownDiscriminators entry always wins the merge even when derivation
// would also have produced a (different) answer for the same union —
// hand-curation is the escape hatch for a union the generic rule gets
// wrong.
func TestMergeDiscriminators_HandCuratedOverridesDerived(t *testing.T) {
	api := &spec.API{
		Types: []spec.TypeDecl{
			{Name: "Widget", OneOf: []string{"WidgetA", "WidgetB"}},
			{Name: "WidgetA", Fields: []spec.Field{
				{Name: "Type", JSONName: "type", Required: true,
					Type: spec.TypeRef{Kind: spec.KindPrimitive, Name: "string"}, EnumValues: []string{"a"}},
			}},
			{Name: "WidgetB", Fields: []spec.Field{
				{Name: "Type", JSONName: "type", Required: true,
					Type: spec.TypeRef{Kind: spec.KindPrimitive, Name: "string"}, EnumValues: []string{"b"}},
			}},
		},
	}

	derived, ungated := deriveDiscriminators(api)
	require.Empty(t, ungated)
	require.Equal(t, discriminatorSpec{
		Field:    "type",
		Variants: map[string]string{"a": "WidgetA", "b": "WidgetB"},
	}, derived["Widget"])

	// Inject a hand-curated override that disagrees with the derived
	// spec, merge, and confirm the override wins outright.
	override := discriminatorSpec{Field: "kind", Variants: map[string]string{"custom": "WidgetA"}}
	knownDiscriminators["Widget"] = override
	t.Cleanup(func() { delete(knownDiscriminators, "Widget") })

	merged := mergeDiscriminators(derived)
	require.Equal(t, override, merged["Widget"])
}

// TestCheckDiscriminatorGate_FailsOnUngatedInboundUnion confirms the
// build gate fails generation when a union Telegram can actually send
// has no derivable discriminator: the variants share no common
// Type/Status/Source field at all.
func TestCheckDiscriminatorGate_FailsOnUngatedInboundUnion(t *testing.T) {
	api := &spec.API{
		Types: []spec.TypeDecl{
			{Name: "Widget", OneOf: []string{"WidgetA", "WidgetB"}},
			{Name: "WidgetA", Fields: []spec.Field{
				{Name: "Kind", JSONName: "kind", Required: true, Type: spec.TypeRef{Kind: spec.KindPrimitive, Name: "string"}},
			}},
			{Name: "WidgetB", Fields: []spec.Field{
				{Name: "Flavor", JSONName: "flavor", Required: true, Type: spec.TypeRef{Kind: spec.KindPrimitive, Name: "string"}},
			}},
			// Container is a plain (non-Input) type with a field of the
			// undiscriminated union — this is what makes Widget reachable
			// from real inbound Telegram data.
			{Name: "Container", Fields: []spec.Field{
				{Name: "W", JSONName: "w", Type: spec.TypeRef{Kind: spec.KindNamed, Name: "Widget"}},
			}},
		},
	}

	err := checkDiscriminatorGate(api)
	require.Error(t, err)
	require.ErrorContains(t, err, "Widget")
	require.ErrorContains(t, err, "WidgetA")
	require.ErrorContains(t, err, "WidgetB")
}

// TestCheckDiscriminatorGate_SilentForInputOnlyUnion confirms the same
// undiscriminated union does NOT fail the build when its only reference
// is a field on an Input*-prefixed type — those are always constructed
// and marshalled by the caller, never decoded from a Telegram response.
func TestCheckDiscriminatorGate_SilentForInputOnlyUnion(t *testing.T) {
	api := &spec.API{
		Types: []spec.TypeDecl{
			{Name: "Widget", OneOf: []string{"WidgetA", "WidgetB"}},
			{Name: "WidgetA", Fields: []spec.Field{
				{Name: "Kind", JSONName: "kind", Required: true, Type: spec.TypeRef{Kind: spec.KindPrimitive, Name: "string"}},
			}},
			{Name: "WidgetB", Fields: []spec.Field{
				{Name: "Flavor", JSONName: "flavor", Required: true, Type: spec.TypeRef{Kind: spec.KindPrimitive, Name: "string"}},
			}},
			{Name: "InputContainer", Fields: []spec.Field{
				{Name: "W", JSONName: "w", Type: spec.TypeRef{Kind: spec.KindNamed, Name: "Widget"}},
			}},
		},
	}

	require.NoError(t, checkDiscriminatorGate(api))
}
