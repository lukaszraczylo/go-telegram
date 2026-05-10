package dispatch

import (
	"context"
	"testing"

	"github.com/lukaszraczylo/go-telegram/api"
	"github.com/lukaszraczylo/go-telegram/client"
)

func BenchmarkRouter_DispatchCommand(b *testing.B) {
	r := New(client.New("t"))
	r.OnCommand("/start", func(c *Context, m *api.Message) error { return nil })
	u := cmdMessage("/start hello")
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		c := NewContext(ctx, r.bot, &u)
		_ = r.dispatch(c, &u)
	}
}

func BenchmarkRouter_DispatchTextRegex(b *testing.B) {
	r := New(client.New("t"))
	r.OnText("^hello.*", func(c *Context, m *api.Message) error { return nil })
	u := api.Update{
		UpdateID: 1,
		Message: &api.Message{
			MessageID: 1, Date: 0,
			Chat: api.Chat{ID: 1, Type: api.ChatTypePrivate},
			Text: "hello world",
		},
	}
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		c := NewContext(ctx, r.bot, &u)
		_ = r.dispatch(c, &u)
	}
}

func BenchmarkRouter_DispatchFilter(b *testing.B) {
	r := New(client.New("t"))
	r.OnMessageFilter(
		func(m *api.Message) bool { return m != nil && m.Text == "ping" },
		func(c *Context, m *api.Message) error { return nil },
	)
	u := api.Update{
		UpdateID: 1,
		Message: &api.Message{
			MessageID: 1, Date: 0,
			Chat: api.Chat{ID: 1, Type: api.ChatTypePrivate},
			Text: "ping",
		},
	}
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		c := NewContext(ctx, r.bot, &u)
		_ = r.dispatch(c, &u)
	}
}

func BenchmarkRouter_NewContext(b *testing.B) {
	bot := client.New("t")
	u := &api.Update{UpdateID: 1}
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		_ = NewContext(ctx, bot, u)
	}
}

func BenchmarkExtractCommand(b *testing.B) {
	text := "/start@BotName hello world"
	cmdLen := len("/start@BotName")
	m := &api.Message{
		MessageID: 1, Date: 0,
		Chat: api.Chat{ID: 1, Type: api.ChatTypePrivate},
		Text: text,
		Entities: []api.MessageEntity{
			{Type: api.MessageEntityTypeBotCommand, Offset: 0, Length: int64(cmdLen)},
		},
	}
	b.ReportAllocs()
	for b.Loop() {
		_, _, _ = extractCommand(m)
	}
}
