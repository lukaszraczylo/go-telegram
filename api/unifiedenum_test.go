package api

import (
	"reflect"
	"testing"

	json "github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
)

// TestUnifiedEnum_ChatMemberStatus_HasAllConstants asserts the unified
// enum exists with the full set of variant values and is a typed string.
func TestUnifiedEnum_ChatMemberStatus_HasAllConstants(t *testing.T) {
	require.IsType(t, ChatMemberStatus(""), ChatMemberStatusCreator)

	values := []ChatMemberStatus{
		ChatMemberStatusCreator,
		ChatMemberStatusAdministrator,
		ChatMemberStatusMember,
		ChatMemberStatusRestricted,
		ChatMemberStatusLeft,
		ChatMemberStatusKicked,
	}
	wantWire := []string{"creator", "administrator", "member", "restricted", "left", "kicked"}
	require.Len(t, values, 6)
	for i, v := range values {
		require.Equal(t, wantWire[i], string(v))
	}
}

// TestUnifiedEnum_ChatMember_VariantFieldsRetyped confirms every concrete
// variant's discriminator field is the unified enum, NOT a per-variant
// alias type. Reflection walks the struct field directly.
func TestUnifiedEnum_ChatMember_VariantFieldsRetyped(t *testing.T) {
	cases := []struct {
		name string
		val  any
	}{
		{"ChatMemberOwner", &ChatMemberOwner{}},
		{"ChatMemberAdministrator", &ChatMemberAdministrator{}},
		{"ChatMemberMember", &ChatMemberMember{}},
		{"ChatMemberRestricted", &ChatMemberRestricted{}},
		{"ChatMemberLeft", &ChatMemberLeft{}},
		{"ChatMemberBanned", &ChatMemberBanned{}},
	}
	wantType := reflect.TypeOf(ChatMemberStatus(""))
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f, ok := reflect.TypeOf(tc.val).Elem().FieldByName("Status")
			require.True(t, ok, "%s missing Status field", tc.name)
			require.Equal(t, wantType, f.Type, "%s.Status type mismatch", tc.name)
		})
	}
}

// TestUnifiedEnum_ChatMember_DirectComparison verifies the unified enum
// lets callers compare a variant's Status directly against constants
// without conversion.
func TestUnifiedEnum_ChatMember_DirectComparison(t *testing.T) {
	owner := &ChatMemberOwner{Status: ChatMemberStatusCreator}
	require.True(t, owner.Status == ChatMemberStatusCreator)
	require.False(t, owner.Status == ChatMemberStatusKicked)

	banned := &ChatMemberBanned{Status: ChatMemberStatusKicked}
	require.True(t, banned.Status == ChatMemberStatusKicked)
}

// TestUnifiedEnum_ChatMember_MarshalDiscriminator verifies the auto-inject
// MarshalJSON still emits the right wire discriminator after the enum
// retype — no regression from commit 370c9c0.
func TestUnifiedEnum_ChatMember_MarshalDiscriminator(t *testing.T) {
	cases := []struct {
		val      any
		wantWire string
	}{
		{&ChatMemberOwner{User: User{ID: 1, FirstName: "a"}}, "creator"},
		{&ChatMemberAdministrator{User: User{ID: 2, FirstName: "b"}}, "administrator"},
		{&ChatMemberMember{User: User{ID: 3, FirstName: "c"}}, "member"},
		{&ChatMemberRestricted{User: User{ID: 4, FirstName: "d"}}, "restricted"},
		{&ChatMemberLeft{User: User{ID: 5, FirstName: "e"}}, "left"},
		{&ChatMemberBanned{User: User{ID: 6, FirstName: "f"}}, "kicked"},
	}
	for _, tc := range cases {
		raw, err := json.Marshal(tc.val)
		require.NoError(t, err)

		var probe struct {
			Status string `json:"status"`
		}
		require.NoError(t, json.Unmarshal(raw, &probe))
		require.Equal(t, tc.wantWire, probe.Status)
	}
}

