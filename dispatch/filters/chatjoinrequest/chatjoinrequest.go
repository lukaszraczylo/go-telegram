// Package chatjoinrequest provides Filter helpers for *api.ChatJoinRequest payloads.
package chatjoinrequest

import (
	"github.com/lukaszraczylo/go-telegram/api"
	"github.com/lukaszraczylo/go-telegram/dispatch"
)

// FromUser returns a Filter that matches join requests where the requesting
// user's ID equals uid.
func FromUser(uid int64) dispatch.Filter[*api.ChatJoinRequest] {
	return func(r *api.ChatJoinRequest) bool {
		return r != nil && r.From.ID == uid
	}
}

// InChat returns a Filter that matches join requests directed at the chat
// with the given chat ID.
func InChat(cid int64) dispatch.Filter[*api.ChatJoinRequest] {
	return func(r *api.ChatJoinRequest) bool {
		return r != nil && r.Chat.ID == cid
	}
}
