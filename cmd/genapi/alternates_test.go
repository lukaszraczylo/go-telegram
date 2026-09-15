package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lukaszraczylo/go-telegram/internal/spec"
	"github.com/stretchr/testify/require"
)

// TestUnionAlternates_NoAlternates confirms a union with no Alternates
// (the common case — every union except RichText) produces no synthetic
// variants.
func TestUnionAlternates_NoAlternates(t *testing.T) {
	td := spec.TypeDecl{Name: "ChatMember", OneOf: []string{"ChatMemberOwner"}}
	require.Nil(t, unionAlternates(td))
}

// TestUnionAlternates_StringAndArray confirms both alternate shapes map
// to the expected synthetic variant names, in declaration order.
func TestUnionAlternates_StringAndArray(t *testing.T) {
	td := spec.TypeDecl{
		Name:  "RichText",
		OneOf: []string{"RichTextBold"},
		Alternates: []spec.Alternate{
			{Shape: spec.AlternateString},
			{Shape: spec.AlternateArray},
		},
	}
	got := unionAlternates(td)
	require.Equal(t, []alternateVariant{
		{Name: "RichTextPlain", IsString: true},
		{Name: "RichTextSequence", IsArray: true},
	}, got)
}

// TestEmitTypes_UnionWithAlternates_EmitsShapeDispatcher builds a minimal
// synthetic API with one discriminated union carrying Alternates (mirrors
// RichText's shape without depending on the full real spec) and confirms
// genapi emits: the two synthetic variant structs, their marker methods
// and MarshalJSON, and a shape-dispatching Unmarshal<Union> that branches
// on the first non-whitespace byte before falling back to the existing
// discriminator switch.
func TestEmitTypes_UnionWithAlternates_EmitsShapeDispatcher(t *testing.T) {
	api := &spec.API{
		Types: []spec.TypeDecl{
			{
				Name:  "Widget",
				OneOf: []string{"WidgetBold"},
				Alternates: []spec.Alternate{
					{Shape: spec.AlternateString},
					{Shape: spec.AlternateArray},
				},
			},
			{
				Name: "WidgetBold",
				Fields: []spec.Field{
					{Name: "Type", JSONName: "type", Required: true,
						Type: spec.TypeRef{Kind: spec.KindPrimitive, Name: "string"}, EnumValues: []string{"bold"}},
				},
			},
		},
	}

	tmp := t.TempDir()
	e := newEmitter(api, tmp)
	require.NoError(t, e.emitTypes())

	got, err := os.ReadFile(filepath.Join(tmp, "types.gen.go"))
	require.NoError(t, err)
	src := string(got)

	// Synthetic variants exist and implement the sealed interface.
	require.Contains(t, src, "type WidgetPlain struct")
	require.Contains(t, src, "Text string")
	require.Contains(t, src, "func (*WidgetPlain) isWidget() {}")
	require.Contains(t, src, "type WidgetSequence struct")
	require.Contains(t, src, "Items []Widget")
	require.Contains(t, src, "func (*WidgetSequence) isWidget() {}")

	// Round-trip MarshalJSON for both.
	require.Contains(t, src, "func (v *WidgetPlain) MarshalJSON() ([]byte, error) {")
	require.Contains(t, src, "return json.Marshal(v.Text)")
	require.Contains(t, src, "func (v *WidgetSequence) MarshalJSON() ([]byte, error) {")
	require.Contains(t, src, "return json.Marshal(v.Items)")

	// Shape-dispatching UnmarshalWidget: trims, branches on '"', '[', '{'.
	require.Contains(t, src, "func UnmarshalWidget(data []byte) (Widget, error) {")
	require.Contains(t, src, `trimmed := bytes.TrimLeft(data, " \t\r\n")`)
	require.Contains(t, src, "case '\"':")
	require.Contains(t, src, "return &WidgetPlain{Text: s}, nil")
	require.Contains(t, src, "case '[':")
	require.Contains(t, src, "return &WidgetSequence{Items: items}, nil")
	require.Contains(t, src, "case '{':")
	// The pre-existing discriminator switch is preserved beneath the
	// shape dispatch, unrecognised discriminator still a clear error.
	require.Contains(t, src, `fmt.Errorf("Widget: unknown type %q", probe.V)`)

	// The struct-holding-the-union case inherits shape dispatch for free:
	// WidgetBold has no union-typed field itself, so nothing further to
	// assert there beyond successful compilation-shaped source (covered
	// by TestEmit_Types_FixtureGolden's gofmt round trip for the real
	// fixture, and by the api-package tests for full decode behaviour).
	require.False(t, strings.Contains(src, "WidgetPlainSequence"), "sanity: no accidental name collision")
}

// TestEmitTypes_UnionWithoutAlternates_Unaffected confirms a discriminated
// union with no Alternates emits the original (pre-alternates) shape:
// no synthetic variants, no shape dispatch, and the same "decodes a X
// from JSON by inspecting" doc comment wording as before this feature.
func TestEmitTypes_UnionWithoutAlternates_Unaffected(t *testing.T) {
	api := &spec.API{
		Types: []spec.TypeDecl{
			{Name: "Widget", OneOf: []string{"WidgetBold"}},
			{
				Name: "WidgetBold",
				Fields: []spec.Field{
					{Name: "Type", JSONName: "type", Required: true,
						Type: spec.TypeRef{Kind: spec.KindPrimitive, Name: "string"}, EnumValues: []string{"bold"}},
				},
			},
		},
	}

	tmp := t.TempDir()
	e := newEmitter(api, tmp)
	require.NoError(t, e.emitTypes())

	got, err := os.ReadFile(filepath.Join(tmp, "types.gen.go"))
	require.NoError(t, err)
	src := string(got)

	require.NotContains(t, src, "WidgetPlain")
	require.NotContains(t, src, "WidgetSequence")
	// The package-level `var _ = bytes.TrimLeft` import guard is always
	// present (it keeps the "bytes" import valid even when no union in
	// this API has alternates); what must NOT appear is the per-union
	// shape-dispatch call site.
	require.NotContains(t, src, "trimmed := bytes.TrimLeft(data,")
	require.Contains(t, src, "// UnmarshalWidget decodes a Widget from JSON by inspecting the")
	require.Contains(t, src, `// "type" field and dispatching to the correct concrete type.`)
}
