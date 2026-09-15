package api

import (
	"testing"

	json "github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
)

// TestUpdate_RichMessage_Decodes is a regression test for the Bot API v10
// rich_message decode failure: RichBlock (21 variants) and RichText (25
// variants) had no knownDiscriminators entry, so genapi never emitted
// UnmarshalRichBlock / UnmarshalRichText and any update whose message
// carried rich_message failed to decode — in production this rejected
// roughly 8.6% of Telegram updates with HTTP 400. Uses goccy/go-json, the
// codec production runs; plain encoding/json failed the same way.
func TestUpdate_RichMessage_Decodes(t *testing.T) {
	body := `{"update_id":1,"message":{"message_id":7,"date":1789000000,"chat":{"id":-100123,"type":"supergroup"},` +
		`"rich_message":{"blocks":[{"type":"paragraph","text":{"type":"bold","text":{"type":"anchor","name":"hi"}}}]}}}`

	var u Update
	require.NoError(t, json.Unmarshal([]byte(body), &u))

	require.NotNil(t, u.Message)
	require.NotNil(t, u.Message.RichMessage)
	require.Len(t, u.Message.RichMessage.Blocks, 1)

	para, ok := u.Message.RichMessage.Blocks[0].(*RichBlockParagraph)
	require.True(t, ok, "blocks[0] must dispatch to *RichBlockParagraph, got %T", u.Message.RichMessage.Blocks[0])

	bold, ok := para.Text.(*RichTextBold)
	require.True(t, ok, "paragraph.text must dispatch to *RichTextBold, got %T", para.Text)

	anchor, ok := bold.Text.(*RichTextAnchor)
	require.True(t, ok, "bold.text must dispatch to *RichTextAnchor, got %T", bold.Text)
	require.Equal(t, "hi", anchor.Name)
}

// TestUnmarshalRichBlock_UnknownType confirms an unrecognised discriminator
// value surfaces a clear error instead of silently decoding a zero-valued
// interface.
func TestUnmarshalRichBlock_UnknownType(t *testing.T) {
	_, err := UnmarshalRichBlock([]byte(`{"type":"not_a_real_block"}`))
	require.Error(t, err)
	require.ErrorContains(t, err, "unknown type")
}

// TestUpdate_RichMessage_UnknownBlockType confirms the unknown-type error
// from UnmarshalRichBlock propagates all the way up through
// RichMessage.UnmarshalJSON to the top-level Update decode.
func TestUpdate_RichMessage_UnknownBlockType(t *testing.T) {
	body := `{"update_id":1,"message":{"message_id":7,"date":1789000000,"chat":{"id":-100123,"type":"supergroup"},` +
		`"rich_message":{"blocks":[{"type":"not_a_real_block"}]}}}`

	var u Update
	err := json.Unmarshal([]byte(body), &u)
	require.Error(t, err)
	require.ErrorContains(t, err, "unknown type")
}

// TestUpdate_RichMessage_PlainStringText is a regression test for the Bot
// API v10 RichText decode gap this task fixes. Telegram declares RichText
// as decodable from "either a String for plain text, an Array of
// RichText, or any of the following [object] types" — but the IR (and
// therefore UnmarshalRichText) only ever recorded the object variants, so
// a paragraph whose text is a bare JSON string failed with "invalid
// character '\"' looking for beginning of value". A plain-text paragraph
// is the most common real rich_message payload, so this was the
// production-facing half of the bug.
func TestUpdate_RichMessage_PlainStringText(t *testing.T) {
	body := `{"update_id":1,"message":{"message_id":7,"date":1789000000,"chat":{"id":-100123,"type":"supergroup"},` +
		`"rich_message":{"blocks":[{"type":"paragraph","text":"hello world"}]}}}`

	var u Update
	require.NoError(t, json.Unmarshal([]byte(body), &u))

	require.NotNil(t, u.Message)
	require.NotNil(t, u.Message.RichMessage)
	require.Len(t, u.Message.RichMessage.Blocks, 1)

	para, ok := u.Message.RichMessage.Blocks[0].(*RichBlockParagraph)
	require.True(t, ok, "blocks[0] must dispatch to *RichBlockParagraph, got %T", u.Message.RichMessage.Blocks[0])

	plain, ok := para.Text.(*RichTextPlain)
	require.True(t, ok, "paragraph.text must dispatch to *RichTextPlain for a bare JSON string, got %T", para.Text)
	require.Equal(t, "hello world", plain.Text)
}

