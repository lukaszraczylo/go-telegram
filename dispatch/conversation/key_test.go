package conversation

import (
	"testing"

	"github.com/lukaszraczylo/go-telegram/api"
	"github.com/stretchr/testify/require"
)

// helpers to build api.Update variants.

func msgUpdate(userID, chatID int64) *api.Update {
	return &api.Update{
		Message: &api.Message{
			From: &api.User{ID: userID},
			Chat: api.Chat{ID: chatID},
		},
	}
}

func editedMsgUpdate(userID, chatID int64) *api.Update {
	return &api.Update{
		EditedMessage: &api.Message{
			From: &api.User{ID: userID},
			Chat: api.Chat{ID: chatID},
		},
	}
}

func callbackUpdate(userID, chatID int64) *api.Update {
	return &api.Update{
		CallbackQuery: &api.CallbackQuery{
			From:    api.User{ID: userID},
			Message: &api.Message{Chat: api.Chat{ID: chatID}},
		},
	}
}

func inlineUpdate(userID int64) *api.Update {
	return &api.Update{
		InlineQuery: &api.InlineQuery{
			From: api.User{ID: userID},
		},
	}
}

func emptyUpdate() *api.Update { return &api.Update{} }

func TestKeyByUser(t *testing.T) {
	t.Run("message update", func(t *testing.T) {
		require.Equal(t, "u:42", KeyByUser(msgUpdate(42, 100)))
	})
	t.Run("edited message", func(t *testing.T) {
		require.Equal(t, "u:7", KeyByUser(editedMsgUpdate(7, 100)))
	})
	t.Run("callback query", func(t *testing.T) {
		require.Equal(t, "u:99", KeyByUser(callbackUpdate(99, 100)))
	})
	t.Run("inline query", func(t *testing.T) {
		require.Equal(t, "u:5", KeyByUser(inlineUpdate(5)))
	})
	t.Run("empty update returns empty string", func(t *testing.T) {
		require.Equal(t, "", KeyByUser(emptyUpdate()))
	})
}

func TestKeyByChat(t *testing.T) {
	t.Run("message update", func(t *testing.T) {
		require.Equal(t, "c:100", KeyByChat(msgUpdate(42, 100)))
	})
	t.Run("edited message", func(t *testing.T) {
		require.Equal(t, "c:200", KeyByChat(editedMsgUpdate(7, 200)))
	})
	t.Run("callback with accessible message", func(t *testing.T) {
		require.Equal(t, "c:300", KeyByChat(callbackUpdate(99, 300)))
	})
	t.Run("inline query has no chat → empty", func(t *testing.T) {
		require.Equal(t, "", KeyByChat(inlineUpdate(5)))
	})
	t.Run("empty update returns empty string", func(t *testing.T) {
		require.Equal(t, "", KeyByChat(emptyUpdate()))
	})
}

func TestKeyByUserAndChat(t *testing.T) {
	t.Run("message update", func(t *testing.T) {
		require.Equal(t, "uc:100:42", KeyByUserAndChat(msgUpdate(42, 100)))
	})
	t.Run("edited message", func(t *testing.T) {
		require.Equal(t, "uc:200:7", KeyByUserAndChat(editedMsgUpdate(7, 200)))
	})
	t.Run("callback query", func(t *testing.T) {
		require.Equal(t, "uc:300:99", KeyByUserAndChat(callbackUpdate(99, 300)))
	})
	t.Run("inline query has no chat → empty", func(t *testing.T) {
		require.Equal(t, "", KeyByUserAndChat(inlineUpdate(5)))
	})
	t.Run("empty update returns empty string", func(t *testing.T) {
		require.Equal(t, "", KeyByUserAndChat(emptyUpdate()))
	})
}

func TestKeyByUserAndChat_CallbackInaccessibleMessage(t *testing.T) {
	// CallbackQuery.Message is InaccessibleMessage (not *Message) — chatID returns 0.
	u := &api.Update{
		CallbackQuery: &api.CallbackQuery{
			From:    api.User{ID: 10},
			Message: &api.InaccessibleMessage{}, // implements MaybeInaccessibleMessage, not *api.Message
		},
	}
	// userID picks up From.ID=10 but chatID fails type assertion → 0
	require.Equal(t, "", KeyByUserAndChat(u), "no key when message inaccessible")
	// KeyByUser still works since From is set.
	require.Equal(t, "u:10", KeyByUser(u))
}
