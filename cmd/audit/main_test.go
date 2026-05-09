package main

import (
	"testing"

	"github.com/lukaszraczylo/go-telegram/internal/spec"
	"github.com/stretchr/testify/require"
)

// ---- auditBool -----------------------------------------------------------

func TestAuditBool_FlagsUnapprovedBoolMethod(t *testing.T) {
	api := &spec.API{
		Methods: []spec.MethodDecl{
			{Name: "getMe", Doc: "A simple method.", Returns: spec.TypeRef{Kind: spec.KindPrimitive, Name: "bool"}},
		},
	}
	ov := &spec.Overrides{}
	problems := auditBool(api, ov)
	require.Len(t, problems, 1)
	require.Contains(t, problems[0], "bool fallback: getMe")
}

func TestAuditBool_SkipsApprovedBoolMethod(t *testing.T) {
	api := &spec.API{
		Methods: []spec.MethodDecl{
			{Name: "setWebhook", Doc: "Use this to set webhook.", Returns: spec.TypeRef{Kind: spec.KindPrimitive, Name: "bool"}},
		},
	}
	ov := &spec.Overrides{ApprovedBoolMethods: []string{"setWebhook"}}
	problems := auditBool(api, ov)
	require.Empty(t, problems)
}

func TestAuditBool_SkipsMethodWithReturnsTrueDoc(t *testing.T) {
	api := &spec.API{
		Methods: []spec.MethodDecl{
			{Name: "doThing", Doc: "Returns True on success.", Returns: spec.TypeRef{Kind: spec.KindPrimitive, Name: "bool"}},
		},
	}
	ov := &spec.Overrides{}
	problems := auditBool(api, ov)
	require.Empty(t, problems)
}

func TestAuditBool_SkipsNonBoolMethods(t *testing.T) {
	api := &spec.API{
		Methods: []spec.MethodDecl{
			{Name: "getMe", Doc: "Gets user.", Returns: spec.TypeRef{Kind: spec.KindNamed, Name: "User"}},
		},
	}
	ov := &spec.Overrides{}
	require.Empty(t, auditBool(api, ov))
}

// ---- auditAny ------------------------------------------------------------

func TestAuditAny_FlagsUnrecognisedOneOf(t *testing.T) {
	api := &spec.API{
		Types: []spec.TypeDecl{
			{
				Name: "Foo",
				Fields: []spec.Field{
					{Name: "Bar", JSONName: "bar", Type: spec.TypeRef{Kind: spec.KindOneOf, Variants: []string{"A", "B", "C"}}},
				},
			},
		},
	}
	out := auditAny(api)
	require.Len(t, out, 1)
	require.Contains(t, out[0], "any field: Foo.Bar")
}

func TestAuditAny_SkipsChatIDShape(t *testing.T) {
	api := &spec.API{
		Types: []spec.TypeDecl{
			{
				Name: "SendMessage",
				Fields: []spec.Field{
					{Name: "ChatID", JSONName: "chat_id", Type: spec.TypeRef{Kind: spec.KindOneOf, Variants: []string{"int64", "string"}}},
				},
			},
		},
	}
	require.Empty(t, auditAny(api))
}

func TestAuditAny_SkipsKnownUnion(t *testing.T) {
	api := &spec.API{
		Types: []spec.TypeDecl{
			{Name: "InputMedia", OneOf: []string{"InputMediaPhoto", "InputMediaVideo"}},
			{
				Name: "SomeMethod",
				Fields: []spec.Field{
					{Name: "Media", JSONName: "media", Type: spec.TypeRef{Kind: spec.KindOneOf, Variants: []string{"InputMediaPhoto", "InputMediaVideo"}}},
				},
			},
		},
	}
	require.Empty(t, auditAny(api))
}

func TestAuditAny_SkipsReplyMarkupShape(t *testing.T) {
	api := &spec.API{
		Methods: []spec.MethodDecl{
			{
				Name: "sendMessage",
				Params: []spec.Field{
					{Name: "ReplyMarkup", JSONName: "reply_markup", Type: spec.TypeRef{Kind: spec.KindOneOf, Variants: []string{"InlineKeyboardMarkup", "ReplyKeyboardMarkup", "ReplyKeyboardRemove", "ForceReply"}}},
				},
			},
		},
	}
	require.Empty(t, auditAny(api))
}

