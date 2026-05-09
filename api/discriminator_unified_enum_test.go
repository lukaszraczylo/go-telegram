package api

import (
	"reflect"
	"testing"

	json "github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
)

// TestUnifiedEnum_BotCommandScopeType_Constants confirms the prose-form
// discriminator detection promoted BotCommandScope's per-variant Type
// fields into one shared enum.
func TestUnifiedEnum_BotCommandScopeType_Constants(t *testing.T) {
	require.IsType(t, BotCommandScopeType(""), BotCommandScopeTypeDefault)

	got := []BotCommandScopeType{
		BotCommandScopeTypeDefault,
		BotCommandScopeTypeAllPrivateChats,
		BotCommandScopeTypeAllGroupChats,
		BotCommandScopeTypeAllChatAdministrators,
		BotCommandScopeTypeChat,
		BotCommandScopeTypeChatAdministrators,
		BotCommandScopeTypeChatMember,
	}
	want := []string{
		"default", "all_private_chats", "all_group_chats",
		"all_chat_administrators", "chat", "chat_administrators", "chat_member",
	}
	require.Len(t, got, len(want))
	for i, v := range got {
		require.Equal(t, want[i], string(v))
	}
}

// TestUnifiedEnum_InlineQueryResultType_VariantFields walks the variants
// and asserts each one's Type field is the unified enum.
func TestUnifiedEnum_InlineQueryResultType_VariantFields(t *testing.T) {
	require.IsType(t, InlineQueryResultType(""), InlineQueryResultTypeArticle)

	wantType := reflect.TypeOf(InlineQueryResultType(""))
	cases := []any{
		&InlineQueryResultArticle{},
		&InlineQueryResultPhoto{},
		&InlineQueryResultGif{},
		&InlineQueryResultMpeg4Gif{},
		&InlineQueryResultVideo{},
		&InlineQueryResultAudio{},
		&InlineQueryResultVoice{},
		&InlineQueryResultDocument{},
		&InlineQueryResultLocation{},
		&InlineQueryResultVenue{},
		&InlineQueryResultContact{},
		&InlineQueryResultGame{},
	}
	for _, c := range cases {
		rt := reflect.TypeOf(c).Elem()
		f, ok := rt.FieldByName("Type")
		require.True(t, ok, "%s missing Type field", rt.Name())
		require.Equal(t, wantType, f.Type, "%s.Type type mismatch", rt.Name())
	}
}

// TestUnifiedEnum_PassportElementErrorSource_VariantFields asserts the
// retype landed on every variant of the PassportElementError union.
func TestUnifiedEnum_PassportElementErrorSource_VariantFields(t *testing.T) {
	require.IsType(t, PassportElementErrorSource(""), PassportElementErrorSourceData)

	wantType := reflect.TypeOf(PassportElementErrorSource(""))
	cases := []any{
		&PassportElementErrorDataField{},
		&PassportElementErrorFrontSide{},
		&PassportElementErrorReverseSide{},
		&PassportElementErrorSelfie{},
		&PassportElementErrorFile{},
		&PassportElementErrorFiles{},
		&PassportElementErrorTranslationFile{},
		&PassportElementErrorTranslationFiles{},
		&PassportElementErrorUnspecified{},
	}
	for _, c := range cases {
		rt := reflect.TypeOf(c).Elem()
		f, ok := rt.FieldByName("Source")
		require.True(t, ok, "%s missing Source field", rt.Name())
		require.Equal(t, wantType, f.Type, "%s.Source type mismatch", rt.Name())
	}
}

// TestUnifiedEnum_InputMediaType_Constants covers a media-shaped union
// where the discriminator value is the wire identifier "animation",
// "photo", etc.
func TestUnifiedEnum_InputMediaType_Constants(t *testing.T) {
	require.IsType(t, InputMediaType(""), InputMediaTypePhoto)

	wantType := reflect.TypeOf(InputMediaType(""))
	for _, c := range []any{
		&InputMediaAnimation{},
		&InputMediaAudio{},
		&InputMediaDocument{},
		&InputMediaPhoto{},
		&InputMediaVideo{},
	} {
		rt := reflect.TypeOf(c).Elem()
		f, ok := rt.FieldByName("Type")
		require.True(t, ok, "%s missing Type field", rt.Name())
		require.Equal(t, wantType, f.Type, "%s.Type type mismatch", rt.Name())
	}
}

// TestUnifiedEnum_MenuButtonType_Constants covers the third single-Type
// union pulled in by the prose detector.
func TestUnifiedEnum_MenuButtonType_Constants(t *testing.T) {
	require.IsType(t, MenuButtonType(""), MenuButtonTypeCommands)

	wantType := reflect.TypeOf(MenuButtonType(""))
	for _, c := range []any{
		&MenuButtonCommands{},
		&MenuButtonWebApp{},
		&MenuButtonDefault{},
	} {
		rt := reflect.TypeOf(c).Elem()
		f, ok := rt.FieldByName("Type")
		require.True(t, ok, "%s missing Type field", rt.Name())
		require.Equal(t, wantType, f.Type, "%s.Type type mismatch", rt.Name())
	}
}

