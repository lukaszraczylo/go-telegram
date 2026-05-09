package chatmember_test

import (
	"testing"

	"github.com/lukaszraczylo/go-telegram/api"
	cmfilter "github.com/lukaszraczylo/go-telegram/dispatch/filters/chatmember"
	"github.com/stretchr/testify/require"
)

func memberUpdate(status string, fromID int64) *api.ChatMemberUpdated {
	var newMember api.ChatMember
	switch status {
	case "member":
		newMember = &api.ChatMemberMember{Status: api.ChatMemberMemberStatusMember}
	case "administrator":
		newMember = &api.ChatMemberAdministrator{Status: api.ChatMemberAdministratorStatusAdministrator}
	case "kicked":
		newMember = &api.ChatMemberBanned{Status: api.ChatMemberBannedStatusKicked}
	case "left":
		newMember = &api.ChatMemberLeft{Status: api.ChatMemberLeftStatusLeft}
	default:
		newMember = &api.ChatMemberMember{Status: api.ChatMemberMemberStatusMember}
	}
	return &api.ChatMemberUpdated{
		From:          api.User{ID: fromID},
		NewChatMember: newMember,
	}
}

func TestNewStatus_Matches(t *testing.T) {
	f := cmfilter.NewStatus("member")
	require.True(t, f(memberUpdate("member", 1)))
	require.False(t, f(memberUpdate("kicked", 1)))
	require.False(t, f(nil))
}

func TestNewStatus_Administrator(t *testing.T) {
	f := cmfilter.NewStatus("administrator")
	require.True(t, f(memberUpdate("administrator", 1)))
	require.False(t, f(memberUpdate("member", 1)))
}

func TestNewStatus_Kicked(t *testing.T) {
	f := cmfilter.NewStatus("kicked")
	require.True(t, f(memberUpdate("kicked", 1)))
	require.False(t, f(memberUpdate("left", 1)))
}

func TestNewStatus_Left(t *testing.T) {
	f := cmfilter.NewStatus("left")
	require.True(t, f(memberUpdate("left", 1)))
	require.False(t, f(memberUpdate("member", 1)))
}

func TestFromUser_Matches(t *testing.T) {
	f := cmfilter.FromUser(42)
	require.True(t, f(memberUpdate("member", 42)))
	require.False(t, f(memberUpdate("member", 99)))
	require.False(t, f(nil))
}

func TestComposedFilters(t *testing.T) {
	f := cmfilter.NewStatus("member").And(cmfilter.FromUser(7))
	require.True(t, f(memberUpdate("member", 7)))
	require.False(t, f(memberUpdate("member", 8)))
	require.False(t, f(memberUpdate("kicked", 7)))
}

func TestNewStatus_Owner(t *testing.T) {
	u := &api.ChatMemberUpdated{
		From:          api.User{ID: 1},
		NewChatMember: &api.ChatMemberOwner{Status: api.ChatMemberOwnerStatusCreator},
	}
	require.True(t, cmfilter.NewStatus("creator")(u))
	require.False(t, cmfilter.NewStatus("member")(u))
}

func TestNewStatus_Restricted(t *testing.T) {
	u := &api.ChatMemberUpdated{
		From:          api.User{ID: 1},
		NewChatMember: &api.ChatMemberRestricted{Status: api.ChatMemberRestrictedStatusRestricted},
	}
	require.True(t, cmfilter.NewStatus("restricted")(u))
	require.False(t, cmfilter.NewStatus("member")(u))
}

func TestNewStatus_UnknownType(t *testing.T) {
	// nil NewChatMember → default branch → false
	u := &api.ChatMemberUpdated{
		From:          api.User{ID: 1},
		NewChatMember: nil,
	}
	require.False(t, cmfilter.NewStatus("member")(u))
}
