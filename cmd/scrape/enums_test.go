package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractEnumValues_CanBeEither(t *testing.T) {
	desc := "Type of the chat, can be either “private”, “group”, “supergroup” or “channel”"
	got := extractEnumValues("type", desc)
	require.Equal(t, []string{"private", "group", "supergroup", "channel"}, got)
}

func TestExtractEnumValues_CurrentlyCanBe(t *testing.T) {
	desc := "Type of the entity. Currently, can be “mention” (@username), “hashtag” (#hashtag), or “code” (monowidth string)"
	got := extractEnumValues("type", desc)
	require.Equal(t, []string{"mention", "hashtag", "code"}, got)
}

func TestExtractEnumValues_AlwaysSingle(t *testing.T) {
	desc := "Type of the message origin, always “user”"
	got := extractEnumValues("type", desc)
	require.Equal(t, []string{"user"}, got)
}

func TestExtractEnumValues_MustBeOneOfMime(t *testing.T) {
	desc := "Optional. MIME type of the thumbnail, must be one of “image/jpeg”, “image/gif”, or “video/mp4”. Defaults to “image/jpeg”"
	got := extractEnumValues("thumbnail_mime_type", desc)
	require.Equal(t, []string{"image/jpeg", "image/gif", "video/mp4"}, got)
}

func TestExtractEnumValues_ParseModeSpecial(t *testing.T) {
	desc := "Optional. Mode for parsing entities in the message text. See formatting options for more details."
	got := extractEnumValues("parse_mode", desc)
	require.Equal(t, []string{"Markdown", "MarkdownV2", "HTML"}, got)
}

func TestExtractEnumValues_QuestionParseMode(t *testing.T) {
	desc := "Mode for parsing entities in the question. See formatting options for more details."
	got := extractEnumValues("question_parse_mode", desc)
	require.Equal(t, []string{"Markdown", "MarkdownV2", "HTML"}, got)
}

func TestExtractEnumValues_FalsePositiveReferencedValue(t *testing.T) {
	// "Can be available only for "X"" is NOT an enum: the quote is a
	// reference to a transaction-type value, not an introduced list.
	desc := "Optional. Bot-specified invoice payload. Can be available only for “invoice_payment” transactions."
	got := extractEnumValues("invoice_payload", desc)
	require.Nil(t, got)
}

func TestExtractEnumValues_FalsePositiveSingleQuotedNonEnum(t *testing.T) {
	// "can be ignored" with a quoted reference value later — not an enum.
	desc := "Optional. Thumbnail of the file sent; can be ignored if thumbnail generation for the file is supported server-side."
	got := extractEnumValues("thumbnail", desc)
	require.Nil(t, got)
}

func TestExtractEnumValues_RefundedPaymentCurrentlyAlways(t *testing.T) {
	desc := "Three-letter ISO 4217 currency code, or “XTR” for payments in Telegram Stars. Currently, always “XTR”"
	got := extractEnumValues("currency", desc)
	require.Equal(t, []string{"XTR"}, got)
}

func TestExtractEnumValues_RejectURLValues(t *testing.T) {
	// "attach://<file_attach_name>" must never be promoted to an enum value.
	desc := "Pass “attach://<file_attach_name>” to upload a new one"
	got := extractEnumValues("media", desc)
	require.Nil(t, got)
}

func TestExtractEnumValues_StringTypeOnly(t *testing.T) {
	// (Sanity — table.go gates on string type, but the function itself
	// should still respond consistently.)
	desc := "ABC, can be “a”, “b”"
	got := extractEnumValues("x", desc)
	require.Equal(t, []string{"a", "b"}, got)
}

func TestExtractEnumValues_DedupeRepeatedValues(t *testing.T) {
	desc := "Currently, one of “XTR” for Telegram Stars or “XTR” again"
	got := extractEnumValues("currency", desc)
	require.Equal(t, []string{"XTR"}, got)
}

func TestExtractEnumValues_ProseDiscriminator(t *testing.T) {
	cases := []struct {
		name string
		desc string
		want []string
	}{
		{"InlineQueryResultArticle", "Type of the result, must be article", []string{"article"}},
		{"InlineQueryResultPhoto", "Type of the result, must be photo", []string{"photo"}},
		{"InlineQueryResultMpeg4Gif", "Type of the result, must be mpeg4_gif", []string{"mpeg4_gif"}},
		{"BotCommandScopeAllPrivateChats", "Scope type, must be all_private_chats", []string{"all_private_chats"}},
		{"BotCommandScopeChat", "Scope type, must be chat", []string{"chat"}},
		{"PassportElementErrorData", "Error source, must be data", []string{"data"}},
		{"MenuButtonWebApp", "Type of the button, must be web_app", []string{"web_app"}},
		{"InputProfilePhotoAnimated", "Type of the profile photo, must be animated", []string{"animated"}},
		{"InputStoryContentVideo", "Type of the content, must be video", []string{"video"}},
		{"InputPaidMediaPhoto", "Type of the media, must be photo", []string{"photo"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, extractEnumValues("type", tc.desc))
		})
	}
}

func TestExtractEnumValues_ProseFalsePositives(t *testing.T) {
	cases := []struct {
		name string
		desc string
	}{
		{"available_only_for", "Optional. Bot-specified invoice payload. Can be available only for “invoice_payment” transactions."},
		{"must_be_sent", "If True, the message must be sent immediately."},
		{"must_be_shown_above", "Optional. True, if the link preview must be shown above the message text"},
		{"must_be_specified", "The identifiers must be specified in a strictly increasing order."},
		{"must_be_paid", "The number of Telegram Stars that must be paid to send the sticker"},
		{"must_be_one_of_numbers", "Number of months the Telegram Premium subscription will be active for the user; must be one of 3, 6, or 12"},
		{"must_be_between", "Currently, price in Telegram Stars must be between 5 and 100000"},
		{"must_be_a_pay_button", "If not empty, the first button must be a Pay button."},
		{"must_be_repainted", "True, if the sticker must be repainted to a text color in messages"},
		{"must_be_active", "the subscription must be active up to the end of the current subscription period"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Nil(t, extractEnumValues("type", tc.desc))
		})
	}
}

func TestExtractEnumValues_CanonicalMustBeOneOfStillWorks(t *testing.T) {
	desc := "Currently, must be one of “Markdown”, “MarkdownV2”, “HTML”"
	got := extractEnumValues("parse_mode_kind", desc)
	require.Equal(t, []string{"Markdown", "MarkdownV2", "HTML"}, got)
}
