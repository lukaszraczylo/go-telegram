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
