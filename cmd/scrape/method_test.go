package main

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lukaszraczylo/go-telegram/internal/spec"
)

func TestExtractReturn(t *testing.T) {
	cases := []struct {
		in   string
		want spec.TypeRef
	}{
		{"Returns basic information about the bot in form of a User object.", spec.TypeRef{Kind: spec.KindNamed, Name: "User"}},
		{"On success, the sent Message is returned.", spec.TypeRef{Kind: spec.KindNamed, Name: "Message"}},
		{"Returns an Array of Update objects.", spec.TypeRef{Kind: spec.KindArray, ElemType: &spec.TypeRef{Kind: spec.KindNamed, Name: "Update"}}},
		{"Returns True on success.", spec.TypeRef{Kind: spec.KindPrimitive, Name: "bool"}},
		{"On success, True is returned.", spec.TypeRef{Kind: spec.KindPrimitive, Name: "bool"}},
		// Issue 5: "Message or True" conditional return → MessageOrBool sentinel.
		{"On success, if the edited message is not an inline message, the edited Message is returned, otherwise True is returned.", spec.TypeRef{Kind: spec.KindNamed, Name: "MessageOrBool"}},
		// Issue 1: new phrasings.
		{"On success, returns a WebhookInfo object.", spec.TypeRef{Kind: spec.KindNamed, Name: "WebhookInfo"}},
		{"Returns a UserProfilePhotos object.", spec.TypeRef{Kind: spec.KindNamed, Name: "UserProfilePhotos"}},
		{"Returns the uploaded File.", spec.TypeRef{Kind: spec.KindNamed, Name: "File"}},
		{"On success, the stopped Poll is returned.", spec.TypeRef{Kind: spec.KindNamed, Name: "Poll"}},
		{"On success, an Array of MessageId is returned.", spec.TypeRef{Kind: spec.KindArray, ElemType: &spec.TypeRef{Kind: spec.KindNamed, Name: "MessageId"}}},
		{"On success, an array of Message objects that were sent is returned.", spec.TypeRef{Kind: spec.KindArray, ElemType: &spec.TypeRef{Kind: spec.KindNamed, Name: "Message"}}},
		// ForwardMessages/CopyMessages shape: "an array of X of the sent messages is returned".
		{"On success, an array of MessageId of the sent messages is returned.", spec.TypeRef{Kind: spec.KindArray, ElemType: &spec.TypeRef{Kind: spec.KindNamed, Name: "MessageId"}}},
		// "Returns X on success" (no article) — OwnedGifts, StarAmount, Story, MenuButton, etc.
		{"Returns the gifts received and owned by a managed business account. Returns OwnedGifts on success.", spec.TypeRef{Kind: spec.KindNamed, Name: "OwnedGifts"}},
		{"Returns StarAmount on success.", spec.TypeRef{Kind: spec.KindNamed, Name: "StarAmount"}},
		{"Posts a story on behalf of a managed business account. Returns Story on success.", spec.TypeRef{Kind: spec.KindNamed, Name: "Story"}},
		{"Returns MenuButton on success.", spec.TypeRef{Kind: spec.KindNamed, Name: "MenuButton"}},
		// "Returns ... as X object" (no article before type) — ChatInviteLink variants.
		{"Returns the new invite link as ChatInviteLink object.", spec.TypeRef{Kind: spec.KindNamed, Name: "ChatInviteLink"}},
		{"Returns the revoked invite link as ChatInviteLink object.", spec.TypeRef{Kind: spec.KindNamed, Name: "ChatInviteLink"}},
		// "Returns ... as a X object" (with article) — createForumTopic.
		{"Returns information about the created topic as a ForumTopic object.", spec.TypeRef{Kind: spec.KindNamed, Name: "ForumTopic"}},
		// "Returns ... as String on success" — exportChatInviteLink / createInvoiceLink.
		{"Returns the new invite link as String on success.", spec.TypeRef{Kind: spec.KindPrimitive, Name: "string"}},
		{"Returns the created invoice link as String on success.", spec.TypeRef{Kind: spec.KindPrimitive, Name: "string"}},
		// "Returns Int on success" — getChatMemberCount.
		{"Returns Int on success.", spec.TypeRef{Kind: spec.KindPrimitive, Name: "int64"}},
	}
	for _, c := range cases {
		require.Equal(t, c.want, extractReturn(c.in), c.in)
	}
}

func TestHasFilesParams(t *testing.T) {
	require.True(t, hasFilesParams([]spec.Field{
		{Type: spec.TypeRef{Kind: spec.KindNamed, Name: "InputFile"}},
	}))
	require.True(t, hasFilesParams([]spec.Field{
		{Type: spec.TypeRef{Kind: spec.KindOneOf, Variants: []string{"InputFile", "string"}}},
	}))
	require.False(t, hasFilesParams([]spec.Field{
		{Type: spec.TypeRef{Kind: spec.KindPrimitive, Name: "string"}},
	}))
	// Issue 2: Array of InputMedia* union triggers HasFiles.
	require.True(t, hasFilesParams([]spec.Field{
		{Type: spec.TypeRef{Kind: spec.KindArray, ElemType: &spec.TypeRef{Kind: spec.KindOneOf, Variants: []string{"InputMediaPhoto", "InputMediaVideo"}}}},
	}))
}

func TestExtractVersion(t *testing.T) {
	sections := []section{{Title: "Recent changes"}, {Title: "Bot API 7.10"}, {Title: "Available types"}}
	require.Equal(t, "7.10", extractVersion(sections))

	// Issue 4: 3-part version must not be truncated.
	sections3 := []section{{Description: "Bot API 8.0.1"}}
	require.Equal(t, "8.0.1", extractVersion(sections3))
}
