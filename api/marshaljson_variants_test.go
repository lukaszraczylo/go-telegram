package api

import (
	"testing"

	json "github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
)

// TestMarshalJSON_TypeDiscriminator_AutoInjected verifies the generated
// MarshalJSON hardcodes the wire discriminator for a Type-keyed variant
// even when the caller leaves the field zero.
func TestMarshalJSON_TypeDiscriminator_AutoInjected(t *testing.T) {
	scope := &BotCommandScopeAllPrivateChats{}
	got, err := json.Marshal(scope)
	require.NoError(t, err)
	require.JSONEq(t, `{"type":"all_private_chats"}`, string(got))
}

// TestMarshalJSON_SourceDiscriminator_AutoInjected verifies the same
// for variants that use a non-Type discriminator field. PassportElement
// errors key on "source" instead.
func TestMarshalJSON_SourceDiscriminator_AutoInjected(t *testing.T) {
	err := &PassportElementErrorDataField{
		Type:      PassportElementErrorDataFieldTypePersonalDetails,
		FieldName: "first_name",
		DataHash:  "abc123",
		Message:   "bad data",
	}
	got, mErr := json.Marshal(err)
	require.NoError(t, mErr)
	require.JSONEq(t,
		`{"source":"data","type":"personal_details","field_name":"first_name","data_hash":"abc123","message":"bad data"}`,
		string(got),
	)
}

// TestMarshalJSON_UserSuppliedDiscriminator_Overridden documents the
// safety guarantee: a typo or stale value the caller pastes into the
// struct literal is silently overridden by the generated MarshalJSON.
// This is what saves callers from Telegram's "silent reject" failure
// mode when a discriminator is wrong.
func TestMarshalJSON_UserSuppliedDiscriminator_Overridden(t *testing.T) {
	scope := &BotCommandScopeAllPrivateChats{Type: "wrong"}
	got, err := json.Marshal(scope)
	require.NoError(t, err)
	require.JSONEq(t, `{"type":"all_private_chats"}`, string(got))
}

// TestMarshalJSON_RoundTrip confirms a marshal-then-unmarshal cycle
// preserves user-supplied fields. Discriminator field is set on the
// way out, read back on the way in — no data loss.
//
// Uses ChatMember (one of the auto-decode unions) so the round-trip
// can route through the generated UnmarshalChatMember dispatcher.
func TestMarshalJSON_RoundTrip(t *testing.T) {
	orig := &ChatMemberLeft{
		User: User{ID: 42, IsBot: false, FirstName: "alice"},
	}
	raw, err := json.Marshal(orig)
	require.NoError(t, err)

	out, err := UnmarshalChatMember(raw)
	require.NoError(t, err)

	round, ok := out.(*ChatMemberLeft)
	require.True(t, ok, "expected *ChatMemberLeft, got %T", out)
	require.Equal(t, ChatMemberLeftStatusLeft, round.Status)
	require.Equal(t, orig.User.ID, round.User.ID)
	require.Equal(t, orig.User.FirstName, round.User.FirstName)
}

// TestMarshalJSON_InputMessageContent_NoDiscriminator confirms that
// variants of InputMessageContent (the structurally-dispatched union
// Telegram identifies by field presence, not by a "type" field) do
// NOT get an injected discriminator. Their fields ride out as-is.
func TestMarshalJSON_InputMessageContent_NoDiscriminator(t *testing.T) {
	content := &InputTextMessageContent{
		MessageText: "hello world",
	}
	got, err := json.Marshal(content)
	require.NoError(t, err)
	// No "type" field should appear; just message_text.
	require.JSONEq(t, `{"message_text":"hello world"}`, string(got))
}

// TestMarshalJSON_NonDiscriminatorMembers_RidealongUnchanged verifies
// the alias-embedding pattern: every non-discriminator field on the
// variant marshals through the *alias and keeps its own json tag and
// omitempty behaviour. Caption + ParseMode here exercise both
// required-string-with-discriminator and optional-with-omitempty.
func TestMarshalJSON_NonDiscriminatorMembers_RidealongUnchanged(t *testing.T) {
	media := &InputMediaPhoto{
		Media:   "https://example.com/photo.jpg",
		Caption: "look",
	}
	got, err := json.Marshal(media)
	require.NoError(t, err)
	require.JSONEq(t,
		`{"type":"photo","media":"https://example.com/photo.jpg","caption":"look"}`,
		string(got),
	)
}
