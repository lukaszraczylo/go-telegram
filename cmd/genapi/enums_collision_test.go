package main

import (
	"testing"

	"github.com/lukaszraczylo/go-telegram/internal/spec"
	"github.com/stretchr/testify/require"
)

// Regression: Bot API 9.x RichBlockListItem.type enumerates "a"/"A"/"i"/"I"/"1",
// which Pascal-case to the same identifier. ConstName must disambiguate.
func TestEnumDecl_ConstName_CaseCollision(t *testing.T) {
	d := enumDecl{Name: "RichBlockListItemType", Values: []string{"a", "A", "i", "I", "1"}}
	got := map[string]bool{}
	for _, v := range d.Values {
		n := d.ConstName(v)
		require.False(t, got[n], "duplicate const ident %q for value %q", n, v)
		got[n] = true
	}
	require.Equal(t, "RichBlockListItemTypeLowerA", d.ConstName("a"))
	require.Equal(t, "RichBlockListItemTypeUpperA", d.ConstName("A"))
	// Non-colliding values keep the plain name.
	require.Equal(t, "RichBlockListItemType1", d.ConstName("1"))
}

// Regression: answerChatJoinRequestQuery has an enum param named "result";
// answerGuestQuery has a NON-enum param also named "result". The enum plan
// must scope method params per method so the enum never leaks onto the
// other method's field.
func TestPlanEnums_MethodParamsScopedPerMethod(t *testing.T) {
	api := &spec.API{
		Methods: []spec.MethodDecl{
			{
				Name: "answerChatJoinRequestQuery",
				Params: []spec.Field{{
					Name: "Result", JSONName: "result", Required: true,
					Type:       spec.TypeRef{Kind: spec.KindPrimitive, Name: "string"},
					EnumValues: []string{"approve", "decline", "queue"},
				}},
			},
			{
				Name: "answerGuestQuery",
				Params: []spec.Field{{
					Name: "Result", JSONName: "result", Required: true,
					Type: spec.TypeRef{Kind: spec.KindNamed, Name: "InlineQueryResult"},
				}},
			},
		},
	}
	plan := planEnums(api)
	require.Equal(t, "Result",
		plan.FieldEnum(methodEnumParent("answerChatJoinRequestQuery"), "Result"))
	require.Empty(t,
		plan.FieldEnum(methodEnumParent("answerGuestQuery"), "Result"),
		"non-enum param must not inherit another method's enum")
}