// TestUnifiedEnum_InlineQueryResultArticle_RoundTrip confirms the
// auto-injected discriminator survives a marshal-unmarshal cycle on the
// concrete variant and lands as the typed enum constant. There's no
// generated UnmarshalInlineQueryResult — the union has no entry in
// knownDiscriminators — so the round-trip targets the variant directly.
func TestUnifiedEnum_InlineQueryResultArticle_RoundTrip(t *testing.T) {
	orig := &InlineQueryResultArticle{
		ID:    "x1",
		Title: "test",
	}
	raw, err := json.Marshal(orig)
	require.NoError(t, err)

	var probe struct {
		Type string `json:"type"`
	}
	require.NoError(t, json.Unmarshal(raw, &probe))
	require.Equal(t, "article", probe.Type)

	// Strip InputMessageContent before re-decoding: it's a sealed
	// interface and the variant has no UnmarshalJSON helper to dispatch
	// it. The discriminator round-trip is the property under test, not
	// nested-union deserialisation.
	var round struct {
		Type  InlineQueryResultType `json:"type"`
		ID    string                `json:"id"`
		Title string                `json:"title"`
	}
	require.NoError(t, json.Unmarshal(raw, &round))
	require.Equal(t, InlineQueryResultTypeArticle, round.Type)
	require.Equal(t, orig.ID, round.ID)
	require.Equal(t, orig.Title, round.Title)
}

// TestUnifiedEnum_PassportElementErrorDataField_RoundTrip mirrors the
// above for the Source-discriminated union.
func TestUnifiedEnum_PassportElementErrorDataField_RoundTrip(t *testing.T) {
	orig := &PassportElementErrorDataField{
		Type:      "personal_details",
		FieldName: "first_name",
		DataHash:  "abc",
		Message:   "boom",
	}
	raw, err := json.Marshal(orig)
	require.NoError(t, err)

	var probe struct {
		Source string `json:"source"`
	}
	require.NoError(t, json.Unmarshal(raw, &probe))
	require.Equal(t, "data", probe.Source)

	var round PassportElementErrorDataField
	require.NoError(t, json.Unmarshal(raw, &round))
	require.Equal(t, PassportElementErrorSourceData, round.Source)
	require.Equal(t, orig.FieldName, round.FieldName)
}

// TestUnifiedEnum_BotCommandScopeChat_RoundTrip covers a bot-command
// scope variant with a non-trivial extra field (ChatID).
func TestUnifiedEnum_BotCommandScopeChat_RoundTrip(t *testing.T) {
	orig := &BotCommandScopeChat{ChatID: ChatIDFromInt(42)}
	raw, err := json.Marshal(orig)
	require.NoError(t, err)

	var probe struct {
		Type string `json:"type"`
	}
	require.NoError(t, json.Unmarshal(raw, &probe))
	require.Equal(t, "chat", probe.Type)

	var round BotCommandScopeChat
	require.NoError(t, json.Unmarshal(raw, &round))
	require.Equal(t, BotCommandScopeTypeChat, round.Type)
}

// TestUnifiedEnum_InputMessageContent_NoEnumEmitted confirms the IMC
// union — which dispatches structurally on field presence rather than a
// shared discriminator — does NOT get a unified enum, since none of its
// variants declare a single-value discriminator field.
func TestUnifiedEnum_InputMessageContent_NoEnumEmitted(t *testing.T) {
	for _, name := range []string{
		"InputTextMessageContent",
		"InputLocationMessageContent",
		"InputVenueMessageContent",
		"InputContactMessageContent",
		"InputInvoiceMessageContent",
	} {
		switch name {
		case "InputTextMessageContent":
			rt := reflect.TypeOf(&InputTextMessageContent{}).Elem()
			_, ok := rt.FieldByName("Type")
			require.False(t, ok, "%s unexpectedly grew a Type field", name)
		case "InputLocationMessageContent":
			rt := reflect.TypeOf(&InputLocationMessageContent{}).Elem()
			_, ok := rt.FieldByName("Type")
			require.False(t, ok, "%s unexpectedly grew a Type field", name)
		case "InputVenueMessageContent":
			rt := reflect.TypeOf(&InputVenueMessageContent{}).Elem()
			_, ok := rt.FieldByName("Type")
			require.False(t, ok, "%s unexpectedly grew a Type field", name)
		case "InputContactMessageContent":
			rt := reflect.TypeOf(&InputContactMessageContent{}).Elem()
			_, ok := rt.FieldByName("Type")
			require.False(t, ok, "%s unexpectedly grew a Type field", name)
		case "InputInvoiceMessageContent":
			rt := reflect.TypeOf(&InputInvoiceMessageContent{}).Elem()
			_, ok := rt.FieldByName("Type")
			require.False(t, ok, "%s unexpectedly grew a Type field", name)
		}
	}
}
