package message_test

import (
	"testing"

	"github.com/lukaszraczylo/go-telegram/api"
	msgfilter "github.com/lukaszraczylo/go-telegram/dispatch/filters/message"
	"github.com/stretchr/testify/require"
)

func msg(text string) *api.Message {
	return &api.Message{
		MessageID: 1,
		Chat:      api.Chat{ID: 1, Type: api.ChatTypePrivate},
		Text:      text,
	}
}

func cmdMsg(cmd string) *api.Message {
	text := cmd
	return &api.Message{
		MessageID: 1,
		Chat:      api.Chat{ID: 1, Type: api.ChatTypePrivate},
		Text:      text,
		Entities: []api.MessageEntity{
			{Type: api.MessageEntityTypeBotCommand, Offset: 0, Length: int64(len([]rune(text)))},
		},
	}
}

func TestText(t *testing.T) {
	f := msgfilter.Text(`^hello`)
	require.True(t, f(msg("hello world")))
	require.False(t, f(msg("world hello")))
	require.False(t, f(nil))
}

func TestText_PanicsOnBadPattern(t *testing.T) {
	require.Panics(t, func() { msgfilter.Text(`[invalid`) })
}

func TestTextEquals(t *testing.T) {
	f := msgfilter.TextEquals("hi")
	require.True(t, f(msg("hi")))
	require.False(t, f(msg("hi there")))
	require.False(t, f(nil))
}

func TestTextPrefix(t *testing.T) {
	f := msgfilter.TextPrefix("/start")
	require.True(t, f(msg("/start now")))
	require.False(t, f(msg("no prefix")))
	require.False(t, f(nil))
}

func TestTextContains(t *testing.T) {
	f := msgfilter.TextContains("bot")
	require.True(t, f(msg("my bot is cool")))
	require.False(t, f(msg("nothing here")))
	require.False(t, f(nil))
}

func TestCommand(t *testing.T) {
	t.Run("matches exact command", func(t *testing.T) {
		f := msgfilter.Command("/start")
		require.True(t, f(cmdMsg("/start")))
	})
	t.Run("matches without leading slash", func(t *testing.T) {
		f := msgfilter.Command("start")
		require.True(t, f(cmdMsg("/start")))
	})
	t.Run("strips BotName suffix", func(t *testing.T) {
		m := &api.Message{
			Text:     "/start@MyBot",
			Entities: []api.MessageEntity{{Type: api.MessageEntityTypeBotCommand, Offset: 0, Length: 12}},
		}
		f := msgfilter.Command("/start")
		require.True(t, f(m))
	})
	t.Run("no match different command", func(t *testing.T) {
		f := msgfilter.Command("/stop")
		require.False(t, f(cmdMsg("/start")))
	})
	t.Run("nil message", func(t *testing.T) {
		require.False(t, msgfilter.Command("/start")(nil))
	})
	t.Run("no entities", func(t *testing.T) {
		require.False(t, msgfilter.Command("/start")(msg("/start")))
	})
}

func TestAnyCommand(t *testing.T) {
	f := msgfilter.AnyCommand()
	require.True(t, f(cmdMsg("/anything")))
	require.False(t, f(msg("plain text")))
	require.False(t, f(nil))
}

func TestIsReply(t *testing.T) {
	f := msgfilter.IsReply()
	m := msg("reply")
	m.ReplyToMessage = &api.Message{MessageID: 2}
	require.True(t, f(m))
	require.False(t, f(msg("no reply")))
	require.False(t, f(nil))
}

func TestIsForward(t *testing.T) {
	// ForwardOrigin is a MessageOrigin interface; set via a concrete type.
	f := msgfilter.IsForward()
	m := msg("fwd")
	m.ForwardOrigin = &api.MessageOriginUser{Type: "user"}
	require.True(t, f(m))
	require.False(t, f(msg("no fwd")))
	require.False(t, f(nil))
}

func TestHasPhoto(t *testing.T) {
	f := msgfilter.HasPhoto()
	m := msg("")
	m.Photo = []api.PhotoSize{{FileID: "x", Width: 100, Height: 100}}
	require.True(t, f(m))
	require.False(t, f(msg("no photo")))
	require.False(t, f(nil))
}

func TestHasDocument(t *testing.T) {
	f := msgfilter.HasDocument()
	m := msg("")
	m.Document = &api.Document{FileID: "doc1"}
	require.True(t, f(m))
	require.False(t, f(msg("no doc")))
	require.False(t, f(nil))
}

func TestHasEntity(t *testing.T) {
	f := msgfilter.HasEntity(api.MessageEntityTypeURL)
	m := msg("check https://example.com")
	m.Entities = []api.MessageEntity{{Type: api.MessageEntityTypeURL, Offset: 6, Length: 19}}
	require.True(t, f(m))
	require.False(t, f(msg("plain")))
	require.False(t, f(nil))
}

func TestChatType(t *testing.T) {
	f := msgfilter.ChatType(api.ChatTypePrivate)
	private := msg("hi")
	require.True(t, f(private))

	group := msg("hi")
	group.Chat.Type = api.ChatTypeGroup
	require.False(t, f(group))
	require.False(t, f(nil))
}

func TestFromUser(t *testing.T) {
	f := msgfilter.FromUser(42)
	m := msg("hi")
	m.From = &api.User{ID: 42}
	require.True(t, f(m))

	m2 := msg("hi")
	m2.From = &api.User{ID: 99}
	require.False(t, f(m2))

	require.False(t, f(msg("no from")))
	require.False(t, f(nil))
}

func TestInChat(t *testing.T) {
	f := msgfilter.InChat(1)
	require.True(t, f(msg("hi")))
	m2 := msg("hi")
	m2.Chat.ID = 2
	require.False(t, f(m2))
	require.False(t, f(nil))
}

func TestComposedMessageFilters(t *testing.T) {
	// private chat AND contains "hello"
	f := msgfilter.ChatType(api.ChatTypePrivate).And(msgfilter.TextContains("hello"))
	m := msg("say hello")
	require.True(t, f(m))

	m2 := msg("say hello")
	m2.Chat.Type = api.ChatTypeGroup
	require.False(t, f(m2))
}
