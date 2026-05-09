package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/lukaszraczylo/go-telegram/api"
	"github.com/lukaszraczylo/go-telegram/client"
	"github.com/lukaszraczylo/go-telegram/dispatch"
	"github.com/lukaszraczylo/go-telegram/dispatch/conversation"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockDoer satisfies client.HTTPDoer.
type mockDoer struct{ mock.Mock }

func (m *mockDoer) Do(r *http.Request) (*http.Response, error) {
	args := m.Called(r)
	if v := args.Get(0); v != nil {
		return v.(*http.Response), args.Error(1)
	}
	return nil, args.Error(1)
}

func anyResp() *http.Response {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(`{"ok":true,"result":{"message_id":1,"date":0,"chat":{"id":1,"type":"private"}}}`)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}

// msgUpd builds a message update with the given user/chat/text.
func msgUpd(userID, chatID int64, text string) api.Update {
	entities := []api.MessageEntity{}
	if len(text) > 0 && text[0] == '/' {
		end := len(text)
		for i, r := range text {
			if r == ' ' {
				end = i
				break
			}
		}
		entities = append(entities, api.MessageEntity{
			Type:   api.MessageEntityTypeBotCommand,
			Offset: 0,
			Length: int64(end),
		})
	}
	return api.Update{
		UpdateID: 1,
		Message: &api.Message{
			MessageID: 1,
			From:      &api.User{ID: userID},
			Chat:      api.Chat{ID: chatID, Type: api.ChatTypePrivate},
			Text:      text,
			Entities:  entities,
		},
	}
}

func makeCtx(bot *client.Bot, u *api.Update) *dispatch.Context {
	return dispatch.NewContext(context.Background(), bot, u)
}

// allowAny mocks unlimited sendMessage calls.
func allowAny(m *mockDoer) {
	m.On("Do", mock.Anything).Return(anyResp(), nil)
}

func TestConversation_NewBot_AsksName(t *testing.T) {
	m := &mockDoer{}
	allowAny(m)
	bot := client.New("test:token", client.WithHTTPClient(m))

	store := conversation.NewMemoryStorage()
	conv := buildConv(store)
	mw := conv.Dispatch(func(c *dispatch.Context, u *api.Update) error { return nil })

	u := msgUpd(42, 1, "/newbot")
	require.NoError(t, mw(makeCtx(bot, &u), &u))

	state, err := store.Get(context.Background(), "uc:1:42")
	require.NoError(t, err)
	require.Equal(t, conversation.State("await_name"), state)
}

func TestConversation_NewBot_StoresName_AsksDesc(t *testing.T) {
	m := &mockDoer{}
	allowAny(m)
	bot := client.New("test:token", client.WithHTTPClient(m))

	store := conversation.NewMemoryStorage()
	conv := buildConv(store)
	mw := conv.Dispatch(func(c *dispatch.Context, u *api.Update) error { return nil })

	// Enter conversation.
	u1 := msgUpd(42, 1, "/newbot")
	require.NoError(t, mw(makeCtx(bot, &u1), &u1))

	// Reply with name — should advance to await_desc.
	u2 := msgUpd(42, 1, "MyBot")
	require.NoError(t, mw(makeCtx(bot, &u2), &u2))

	state, err := store.Get(context.Background(), "uc:1:42")
	require.NoError(t, err)
	require.Equal(t, conversation.State("await_desc"), state)
}

func TestConversation_Cancel_EndsConversation(t *testing.T) {
	m := &mockDoer{}
	allowAny(m)
	bot := client.New("test:token", client.WithHTTPClient(m))

	store := conversation.NewMemoryStorage()
	conv := buildConv(store)
	mw := conv.Dispatch(func(c *dispatch.Context, u *api.Update) error { return nil })

	// Enter conversation.
	u1 := msgUpd(42, 1, "/newbot")
	require.NoError(t, mw(makeCtx(bot, &u1), &u1))

	// Cancel mid-flow.
	u2 := msgUpd(42, 1, "/cancel")
	require.NoError(t, mw(makeCtx(bot, &u2), &u2))

	_, err := store.Get(context.Background(), "uc:1:42")
	require.ErrorIs(t, err, conversation.ErrKeyNotFound, "cancel must clear state")
}
