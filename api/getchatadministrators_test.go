package api

import (
	"context"
	"testing"

	"github.com/lukaszraczylo/go-telegram/client"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestGetChatAdministrators_DecodesUnionSlice is a regression test for the
// bug where GetChatAdministrators was emitted with the generic client.Call
// against []ChatMember — encoding/json cannot unmarshal a slice of an
// interface, so the call always failed at the parse step.
//
// The fix makes the codegen emit CallRaw + per-element UnmarshalChatMember
// for any method returning []<sealed-interface union>.
func TestGetChatAdministrators_DecodesUnionSlice(t *testing.T) {
	body := `{"ok":true,"result":[
		{"status":"creator","user":{"id":1,"is_bot":false,"first_name":"Owner"},"is_anonymous":false},
		{"status":"administrator","user":{"id":2,"is_bot":false,"first_name":"Admin"},"can_be_edited":false,"is_anonymous":false,"can_manage_chat":true,"can_delete_messages":true,"can_manage_video_chats":false,"can_restrict_members":true,"can_promote_members":false,"can_change_info":true,"can_invite_users":true,"can_post_stories":false,"can_edit_stories":false,"can_delete_stories":false}
	]}`

	m := &mockDoer{}
	m.On("Do", mock.Anything).Return(newJSONResp(200, body), nil)
	bot := client.New("test:token", client.WithHTTPClient(m))

	admins, err := GetChatAdministrators(context.Background(), bot,
		&GetChatAdministratorsParams{ChatID: ChatIDFromInt(-100123)})
	require.NoError(t, err)
	require.Len(t, admins, 2)

	owner, ok := admins[0].(*ChatMemberOwner)
	require.True(t, ok, "first element must dispatch to ChatMemberOwner, got %T", admins[0])
	require.Equal(t, int64(1), owner.User.ID)

	admin, ok := admins[1].(*ChatMemberAdministrator)
	require.True(t, ok, "second element must dispatch to ChatMemberAdministrator, got %T", admins[1])
	require.True(t, admin.CanManageChat)
	require.False(t, admin.CanPromoteMembers)
}

// TestGetChatAdministrators_EmptyArray covers the zero-admin edge case
// (a basic group with no admins, or the bot itself filtered out).
func TestGetChatAdministrators_EmptyArray(t *testing.T) {
	m := &mockDoer{}
	m.On("Do", mock.Anything).Return(newJSONResp(200, `{"ok":true,"result":[]}`), nil)
	bot := client.New("test:token", client.WithHTTPClient(m))

	admins, err := GetChatAdministrators(context.Background(), bot,
		&GetChatAdministratorsParams{ChatID: ChatIDFromInt(-100123)})
	require.NoError(t, err)
	require.Empty(t, admins)
}
