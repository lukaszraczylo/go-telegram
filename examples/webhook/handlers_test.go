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

const sendMsgResult = `{"ok":true,"result":{"message_id":1,"date":0,"chat":{"id":42,"type":"private"}}}`

func TestHandlePing_RepliesWithPong(t *testing.T) {
	m := &mockDoer{}
	m.On("Do", mock.MatchedBy(func(r *http.Request) bool {
		if !strings.HasSuffix(r.URL.Path, "/sendMessage") {
			return false
		}
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(r.Body)
		return strings.Contains(buf.String(), `"pong"`)
	})).Return(okResp(sendMsgResult), nil)

	bot := client.New("test:token", client.WithHTTPClient(m))
	msg := &api.Message{
		MessageID: 1,
		Chat:      api.Chat{ID: 42, Type: string(api.ChatTypePrivate)},
		From:      &api.User{ID: 7, FirstName: "Alice"},
		Text:      "/ping",
	}
	upd := &api.Update{UpdateID: 1, Message: msg}
	c := dispatch.NewContext(context.Background(), bot, upd)

	require.NoError(t, handlePing(c, msg))
	m.AssertExpectations(t)
}
