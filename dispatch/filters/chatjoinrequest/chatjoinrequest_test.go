package chatjoinrequest_test

import (
	"testing"

	"github.com/lukaszraczylo/go-telegram/api"
	cjrfilter "github.com/lukaszraczylo/go-telegram/dispatch/filters/chatjoinrequest"
	"github.com/stretchr/testify/require"
)

func joinRequest(fromID, chatID int64) *api.ChatJoinRequest {
	return &api.ChatJoinRequest{
		Chat: api.Chat{ID: chatID},
		From: api.User{ID: fromID},
	}
}

func TestFromUser_Matches(t *testing.T) {
	f := cjrfilter.FromUser(10)
	require.True(t, f(joinRequest(10, 100)))
	require.False(t, f(joinRequest(99, 100)))
	require.False(t, f(nil))
}

func TestInChat_Matches(t *testing.T) {
	f := cjrfilter.InChat(100)
	require.True(t, f(joinRequest(10, 100)))
	require.False(t, f(joinRequest(10, 200)))
	require.False(t, f(nil))
}

func TestComposedFilters(t *testing.T) {
	f := cjrfilter.FromUser(10).And(cjrfilter.InChat(100))
	require.True(t, f(joinRequest(10, 100)))
	require.False(t, f(joinRequest(10, 200)))
	require.False(t, f(joinRequest(99, 100)))
}
