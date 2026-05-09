// Package chatmember provides Filter helpers for *api.ChatMemberUpdated payloads.
package chatmember

import (
	"github.com/lukaszraczylo/go-telegram/api"
	"github.com/lukaszraczylo/go-telegram/dispatch"
)

// NewStatus returns a Filter that matches updates where the new chat member
// status equals s (e.g. "member", "administrator", "kicked", "left").
func NewStatus(s string) dispatch.Filter[*api.ChatMemberUpdated] {
	return func(u *api.ChatMemberUpdated) bool {
		if u == nil {
			return false
		}
		switch m := u.NewChatMember.(type) {
		case *api.ChatMemberOwner:
			return string(m.Status) == s
		case *api.ChatMemberAdministrator:
			return string(m.Status) == s
		case *api.ChatMemberMember:
			return string(m.Status) == s
		case *api.ChatMemberRestricted:
			return string(m.Status) == s
		case *api.ChatMemberLeft:
			return string(m.Status) == s
		case *api.ChatMemberBanned:
			return string(m.Status) == s
		default:
			return false
		}
	}
}

// FromUser returns a Filter that matches updates where the acting user
// (From.ID) equals uid.
func FromUser(uid int64) dispatch.Filter[*api.ChatMemberUpdated] {
	return func(u *api.ChatMemberUpdated) bool {
		return u != nil && u.From.ID == uid
	}
}
