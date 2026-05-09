package main

import (
	"testing"

	"github.com/lukaszraczylo/go-telegram/internal/spec"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// goType — all branches
// ---------------------------------------------------------------------------

func TestGoType_Primitive(t *testing.T) {
	cases := []struct {
		name     string
		optional bool
		want     string
	}{
		{"bool", false, "bool"},
		{"bool", true, "*bool"},
		{"int64", false, "int64"},
		{"int64", true, "*int64"},
		{"float64", false, "float64"},
		{"float64", true, "*float64"},
		{"string", false, "string"},
		{"string", true, "string"}, // string is not pointer-wrapped
	}
	for _, c := range cases {
		tr := spec.TypeRef{Kind: spec.KindPrimitive, Name: c.name}
		got := goType(tr, c.optional)
		require.Equal(t, c.want, got, "goType(%q, optional=%v)", c.name, c.optional)
	}
}

func TestGoType_Named_Required(t *testing.T) {
	tr := spec.TypeRef{Kind: spec.KindNamed, Name: "Message"}
	require.Equal(t, "Message", goType(tr, false))
	require.Equal(t, "*Message", goType(tr, true))
}

func TestGoType_Named_InputFile(t *testing.T) {
	tr := spec.TypeRef{Kind: spec.KindNamed, Name: "InputFile"}
	// InputFile is always pointer even when required.
	require.Equal(t, "*InputFile", goType(tr, false))
	require.Equal(t, "*InputFile", goType(tr, true))
}

func TestGoType_Named_UnionInterface(t *testing.T) {
	// ChatMember is a known discriminated union — no * even when optional.
	tr := spec.TypeRef{Kind: spec.KindNamed, Name: "ChatMember"}
	require.Equal(t, "ChatMember", goType(tr, false))
	require.Equal(t, "ChatMember", goType(tr, true))
}

func TestGoType_Array_NilElem(t *testing.T) {
	tr := spec.TypeRef{Kind: spec.KindArray}
	require.Equal(t, "[]any", goType(tr, false))
}

func TestGoType_Array_WithElem(t *testing.T) {
	elem := spec.TypeRef{Kind: spec.KindNamed, Name: "Update"}
	tr := spec.TypeRef{Kind: spec.KindArray, ElemType: &elem}
	require.Equal(t, "[]Update", goType(tr, false))
}

func TestGoType_OneOf_ChatID(t *testing.T) {
	tr := spec.TypeRef{Kind: spec.KindOneOf, Variants: []string{"int64", "string"}}
	require.Equal(t, "ChatID", goType(tr, false))
	require.Equal(t, "*ChatID", goType(tr, true))
}

func TestGoType_OneOf_InputFile(t *testing.T) {
	tr := spec.TypeRef{Kind: spec.KindOneOf, Variants: []string{"InputFile", "string"}}
	require.Equal(t, "*InputFile", goType(tr, false))
}

func TestGoType_OneOf_SealedInterface(t *testing.T) {
	tr := spec.TypeRef{Kind: spec.KindOneOf, Variants: []string{"A", "B"}}
	require.Equal(t, "any", goType(tr, false))
}

func TestGoType_Unknown(t *testing.T) {
	tr := spec.TypeRef{Kind: spec.Kind(99)}
	require.Equal(t, "any", goType(tr, false))
}

// ---------------------------------------------------------------------------
// returnGoType — all branches
// ---------------------------------------------------------------------------

func TestReturnGoType_Primitive(t *testing.T) {
	tr := spec.TypeRef{Kind: spec.KindPrimitive, Name: "bool"}
	require.Equal(t, "bool", returnGoType(tr))
}

func TestReturnGoType_Named(t *testing.T) {
	tr := spec.TypeRef{Kind: spec.KindNamed, Name: "Message"}
	require.Equal(t, "*Message", returnGoType(tr))
}

func TestReturnGoType_Array_NilElem(t *testing.T) {
	tr := spec.TypeRef{Kind: spec.KindArray}
	require.Equal(t, "[]any", returnGoType(tr))
}

func TestReturnGoType_Array_WithElem(t *testing.T) {
	elem := spec.TypeRef{Kind: spec.KindNamed, Name: "Update"}
	tr := spec.TypeRef{Kind: spec.KindArray, ElemType: &elem}
	require.Equal(t, "[]Update", returnGoType(tr))
}

func TestReturnGoType_OneOf_ChatID(t *testing.T) {
	tr := spec.TypeRef{Kind: spec.KindOneOf, Variants: []string{"int64", "string"}}
	require.Equal(t, "ChatID", returnGoType(tr))
}

func TestReturnGoType_OneOf_Other(t *testing.T) {
	tr := spec.TypeRef{Kind: spec.KindOneOf, Variants: []string{"A", "B"}}
	require.Equal(t, "any", returnGoType(tr))
}

func TestReturnGoType_Unknown(t *testing.T) {
	tr := spec.TypeRef{Kind: spec.Kind(99)}
	require.Equal(t, "any", returnGoType(tr))
}

// ---------------------------------------------------------------------------
// returnGoElem — all branches
// ---------------------------------------------------------------------------

func TestReturnGoElem_Primitive(t *testing.T) {
	require.Equal(t, "int64", returnGoElem(spec.TypeRef{Kind: spec.KindPrimitive, Name: "int64"}))
}

func TestReturnGoElem_Named(t *testing.T) {
	require.Equal(t, "Message", returnGoElem(spec.TypeRef{Kind: spec.KindNamed, Name: "Message"}))
}

func TestReturnGoElem_Array_NilElem(t *testing.T) {
	require.Equal(t, "any", returnGoElem(spec.TypeRef{Kind: spec.KindArray}))
}

func TestReturnGoElem_Array_WithElem(t *testing.T) {
	elem := spec.TypeRef{Kind: spec.KindNamed, Name: "PhotoSize"}
	tr := spec.TypeRef{Kind: spec.KindArray, ElemType: &elem}
	require.Equal(t, "[]PhotoSize", returnGoElem(tr))
}

func TestReturnGoElem_Unknown(t *testing.T) {
	require.Equal(t, "any", returnGoElem(spec.TypeRef{Kind: spec.Kind(99)}))
}

// ---------------------------------------------------------------------------
// multipartFieldEntry — all branches
// ---------------------------------------------------------------------------

func makeField(name, jname, typName string, kind spec.Kind, required bool) spec.Field {
	return spec.Field{
		Name:     name,
		JSONName: jname,
		Type:     spec.TypeRef{Kind: kind, Name: typName},
		Required: required,
	}
}

func makeFieldVariants(name, jname string, kind spec.Kind, variants []string, required bool) spec.Field {
	return spec.Field{
		Name:     name,
		JSONName: jname,
		Type:     spec.TypeRef{Kind: kind, Variants: variants},
		Required: required,
	}
}

func TestMultipartFieldEntry_Int64Required(t *testing.T) {
	f := makeField("ChatID", "chat_id", "int64", spec.KindPrimitive, true)
	got := multipartFieldEntry(nil, "", f)
	require.Contains(t, got, `FormatInt`)
	require.NotContains(t, got, "if p.")
}

func TestMultipartFieldEntry_Int64Optional(t *testing.T) {
	f := makeField("MessageThreadID", "message_thread_id", "int64", spec.KindPrimitive, false)
	got := multipartFieldEntry(nil, "", f)
	require.Contains(t, got, `FormatInt`)
	require.Contains(t, got, "if p.")
}

func TestMultipartFieldEntry_StringRequired(t *testing.T) {
	f := makeField("Text", "text", "string", spec.KindPrimitive, true)
	got := multipartFieldEntry(nil, "", f)
	require.Contains(t, got, `out["text"]`)
	require.NotContains(t, got, "if p.Text")
}

func TestMultipartFieldEntry_StringOptional(t *testing.T) {
	f := makeField("ParseMode", "parse_mode", "string", spec.KindPrimitive, false)
	got := multipartFieldEntry(nil, "", f)
	require.Contains(t, got, `if p.ParseMode`)
}

func TestMultipartFieldEntry_BoolRequired(t *testing.T) {
	f := makeField("DisableNotification", "disable_notification", "bool", spec.KindPrimitive, true)
	got := multipartFieldEntry(nil, "", f)
	require.Contains(t, got, `FormatBool`)
	require.NotContains(t, got, "if p.")
}

func TestMultipartFieldEntry_BoolOptional(t *testing.T) {
	f := makeField("Protected", "protect_content", "bool", spec.KindPrimitive, false)
	got := multipartFieldEntry(nil, "", f)
	require.Contains(t, got, `FormatBool`)
	require.Contains(t, got, "if p.")
}

func TestMultipartFieldEntry_Float64Required(t *testing.T) {
	f := makeField("Latitude", "latitude", "float64", spec.KindPrimitive, true)
	got := multipartFieldEntry(nil, "", f)
	require.Contains(t, got, `FormatFloat`)
	require.NotContains(t, got, "if p.")
}

func TestMultipartFieldEntry_Float64Optional(t *testing.T) {
	f := makeField("Longitude", "longitude", "float64", spec.KindPrimitive, false)
	got := multipartFieldEntry(nil, "", f)
	require.Contains(t, got, `FormatFloat`)
	require.Contains(t, got, "if p.")
}

func TestMultipartFieldEntry_OneOf_ChatIDRequired(t *testing.T) {
	f := makeFieldVariants("ChatID", "chat_id", spec.KindOneOf, []string{"int64", "string"}, true)
	got := multipartFieldEntry(nil, "", f)
	require.Contains(t, got, `.String()`)
	require.NotContains(t, got, "IsZero")
}

func TestMultipartFieldEntry_OneOf_ChatIDOptional(t *testing.T) {
	f := makeFieldVariants("ChatID", "chat_id", spec.KindOneOf, []string{"int64", "string"}, false)
	got := multipartFieldEntry(nil, "", f)
	require.Contains(t, got, `IsZero`)
}

func TestMultipartFieldEntry_OneOf_InputFileOrString(t *testing.T) {
	f := makeFieldVariants("Photo", "photo", spec.KindOneOf, []string{"InputFile", "string"}, false)
	got := multipartFieldEntry(nil, "", f)
	require.Contains(t, got, `PathOrID`)
}

func TestMultipartFieldEntry_OneOf_SealedRequired(t *testing.T) {
	f := makeFieldVariants("Markup", "reply_markup", spec.KindOneOf, []string{"A", "B"}, true)
	got := multipartFieldEntry(nil, "", f)
	require.Contains(t, got, `json.Marshal`)
}

func TestMultipartFieldEntry_OneOf_SealedOptional(t *testing.T) {
	f := makeFieldVariants("Markup", "reply_markup", spec.KindOneOf, []string{"A", "B"}, false)
	got := multipartFieldEntry(nil, "", f)
	require.Contains(t, got, `json.Marshal`)
	require.Contains(t, got, "if p.Markup")
}

func TestMultipartFieldEntry_Named_Required(t *testing.T) {
	f := makeField("Entities", "entities", "MessageEntity", spec.KindNamed, true)
	got := multipartFieldEntry(nil, "", f)
	require.Contains(t, got, `json.Marshal`)
	require.NotContains(t, got, "if p.")
}

func TestMultipartFieldEntry_Named_Optional(t *testing.T) {
	f := makeField("Entities", "entities", "MessageEntity", spec.KindNamed, false)
	got := multipartFieldEntry(nil, "", f)
	require.Contains(t, got, `json.Marshal`)
	require.Contains(t, got, "if p.")
}

// ---------------------------------------------------------------------------
// unionTypeFor — all branches
// ---------------------------------------------------------------------------

func TestUnionTypeFor_DirectNamed(t *testing.T) {
	tr := spec.TypeRef{Kind: spec.KindNamed, Name: "ChatMember"}
	name, ok := unionTypeFor(tr)
	require.True(t, ok)
	require.Equal(t, "ChatMember", name)
}

func TestUnionTypeFor_Array(t *testing.T) {
	elem := spec.TypeRef{Kind: spec.KindNamed, Name: "ChatMember"}
	tr := spec.TypeRef{Kind: spec.KindArray, ElemType: &elem}
	name, ok := unionTypeFor(tr)
	require.True(t, ok)
	require.Equal(t, "ChatMember", name)
}

func TestUnionTypeFor_ArrayNilElem(t *testing.T) {
	tr := spec.TypeRef{Kind: spec.KindArray}
	_, ok := unionTypeFor(tr)
	require.False(t, ok)
}

func TestUnionTypeFor_NotUnion(t *testing.T) {
	tr := spec.TypeRef{Kind: spec.KindNamed, Name: "Message"}
	_, ok := unionTypeFor(tr)
	require.False(t, ok)
}

func TestUnionTypeFor_Unknown(t *testing.T) {
	tr := spec.TypeRef{Kind: spec.Kind(99)}
	_, ok := unionTypeFor(tr)
	require.False(t, ok)
}

// ---------------------------------------------------------------------------
// unionNameByVariants
// ---------------------------------------------------------------------------

func TestUnionNameByVariants_ChatMember(t *testing.T) {
	// Use the actual variants from knownDiscriminators["ChatMember"].
	variants := []string{
		"ChatMemberOwner", "ChatMemberAdministrator", "ChatMemberMember",
		"ChatMemberRestricted", "ChatMemberLeft", "ChatMemberBanned",
	}
	name := unionNameByVariants(variants)
	require.Equal(t, "ChatMember", name)
}

func TestUnionNameByVariants_Unknown(t *testing.T) {
	name := unionNameByVariants([]string{"X", "Y", "Z"})
	require.Equal(t, "", name)
}

// ---------------------------------------------------------------------------
// hasUnionElem
// ---------------------------------------------------------------------------

func TestHasUnionElem_NonArray(t *testing.T) {
	tr := spec.TypeRef{Kind: spec.KindNamed, Name: "ChatMember"}
	require.False(t, hasUnionElem(tr))
}

func TestHasUnionElem_ArrayNilElem(t *testing.T) {
	tr := spec.TypeRef{Kind: spec.KindArray}
	require.False(t, hasUnionElem(tr))
}

func TestHasUnionElem_ArrayUnionElem(t *testing.T) {
	elem := spec.TypeRef{Kind: spec.KindNamed, Name: "ChatMember"}
	tr := spec.TypeRef{Kind: spec.KindArray, ElemType: &elem}
	require.True(t, hasUnionElem(tr))
}

func TestHasUnionElem_ArrayNonUnionElem(t *testing.T) {
	elem := spec.TypeRef{Kind: spec.KindNamed, Name: "Message"}
	tr := spec.TypeRef{Kind: spec.KindArray, ElemType: &elem}
	require.False(t, hasUnionElem(tr))
}

// ---------------------------------------------------------------------------
// unionFieldsOf
// ---------------------------------------------------------------------------

func TestUnionFieldsOf_WithUnionField(t *testing.T) {
	td := spec.TypeDecl{
		Name: "ChatMemberUpdated",
		Fields: []spec.Field{
			{Name: "NewChatMember", JSONName: "new_chat_member", Type: spec.TypeRef{Kind: spec.KindNamed, Name: "ChatMember"}},
			{Name: "OldChatMember", JSONName: "old_chat_member", Type: spec.TypeRef{Kind: spec.KindNamed, Name: "ChatMember"}},
			{Name: "Date", JSONName: "date", Type: spec.TypeRef{Kind: spec.KindPrimitive, Name: "int64"}},
		},
	}
	uf := unionFieldsOf(td)
	require.Len(t, uf, 2)
	require.Equal(t, "ChatMember", uf[0].UnionName)
}

// ---------------------------------------------------------------------------
// splitLines — edge cases
// ---------------------------------------------------------------------------

func TestSplitLines_Empty(t *testing.T) {
	require.Empty(t, splitLines(""))
}

func TestSplitLines_NoNewline(t *testing.T) {
	got := splitLines("hello world")
	require.Equal(t, []string{"hello world"}, got)
}

func TestSplitLines_TrailingNewline(t *testing.T) {
	got := splitLines("line1\nline2\n")
	require.Equal(t, []string{"line1", "line2"}, got)
}

func TestSplitLines_MultiLine(t *testing.T) {
	got := splitLines("a\nb\nc")
	require.Equal(t, []string{"a", "b", "c"}, got)
}

// ---------------------------------------------------------------------------
// docComment
// ---------------------------------------------------------------------------

func TestDocComment_Empty(t *testing.T) {
	require.Equal(t, "", docComment(""))
}

func TestDocComment_SingleLine(t *testing.T) {
	got := docComment("Hello world.")
	require.Equal(t, "// Hello world.\n", got)
}

func TestDocComment_MultiLine(t *testing.T) {
	got := docComment("Line 1\nLine 2")
	require.Contains(t, got, "// Line 1\n")
	require.Contains(t, got, "// Line 2\n")
}

// ---------------------------------------------------------------------------
// title
// ---------------------------------------------------------------------------

func TestTitle_Empty(t *testing.T) {
	require.Equal(t, "", title(""))
}

func TestTitle_Lowercase(t *testing.T) {
	require.Equal(t, "SendMessage", title("sendMessage"))
}

func TestTitle_AlreadyUpper(t *testing.T) {
	require.Equal(t, "GetMe", title("GetMe"))
}

// ---------------------------------------------------------------------------
// mentionsInputFileTr
// ---------------------------------------------------------------------------

func TestMentionsInputFileTr_Named(t *testing.T) {
	tr := spec.TypeRef{Kind: spec.KindNamed, Name: "InputFile"}
	require.True(t, mentionsInputFileTr(tr))
}

func TestMentionsInputFileTr_NotInputFile(t *testing.T) {
	tr := spec.TypeRef{Kind: spec.KindNamed, Name: "Message"}
	require.False(t, mentionsInputFileTr(tr))
}

func TestMentionsInputFileTr_Array(t *testing.T) {
	elem := spec.TypeRef{Kind: spec.KindNamed, Name: "InputFile"}
	tr := spec.TypeRef{Kind: spec.KindArray, ElemType: &elem}
	require.True(t, mentionsInputFileTr(tr))
}

func TestMentionsInputFileTr_ArrayNilElem(t *testing.T) {
	tr := spec.TypeRef{Kind: spec.KindArray}
	require.False(t, mentionsInputFileTr(tr))
}

func TestMentionsInputFileTr_OneOf_WithInputFile(t *testing.T) {
	tr := spec.TypeRef{Kind: spec.KindOneOf, Variants: []string{"InputFile", "string"}}
	require.True(t, mentionsInputFileTr(tr))
}

func TestMentionsInputFileTr_OneOf_WithoutInputFile(t *testing.T) {
	tr := spec.TypeRef{Kind: spec.KindOneOf, Variants: []string{"A", "B"}}
	require.False(t, mentionsInputFileTr(tr))
}

// ---------------------------------------------------------------------------
// loadAPI — error paths
// ---------------------------------------------------------------------------

func TestLoadAPI_MissingFile(t *testing.T) {
	_, err := loadAPI("/nonexistent/path/api.json")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// runtimeTypes filter in emitTypes
// ---------------------------------------------------------------------------

func TestRuntimeTypes_NeverEmitted(t *testing.T) {
	for name := range runtimeTypes {
		require.True(t, runtimeTypes[name], "runtimeType %q should be true", name)
	}
	require.True(t, runtimeTypes["InputFile"])
	require.True(t, runtimeTypes["ChatID"])
	require.True(t, runtimeTypes["MessageOrBool"])
	require.True(t, runtimeTypes["ResponseParameters"])
}

// ---------------------------------------------------------------------------
// sentinelForField — all branches
// ---------------------------------------------------------------------------

func TestSentinelForField(t *testing.T) {
	unionTypes := map[string]bool{"ChatMember": true}
	cases := []struct {
		name     string
		field    spec.Field
		contains string
	}{
		{
			name:     "int64 primitive",
			field:    makeField("Count", "count", "int64", spec.KindPrimitive, true),
			contains: "42",
		},
		{
			name:     "string primitive",
			field:    makeField("Text", "text", "string", spec.KindPrimitive, true),
			contains: "test_value",
		},
		{
			name:     "bool primitive",
			field:    makeField("Flag", "flag", "bool", spec.KindPrimitive, true),
			contains: "true",
		},
		{
			name:     "float64 primitive",
			field:    makeField("Lat", "lat", "float64", spec.KindPrimitive, true),
			contains: "1.0",
		},
		{
			name:     "named ChatID",
			field:    makeField("ChatID", "chat_id", "ChatID", spec.KindNamed, true),
			contains: "ChatIDFromInt",
		},
		{
			name:     "named InputFile",
			field:    makeField("Photo", "photo", "InputFile", spec.KindNamed, true),
			contains: "InputFile",
		},
		{
			name:     "named union (nil-able)",
			field:    makeField("Member", "member", "ChatMember", spec.KindNamed, true),
			contains: "nil",
		},
		{
			name:     "named required struct",
			field:    makeField("Chat", "chat", "Chat", spec.KindNamed, true),
			contains: "Chat{}",
		},
		{
			name:     "named optional struct",
			field:    makeField("Chat", "chat", "Chat", spec.KindNamed, false),
			contains: "&Chat{}",
		},
		{
			name:     "array",
			field:    spec.Field{Name: "Items", JSONName: "items", Type: spec.TypeRef{Kind: spec.KindArray}},
			contains: "nil",
		},
		{
			name:     "oneOf ChatID variants",
			field:    makeFieldVariants("ChatID", "chat_id", spec.KindOneOf, []string{"int64", "string"}, true),
			contains: "ChatIDFromInt",
		},
		{
			name:     "oneOf InputFile variants",
			field:    makeFieldVariants("Photo", "photo", spec.KindOneOf, []string{"InputFile", "string"}, true),
			contains: "InputFile",
		},
		{
			name:     "oneOf sealed",
			field:    makeFieldVariants("Markup", "markup", spec.KindOneOf, []string{"A", "B"}, true),
			contains: "nil",
		},
		{
			name:     "unknown kind",
			field:    spec.Field{Name: "X", JSONName: "x", Type: spec.TypeRef{Kind: spec.Kind(99)}},
			contains: "nil",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := sentinelForField(c.field, unionTypes, nil)
			require.Contains(t, got, c.contains, "sentinelForField for %q", c.name)
		})
	}
}

// ---------------------------------------------------------------------------
// successBody — all branches
// ---------------------------------------------------------------------------

func TestSuccessBody(t *testing.T) {
	cases := []struct {
		name string
		tr   spec.TypeRef
		want string
	}{
		{"bool", spec.TypeRef{Kind: spec.KindPrimitive, Name: "bool"}, "true"},
		{"int64", spec.TypeRef{Kind: spec.KindPrimitive, Name: "int64"}, "0"},
		{"float64", spec.TypeRef{Kind: spec.KindPrimitive, Name: "float64"}, "0"},
		{"string", spec.TypeRef{Kind: spec.KindPrimitive, Name: "string"}, `""`},
		{"MessageOrBool", spec.TypeRef{Kind: spec.KindNamed, Name: "MessageOrBool"}, "true"},
		{"named", spec.TypeRef{Kind: spec.KindNamed, Name: "Message"}, "{}"},
		{"array", spec.TypeRef{Kind: spec.KindArray}, "[]"},
		{"oneOf", spec.TypeRef{Kind: spec.KindOneOf, Variants: []string{"A", "B"}}, "null"},
		{"unknown", spec.TypeRef{Kind: spec.Kind(99)}, "null"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := successBody(c.tr)
			require.Equal(t, c.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// unionTypeFor — KindOneOf branch (variant-set match)
// ---------------------------------------------------------------------------

func TestUnionTypeFor_OneOfVariants(t *testing.T) {
	// These variant *type names* match the ChatMember discriminator.
	tr := spec.TypeRef{
		Kind: spec.KindOneOf,
		Variants: []string{
			"ChatMemberOwner", "ChatMemberAdministrator", "ChatMemberMember",
			"ChatMemberRestricted", "ChatMemberLeft", "ChatMemberBanned",
		},
	}
	name, ok := unionTypeFor(tr)
	require.True(t, ok)
	require.Equal(t, "ChatMember", name)
}

func TestUnionTypeFor_OneOfNoMatch(t *testing.T) {
	tr := spec.TypeRef{Kind: spec.KindOneOf, Variants: []string{"Foo", "Bar"}}
	_, ok := unionTypeFor(tr)
	require.False(t, ok)
}

// ---------------------------------------------------------------------------
// funcs() returns a non-nil FuncMap with expected keys
// ---------------------------------------------------------------------------

func TestFuncs_HasExpectedKeys(t *testing.T) {
	fm := funcs(nil)
	require.NotNil(t, fm)
	for _, key := range []string{"goType", "docComment", "returnGoType", "unionFields"} {
		require.NotNil(t, fm[key], "funcs() missing key %q", key)
	}
}
