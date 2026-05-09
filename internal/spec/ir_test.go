package spec

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAPIRoundTripJSON(t *testing.T) {
	in := API{
		Version: "7.10",
		Types: []TypeDecl{{
			Name: "User",
			Doc:  "This object represents a Telegram user or bot.",
			Fields: []Field{
				{Name: "ID", JSONName: "id", Type: TypeRef{Kind: KindPrimitive, Name: "int64"}, Required: true, Doc: "Unique identifier."},
				{Name: "Username", JSONName: "username", Type: TypeRef{Kind: KindPrimitive, Name: "string"}, Required: false, Doc: "Optional username."},
			},
		}},
		Methods: []MethodDecl{{
			Name:    "getMe",
			Doc:     "A simple method for testing your bot's authentication token.",
			Returns: TypeRef{Kind: KindNamed, Name: "User"},
		}},
	}

	data, err := json.MarshalIndent(in, "", "  ")
	require.NoError(t, err)

	var out API
	require.NoError(t, json.Unmarshal(data, &out))
	require.Equal(t, in, out)
}

func TestTypeRefKindString(t *testing.T) {
	require.Equal(t, "primitive", KindPrimitive.String())
	require.Equal(t, "named", KindNamed.String())
	require.Equal(t, "array", KindArray.String())
	require.Equal(t, "oneOf", KindOneOf.String())
}

func TestAPIRoundTrip_ArrayAndOneOf(t *testing.T) {
	elem := &TypeRef{Kind: KindNamed, Name: "Update"}
	in := API{
		Version: "7.10",
		Types: []TypeDecl{{
			Name:  "InputMedia",
			OneOf: []string{"InputMediaPhoto", "InputMediaVideo"},
		}},
		Methods: []MethodDecl{{
			Name:    "getUpdates",
			Params:  []Field{{Name: "Limit", JSONName: "limit", Type: TypeRef{Kind: KindPrimitive, Name: "int"}}},
			Returns: TypeRef{Kind: KindArray, ElemType: elem},
		}},
	}
	data, err := json.Marshal(in)
	require.NoError(t, err)
	var out API
	require.NoError(t, json.Unmarshal(data, &out))
	require.Equal(t, in, out)
}

func TestKind_MarshalUnmarshalText(t *testing.T) {
	cases := []Kind{KindPrimitive, KindNamed, KindArray, KindOneOf}
	for _, k := range cases {
		b, err := k.MarshalText()
		require.NoError(t, err)
		var out Kind
		require.NoError(t, out.UnmarshalText(b))
		require.Equal(t, k, out)
	}
}

func TestKind_UnmarshalText_UnknownReturnsError(t *testing.T) {
	var k Kind
	err := k.UnmarshalText([]byte("bogus"))
	require.Error(t, err)
}

func TestField_OmitsOptional(t *testing.T) {
	f := Field{Name: "X", JSONName: "x", Type: TypeRef{Kind: KindPrimitive, Name: "string"}}
	data, err := json.Marshal(f)
	require.NoError(t, err)
	require.NotContains(t, string(data), "required")
	require.NotContains(t, string(data), "doc")
}
