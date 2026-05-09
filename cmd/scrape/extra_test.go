package main

import (
	"testing"

	"github.com/lukaszraczylo/go-telegram/internal/spec"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// parseTypeRef — edge cases
// ---------------------------------------------------------------------------

func TestParseTypeRef_Empty(t *testing.T) {
	// Empty string → named with empty name (fallback).
	got := parseTypeRef("")
	require.Equal(t, spec.KindNamed, got.Kind)
	require.Equal(t, "", got.Name)
}

func TestParseTypeRef_Whitespace(t *testing.T) {
	got := parseTypeRef("  Integer  ")
	require.Equal(t, spec.KindPrimitive, got.Kind)
	require.Equal(t, "int64", got.Name)
}

func TestParseTypeRef_True(t *testing.T) {
	got := parseTypeRef("True")
	require.Equal(t, spec.KindPrimitive, got.Kind)
	require.Equal(t, "bool", got.Name)
}

func TestParseTypeRef_False(t *testing.T) {
	got := parseTypeRef("False")
	require.Equal(t, spec.KindPrimitive, got.Kind)
	require.Equal(t, "bool", got.Name)
}

func TestParseTypeRef_FloatNumber(t *testing.T) {
	got := parseTypeRef("Float number")
	require.Equal(t, spec.KindPrimitive, got.Kind)
	require.Equal(t, "float64", got.Name)
}

func TestParseTypeRef_Int(t *testing.T) {
	got := parseTypeRef("Int")
	require.Equal(t, spec.KindPrimitive, got.Kind)
	require.Equal(t, "int64", got.Name)
}

func TestParseTypeRef_Bool(t *testing.T) {
	got := parseTypeRef("Bool")
	require.Equal(t, spec.KindPrimitive, got.Kind)
	require.Equal(t, "bool", got.Name)
}

func TestParseTypeRef_CommaAndUnion(t *testing.T) {
	// "Foo, Bar and Baz" → oneOf{Foo, Bar, Baz}
	got := parseTypeRef("InputMediaPhoto, InputMediaVideo and InputMediaDocument")
	require.Equal(t, spec.KindOneOf, got.Kind)
	require.Len(t, got.Variants, 3)
	require.Contains(t, got.Variants, "InputMediaPhoto")
	require.Contains(t, got.Variants, "InputMediaVideo")
	require.Contains(t, got.Variants, "InputMediaDocument")
}

func TestParseTypeRef_ArrayOfNothing(t *testing.T) {
	// "Array of " with trailing space — TrimSpace removes the trailing space
	// leaving "Array of" which does NOT match the "Array of " prefix, so it
	// falls through to primitiveOrNamed and returns KindNamed (not KindArray).
	got := parseTypeRef("Array of ")
	require.Equal(t, spec.KindNamed, got.Kind)
}

// ---------------------------------------------------------------------------
// splitCommaAnd
// ---------------------------------------------------------------------------

func TestSplitCommaAnd_ThreeVariants(t *testing.T) {
	got := splitCommaAnd("A, B and C")
	require.Equal(t, []string{"A", "B", "C"}, got)
}

func TestSplitCommaAnd_FourVariants(t *testing.T) {
	got := splitCommaAnd("A, B, C and D")
	require.Equal(t, []string{"A", "B", "C", "D"}, got)
}

func TestSplitCommaAnd_ExtraSpaces(t *testing.T) {
	got := splitCommaAnd("  Foo ,  Bar  and  Baz  ")
	require.Len(t, got, 3)
}

// ---------------------------------------------------------------------------
// goName — edge cases
// ---------------------------------------------------------------------------

func TestGoName_Empty(t *testing.T) {
	require.Equal(t, "", goName(""))
}

func TestGoName_SingleWord(t *testing.T) {
	require.Equal(t, "Photo", goName("photo"))
}

func TestGoName_JSON(t *testing.T) {
	require.Equal(t, "JSON", goName("json"))
}

func TestGoName_HTML(t *testing.T) {
	require.Equal(t, "HTML", goName("html"))
}

func TestGoName_HTTPS(t *testing.T) {
	require.Equal(t, "HTTPS", goName("https"))
}

func TestGoName_AlreadyUpperSegment(t *testing.T) {
	// Segment that starts with uppercase letter should be passed through.
	require.Equal(t, "MediaGroupID", goName("media_group_id"))
}

// ---------------------------------------------------------------------------
// extractReturn — additional patterns
// ---------------------------------------------------------------------------

func TestExtractReturn_ArrayPattern(t *testing.T) {
	desc := "Returns an Array of Update objects."
	got := extractReturn(desc)
	require.Equal(t, spec.KindArray, got.Kind)
	require.Equal(t, "Update", got.ElemType.Name)
}

func TestExtractReturn_BoolPattern(t *testing.T) {
	desc := "Returns True on success."
	got := extractReturn(desc)
	require.Equal(t, spec.KindPrimitive, got.Kind)
	require.Equal(t, "bool", got.Name)
}

func TestExtractReturn_OnSuccessTrueIsReturned(t *testing.T) {
	desc := "On success, true is returned."
	got := extractReturn(desc)
	require.Equal(t, spec.KindPrimitive, got.Kind)
	require.Equal(t, "bool", got.Name)
}

func TestExtractReturn_NamedObject(t *testing.T) {
	desc := "On success, returns a Message object."
	got := extractReturn(desc)
	require.Equal(t, spec.KindNamed, got.Kind)
	require.Equal(t, "Message", got.Name)
}

func TestExtractReturn_MessageOrBool(t *testing.T) {
	desc := "On success, the edited Message is returned, otherwise True is returned."
	got := extractReturn(desc)
	require.Equal(t, spec.KindNamed, got.Kind)
	require.Equal(t, "MessageOrBool", got.Name)
}

func TestExtractReturn_InFormOf(t *testing.T) {
	desc := "The answer is provided in form of a ChatInviteLink object."
	got := extractReturn(desc)
	require.Equal(t, spec.KindNamed, got.Kind)
	require.Equal(t, "ChatInviteLink", got.Name)
}

func TestExtractReturn_Fallback(t *testing.T) {
	// No recognized pattern → bool fallback.
	got := extractReturn("This method does something interesting.")
	require.Equal(t, spec.KindPrimitive, got.Kind)
	require.Equal(t, "bool", got.Name)
}

func TestExtractReturn_MultipleReturnsFirstWins(t *testing.T) {
	// Doc with multiple "Returns" phrases — first matching pattern should win.
	// The indefinite-article pattern ("Returns a X object") appears earlier in
	// the priority list than "Returns True", so it matches "Returns a Message"
	// before the bool pattern can fire.
	desc := "Returns True on success. You can also Returns a Message object later."
	got := extractReturn(desc)
	// The indefinite-article pattern fires first → returns Message (KindNamed).
	require.Equal(t, spec.KindNamed, got.Kind)
	require.Equal(t, "Message", got.Name)
}

// ---------------------------------------------------------------------------
// extractVersion
// ---------------------------------------------------------------------------

func TestExtractVersion_InTitle(t *testing.T) {
	sections := []section{
		{Title: "Bot API 7.3", Description: ""},
	}
	require.Equal(t, "7.3", extractVersion(sections))
}

func TestExtractVersion_InDescription(t *testing.T) {
	sections := []section{
		{Title: "April 2024", Description: "Released Bot API 7.2."},
	}
	require.Equal(t, "7.2", extractVersion(sections))
}

func TestExtractVersion_NotFound(t *testing.T) {
	sections := []section{
		{Title: "Introduction", Description: "Welcome to the API."},
	}
	require.Equal(t, "", extractVersion(sections))
}
