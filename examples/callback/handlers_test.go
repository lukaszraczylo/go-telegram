package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/lukaszraczylo/go-telegram/api"
	"github.com/lukaszraczylo/go-telegram/client"
	"github.com/lukaszraczylo/go-telegram/dispatch"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockDoer struct{ mock.Mock }

func (m *mockDoer) Do(r *http.Request) (*http.Response, error) {
	args := m.Called(r)
	if v := args.Get(0); v != nil {
		return v.(*http.Response), args.Error(1)
	}
	return nil, args.Error(1)
}

func okResp(body string) *http.Response {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}

const (
	sendMsgResult  = `{"ok":true,"result":{"message_id":1,"date":0,"chat":{"id":42,"type":"private"}}}`
	editMsgResult  = `{"ok":true,"result":{"message_id":10,"date":0,"chat":{"id":42,"type":"private"}}}`
	answerCbResult = `{"ok":true,"result":true}`
)

func makeCtx(bot *client.Bot, upd *api.Update, extra map[string]any) *dispatch.Context {
	c := dispatch.NewContext(context.Background(), bot, upd)
	for k, v := range extra {
		c.Set(k, v)
	}
	return c
}

// --- handleStart ---

func TestHandleStart_SendsInitialKeyboard(t *testing.T) {
	m := &mockDoer{}
	m.On("Do", mock.MatchedBy(func(r *http.Request) bool {
		if !strings.HasSuffix(r.URL.Path, "/sendMessage") {
			return false
		}
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(r.Body)
		body := buf.String()
		// Counter starts at 0; keyboard must contain both buttons
		return strings.Contains(body, `"Counter: 0"`) &&
			strings.Contains(body, `"reply_markup"`) &&
			strings.Contains(body, `"count:0:dec"`) &&
			strings.Contains(body, `"count:0:inc"`)
	})).Return(okResp(sendMsgResult), nil)

	bot := client.New("test:token", client.WithHTTPClient(m))
	msg := &api.Message{
		MessageID: 1,
		Chat:      api.Chat{ID: 42, Type: api.ChatTypePrivate},
		From:      &api.User{ID: 7, FirstName: "Alice"},
		Text:      "/start",
	}
	upd := &api.Update{UpdateID: 1, Message: msg}

	require.NoError(t, handleStart(makeCtx(bot, upd, nil), msg))
	m.AssertExpectations(t)
}

// --- handleCallback ---

func callbackCtx(bot *client.Bot, q *api.CallbackQuery, groups []string) *dispatch.Context {
	upd := &api.Update{UpdateID: 1, CallbackQuery: q}
	c := makeCtx(bot, upd, nil)
	c.RegexMatch = groups
	return c
}

func callbackQuery(data string, msgID int64, chatID int64) *api.CallbackQuery {
	msg := &api.Message{
		MessageID: msgID,
		Chat:      api.Chat{ID: chatID, Type: api.ChatTypePrivate},
	}
	return &api.CallbackQuery{
		ID:      "cb1",
		From:    api.User{ID: 7},
		Message: msg,
		Data:    data,
	}
}

func TestHandleCallback_Increments(t *testing.T) {
	m := &mockDoer{}
	// AnswerCallbackQuery
	m.On("Do", mock.MatchedBy(func(r *http.Request) bool {
		return strings.HasSuffix(r.URL.Path, "/answerCallbackQuery")
	})).Return(okResp(answerCbResult), nil)
	// EditMessageText — counter must show 6
	m.On("Do", mock.MatchedBy(func(r *http.Request) bool {
		if !strings.HasSuffix(r.URL.Path, "/editMessageText") {
			return false
		}
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(r.Body)
		return strings.Contains(buf.String(), `"Counter: 6"`)
	})).Return(okResp(editMsgResult), nil)

	bot := client.New("test:token", client.WithHTTPClient(m))
	q := callbackQuery("count:5:inc", 10, 42)
	// groups: [full_match, "5", "inc"]
	c := callbackCtx(bot, q, []string{"count:5:inc", "5", "inc"})

	require.NoError(t, handleCallback(c, q))
	m.AssertExpectations(t)
}

func TestHandleCallback_Decrements(t *testing.T) {
	m := &mockDoer{}
	m.On("Do", mock.MatchedBy(func(r *http.Request) bool {
		return strings.HasSuffix(r.URL.Path, "/answerCallbackQuery")
	})).Return(okResp(answerCbResult), nil)
	m.On("Do", mock.MatchedBy(func(r *http.Request) bool {
		if !strings.HasSuffix(r.URL.Path, "/editMessageText") {
			return false
		}
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(r.Body)
		return strings.Contains(buf.String(), `"Counter: 4"`)
	})).Return(okResp(editMsgResult), nil)

	bot := client.New("test:token", client.WithHTTPClient(m))
	q := callbackQuery("count:5:dec", 10, 42)
	c := callbackCtx(bot, q, []string{"count:5:dec", "5", "dec"})

	require.NoError(t, handleCallback(c, q))
	m.AssertExpectations(t)
}