// TestUnifiedEnum_ChatMember_RoundTrip confirms a marshal-unmarshal cycle
// preserves the unified-enum field value.
func TestUnifiedEnum_ChatMember_RoundTrip(t *testing.T) {
	orig := &ChatMemberOwner{User: User{ID: 99, FirstName: "owner"}}
	raw, err := json.Marshal(orig)
	require.NoError(t, err)

	out, err := UnmarshalChatMember(raw)
	require.NoError(t, err)

	round, ok := out.(*ChatMemberOwner)
	require.True(t, ok, "expected *ChatMemberOwner, got %T", out)
	require.Equal(t, ChatMemberStatusCreator, round.Status)
	require.Equal(t, orig.User.ID, round.User.ID)
}

// TestUnifiedEnum_MessageOriginType verifies a second union also unifies
// correctly — guards against a one-off implementation that only handles
// ChatMember.
func TestUnifiedEnum_MessageOriginType(t *testing.T) {
	require.IsType(t, MessageOriginType(""), MessageOriginTypeUser)

	values := []MessageOriginType{
		MessageOriginTypeUser,
		MessageOriginTypeHiddenUser,
		MessageOriginTypeChat,
		MessageOriginTypeChannel,
	}
	wantWire := []string{"user", "hidden_user", "chat", "channel"}
	for i, v := range values {
		require.Equal(t, wantWire[i], string(v))
	}

	// Variant fields use the unified type.
	wantType := reflect.TypeOf(MessageOriginType(""))
	for _, name := range []string{"MessageOriginUser", "MessageOriginHiddenUser", "MessageOriginChat", "MessageOriginChannel"} {
		switch name {
		case "MessageOriginUser":
			f, ok := reflect.TypeOf(&MessageOriginUser{}).Elem().FieldByName("Type")
			require.True(t, ok)
			require.Equal(t, wantType, f.Type)
		case "MessageOriginHiddenUser":
			f, ok := reflect.TypeOf(&MessageOriginHiddenUser{}).Elem().FieldByName("Type")
			require.True(t, ok)
			require.Equal(t, wantType, f.Type)
		case "MessageOriginChat":
			f, ok := reflect.TypeOf(&MessageOriginChat{}).Elem().FieldByName("Type")
			require.True(t, ok)
			require.Equal(t, wantType, f.Type)
		case "MessageOriginChannel":
			f, ok := reflect.TypeOf(&MessageOriginChannel{}).Elem().FieldByName("Type")
			require.True(t, ok)
			require.Equal(t, wantType, f.Type)
		}
	}
}

// TestUnifiedEnum_StutterSuffix_Kind covers the naming-collision rule:
// when the union name ends in a discriminator concept noun, the unified
// enum is suffixed with "Kind" to avoid stuttery names like
// "BackgroundTypeType".
func TestUnifiedEnum_StutterSuffix_Kind(t *testing.T) {
	require.IsType(t, BackgroundTypeKind(""), BackgroundTypeKindFill)
	require.IsType(t, ReactionTypeKind(""), ReactionTypeKindEmoji)
	require.IsType(t, StoryAreaTypeKind(""), StoryAreaTypeKindLocation)
	require.IsType(t, ChatBoostSourceKind(""), ChatBoostSourceKindPremium)
	require.IsType(t, RevenueWithdrawalStateKind(""), RevenueWithdrawalStateKindPending)

	// Variant struct field types match the unified enum.
	wantType := reflect.TypeOf(BackgroundTypeKind(""))
	f, ok := reflect.TypeOf(&BackgroundTypeFill{}).Elem().FieldByName("Type")
	require.True(t, ok)
	require.Equal(t, wantType, f.Type)
}

// TestUnifiedEnum_PerVariantTypesNotEmitted asserts the obsolete
// per-variant single-value enum types (e.g. ChatMemberOwnerStatus) are
// gone — ensures the codegen doesn't double-emit. We rely on compile-time
// behaviour: if any of these names existed, a referencing package would
// fail to build. Instead we verify the variant struct field type's name
// is the unified one.
func TestUnifiedEnum_PerVariantTypesNotEmitted(t *testing.T) {
	got := reflect.TypeOf(&ChatMemberOwner{}).Elem()
	statusField, ok := got.FieldByName("Status")
	require.True(t, ok)
	require.Equal(t, "ChatMemberStatus", statusField.Type.Name(),
		"ChatMemberOwner.Status should be ChatMemberStatus, not ChatMemberOwnerStatus")
}
