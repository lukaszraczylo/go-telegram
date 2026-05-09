package conversation

import (
	"fmt"

	"github.com/lukaszraczylo/go-telegram/api"
)

// KeyStrategy derives a persistence key from an update. Strategies
// determine how conversation scope works — per-user, per-chat, or
// per-user-and-chat. Implementations must return a stable string for
// the same logical scope across updates.
//
// Returns the empty string if the update doesn't have enough context
// to derive a key (in which case the conversation handler skips it).
type KeyStrategy func(u *api.Update) string

// KeyByUser derives a key from the sending user's ID. Useful for DM
// conversations and any flow that should follow the user across chats.
var KeyByUser KeyStrategy = func(u *api.Update) string {
	if uid := userID(u); uid != 0 {
		return fmt.Sprintf("u:%d", uid)
	}
	return ""
}

// KeyByChat derives a key from the chat ID. Useful for group flows where
// any user in the chat can drive the conversation.
var KeyByChat KeyStrategy = func(u *api.Update) string {
	if cid := chatID(u); cid != 0 {
		return fmt.Sprintf("c:%d", cid)
	}
	return ""
}

// KeyByUserAndChat derives a key from both user and chat IDs. The most
// common strategy: each user has their own conversation per chat.
var KeyByUserAndChat KeyStrategy = func(u *api.Update) string {
	uid := userID(u)
	cid := chatID(u)
	if uid == 0 || cid == 0 {
		return ""
	}
	return fmt.Sprintf("uc:%d:%d", cid, uid)
}

// userID extracts the sending user's ID from any update payload.
func userID(u *api.Update) int64 {
	switch {
	case u.Message != nil && u.Message.From != nil:
		return u.Message.From.ID
	case u.EditedMessage != nil && u.EditedMessage.From != nil:
		return u.EditedMessage.From.ID
	case u.CallbackQuery != nil:
		return u.CallbackQuery.From.ID
	case u.InlineQuery != nil:
		return u.InlineQuery.From.ID
	}
	return 0
}

// chatID extracts the relevant chat ID.
func chatID(u *api.Update) int64 {
	switch {
	case u.Message != nil:
		return u.Message.Chat.ID
	case u.EditedMessage != nil:
		return u.EditedMessage.Chat.ID
	case u.ChannelPost != nil:
		return u.ChannelPost.Chat.ID
	case u.EditedChannelPost != nil:
		return u.EditedChannelPost.Chat.ID
	case u.CallbackQuery != nil && u.CallbackQuery.Message != nil:
		if msg, ok := u.CallbackQuery.Message.(*api.Message); ok {
			return msg.Chat.ID
		}
	}
	return 0
}