// TestUpdate_RichMessage_ArrayText confirms Telegram's second declared
// RichText alternate — a bare JSON array of RichText — decodes into
// RichTextSequence, with each element dispatched recursively through
// UnmarshalRichText (an object variant in this case).
func TestUpdate_RichMessage_ArrayText(t *testing.T) {
	body := `{"update_id":1,"message":{"message_id":7,"date":1789000000,"chat":{"id":-100123,"type":"supergroup"},` +
		`"rich_message":{"blocks":[{"type":"paragraph","text":[` +
		`{"type":"bold","text":"x"},{"type":"italic","text":"y"}` +
		`]}]}}}`

	var u Update
	require.NoError(t, json.Unmarshal([]byte(body), &u))

	para, ok := u.Message.RichMessage.Blocks[0].(*RichBlockParagraph)
	require.True(t, ok)

	seq, ok := para.Text.(*RichTextSequence)
	require.True(t, ok, "paragraph.text must dispatch to *RichTextSequence for a JSON array, got %T", para.Text)
	require.Len(t, seq.Items, 2)

	bold, ok := seq.Items[0].(*RichTextBold)
	require.True(t, ok, "sequence[0] must be *RichTextBold, got %T", seq.Items[0])
	boldText, ok := bold.Text.(*RichTextPlain)
	require.True(t, ok)
	require.Equal(t, "x", boldText.Text)

	italic, ok := seq.Items[1].(*RichTextItalic)
	require.True(t, ok, "sequence[1] must be *RichTextItalic, got %T", seq.Items[1])
	italicText, ok := italic.Text.(*RichTextPlain)
	require.True(t, ok)
	require.Equal(t, "y", italicText.Text)
}

// TestUpdate_RichMessage_ArrayText_MixedElements confirms an array whose
// elements mix the bare-string shape and the object shape dispatches
// each element independently — the array alternate and the string
// alternate compose, they aren't mutually exclusive per element.
func TestUpdate_RichMessage_ArrayText_MixedElements(t *testing.T) {
	body := `{"update_id":1,"message":{"message_id":7,"date":1789000000,"chat":{"id":-100123,"type":"supergroup"},` +
		`"rich_message":{"blocks":[{"type":"paragraph","text":[` +
		`"plain segment",{"type":"bold","text":"bold segment"}` +
		`]}]}}}`

	var u Update
	require.NoError(t, json.Unmarshal([]byte(body), &u))

	para, ok := u.Message.RichMessage.Blocks[0].(*RichBlockParagraph)
	require.True(t, ok)

	seq, ok := para.Text.(*RichTextSequence)
	require.True(t, ok)
	require.Len(t, seq.Items, 2)

	plain, ok := seq.Items[0].(*RichTextPlain)
	require.True(t, ok, "sequence[0] must be *RichTextPlain, got %T", seq.Items[0])
	require.Equal(t, "plain segment", plain.Text)

	bold, ok := seq.Items[1].(*RichTextBold)
	require.True(t, ok, "sequence[1] must be *RichTextBold, got %T", seq.Items[1])
	boldText, ok := bold.Text.(*RichTextPlain)
	require.True(t, ok)
	require.Equal(t, "bold segment", boldText.Text)
}

// TestRichTextPlain_MarshalJSON_RoundTrip confirms the plain-text
// alternate marshals back to a bare JSON string (not an object), and
// that string decodes straight back through UnmarshalRichText.
func TestRichTextPlain_MarshalJSON_RoundTrip(t *testing.T) {
	v := &RichTextPlain{Text: "hello"}

	b, err := json.Marshal(v)
	require.NoError(t, err)
	require.JSONEq(t, `"hello"`, string(b))

	decoded, err := UnmarshalRichText(b)
	require.NoError(t, err)
	got, ok := decoded.(*RichTextPlain)
	require.True(t, ok)
	require.Equal(t, "hello", got.Text)
}

// TestRichTextSequence_MarshalJSON_RoundTrip confirms the array
// alternate marshals back to a bare JSON array (not an object wrapping
// one), preserving each element's own wire shape, and round-trips
// through UnmarshalRichText.
func TestRichTextSequence_MarshalJSON_RoundTrip(t *testing.T) {
	v := &RichTextSequence{
		Items: []RichText{
			&RichTextPlain{Text: "a"},
			&RichTextBold{Text: &RichTextPlain{Text: "b"}},
		},
	}

	b, err := json.Marshal(v)
	require.NoError(t, err)
	require.JSONEq(t, `["a",{"type":"bold","text":"b"}]`, string(b))

	decoded, err := UnmarshalRichText(b)
	require.NoError(t, err)
	seq, ok := decoded.(*RichTextSequence)
	require.True(t, ok)
	require.Len(t, seq.Items, 2)

	p0, ok := seq.Items[0].(*RichTextPlain)
	require.True(t, ok)
	require.Equal(t, "a", p0.Text)

	b1, ok := seq.Items[1].(*RichTextBold)
	require.True(t, ok)
	p1, ok := b1.Text.(*RichTextPlain)
	require.True(t, ok)
	require.Equal(t, "b", p1.Text)
}

// TestUnmarshalRichText_UnrecognisedShape confirms a JSON value that is
// neither a string, an array, nor an object surfaces a clear error
// instead of silently producing a zero-valued interface.
func TestUnmarshalRichText_UnrecognisedShape(t *testing.T) {
	_, err := UnmarshalRichText([]byte(`42`))
	require.Error(t, err)
	require.ErrorContains(t, err, "unrecognised JSON value")
}