func TestAuditAny_SkipsInputFileShape(t *testing.T) {
	api := &spec.API{
		Methods: []spec.MethodDecl{
			{
				Name: "sendPhoto",
				Params: []spec.Field{
					{Name: "Photo", JSONName: "photo", Type: spec.TypeRef{Kind: spec.KindOneOf, Variants: []string{"InputFile", "string"}}},
				},
			},
		},
	}
	require.Empty(t, auditAny(api))
}

// ---- diffSignatures ------------------------------------------------------

func TestDiffSignatures_AddedMethod(t *testing.T) {
	prev := &spec.API{}
	cur := &spec.API{
		Methods: []spec.MethodDecl{
			{Name: "newMethod", Returns: spec.TypeRef{Kind: spec.KindPrimitive, Name: "bool"}},
		},
	}
	changes := diffSignatures(prev, cur)
	require.Len(t, changes, 1)
	require.Contains(t, changes[0], "added method: newMethod")
}

func TestDiffSignatures_RemovedMethod(t *testing.T) {
	prev := &spec.API{
		Methods: []spec.MethodDecl{
			{Name: "oldMethod", Returns: spec.TypeRef{Kind: spec.KindPrimitive, Name: "bool"}},
		},
	}
	cur := &spec.API{}
	changes := diffSignatures(prev, cur)
	require.Len(t, changes, 1)
	require.Contains(t, changes[0], "removed method: oldMethod")
}

func TestDiffSignatures_ChangedReturn(t *testing.T) {
	prev := &spec.API{
		Methods: []spec.MethodDecl{
			{Name: "getMe", Returns: spec.TypeRef{Kind: spec.KindPrimitive, Name: "bool"}},
		},
	}
	cur := &spec.API{
		Methods: []spec.MethodDecl{
			{Name: "getMe", Returns: spec.TypeRef{Kind: spec.KindNamed, Name: "User"}},
		},
	}
	changes := diffSignatures(prev, cur)
	require.Len(t, changes, 1)
	require.Contains(t, changes[0], "getMe")
	require.Contains(t, changes[0], "bool")
	require.Contains(t, changes[0], "User")
}

func TestDiffSignatures_Clean(t *testing.T) {
	api := &spec.API{
		Methods: []spec.MethodDecl{
			{Name: "getMe", Returns: spec.TypeRef{Kind: spec.KindNamed, Name: "User"}},
		},
	}
	require.Empty(t, diffSignatures(api, api))
}

// ---- typeRefEqual --------------------------------------------------------

func TestTypeRefEqual_Primitive(t *testing.T) {
	a := spec.TypeRef{Kind: spec.KindPrimitive, Name: "bool"}
	require.True(t, typeRefEqual(a, a))
	b := spec.TypeRef{Kind: spec.KindPrimitive, Name: "string"}
	require.False(t, typeRefEqual(a, b))
}

func TestTypeRefEqual_Array(t *testing.T) {
	elem := &spec.TypeRef{Kind: spec.KindNamed, Name: "Update"}
	a := spec.TypeRef{Kind: spec.KindArray, ElemType: elem}
	b := spec.TypeRef{Kind: spec.KindArray, ElemType: &spec.TypeRef{Kind: spec.KindNamed, Name: "Update"}}
	require.True(t, typeRefEqual(a, b))

	c := spec.TypeRef{Kind: spec.KindArray, ElemType: &spec.TypeRef{Kind: spec.KindNamed, Name: "Message"}}
	require.False(t, typeRefEqual(a, c))
}

func TestTypeRefEqual_OneOf(t *testing.T) {
	a := spec.TypeRef{Kind: spec.KindOneOf, Variants: []string{"int64", "string"}}
	b := spec.TypeRef{Kind: spec.KindOneOf, Variants: []string{"string", "int64"}}
	require.True(t, typeRefEqual(a, b))

	c := spec.TypeRef{Kind: spec.KindOneOf, Variants: []string{"int64"}}
	require.False(t, typeRefEqual(a, c))
}

func TestTypeRefEqual_NilVsNonNilElem(t *testing.T) {
	a := spec.TypeRef{Kind: spec.KindArray}
	b := spec.TypeRef{Kind: spec.KindArray, ElemType: &spec.TypeRef{Kind: spec.KindNamed, Name: "Update"}}
	require.False(t, typeRefEqual(a, b))
}
