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
		{"webhook_URL", "WebhookURL"}, // Issue 3: already-uppercase segment must not be corrupted.
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
