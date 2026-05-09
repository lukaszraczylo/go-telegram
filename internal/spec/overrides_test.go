package spec

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadOverrides_MissingFile(t *testing.T) {
	o, err := LoadOverrides(filepath.Join(t.TempDir(), "nonexistent.json"))
	require.NoError(t, err)
	require.NotNil(t, o)
	require.Empty(t, o.MethodReturns)
	require.Empty(t, o.FieldTypes)
	require.Empty(t, o.ApprovedBoolMethods)
}

func TestLoadOverrides_MalformedJSON(t *testing.T) {
	p := filepath.Join(t.TempDir(), "bad.json")
	require.NoError(t, os.WriteFile(p, []byte("{bad json"), 0o600))
	_, err := LoadOverrides(p)
	require.Error(t, err)
}

func TestApply_PatchesMethodReturn(t *testing.T) {
	api := &API{
		Methods: []MethodDecl{
			{Name: "getMe", Returns: TypeRef{Kind: KindPrimitive, Name: "bool"}},
		},
	}
	o := &Overrides{
		MethodReturns: map[string]TypeRef{
			"getMe": {Kind: KindNamed, Name: "User"},
		},
	}
	o.Apply(api)
	require.Equal(t, KindNamed, api.Methods[0].Returns.Kind)
	require.Equal(t, "User", api.Methods[0].Returns.Name)
}

func TestApply_PatchesFieldType(t *testing.T) {
	api := &API{
		Types: []TypeDecl{
			{
				Name: "Message",
				Fields: []Field{
					{Name: "ChatID", JSONName: "chat_id", Type: TypeRef{Kind: KindPrimitive, Name: "string"}},
				},
			},
		},
	}
	o := &Overrides{
		FieldTypes: map[string]TypeRef{
			"Message.ChatID": {Kind: KindOneOf, Variants: []string{"int64", "string"}},
		},
	}
	o.Apply(api)
	require.Equal(t, KindOneOf, api.Types[0].Fields[0].Type.Kind)
	require.Equal(t, []string{"int64", "string"}, api.Types[0].Fields[0].Type.Variants)
}

func TestApply_NilOverrides(t *testing.T) {
	api := &API{
		Methods: []MethodDecl{{Name: "getMe", Returns: TypeRef{Kind: KindPrimitive, Name: "bool"}}},
	}
	var o *Overrides
	require.NotPanics(t, func() { o.Apply(api) })
	require.Equal(t, "bool", api.Methods[0].Returns.Name)
}

func TestIsBoolApproved_Hit(t *testing.T) {
	o := &Overrides{ApprovedBoolMethods: []string{"setWebhook", "deleteWebhook"}}
	require.True(t, o.IsBoolApproved("setWebhook"))
	require.True(t, o.IsBoolApproved("deleteWebhook"))
}

func TestIsBoolApproved_Miss(t *testing.T) {
	o := &Overrides{ApprovedBoolMethods: []string{"setWebhook"}}
	require.False(t, o.IsBoolApproved("getMe"))
}

func TestIsBoolApproved_NilOverrides(t *testing.T) {
	var o *Overrides
	require.False(t, o.IsBoolApproved("anything"))
}
