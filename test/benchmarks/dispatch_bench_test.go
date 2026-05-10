// Dispatcher routing benchmarks: register 20 handlers across each library's
// dispatcher, feed an update that matches the LAST-registered handler, and
// measure cost per dispatch. Worst-case filter chain traversal.
//
// Coverage notes (see results/raw.txt and report for the full caveats):
//
//   - ours, telebot, gobot expose a synchronous single-update entry point
//     (Process / ProcessUpdate). Bench measures that path directly.
//   - gotba (go-telegram-bot-api/v5) ships no built-in dispatcher; users
//     route via a manual switch on Update fields. Skipped here — would be
//     comparing our framework against a hand-written switch.
//   - telego routes via a buffered channel + goroutine pool inside
//     telegohandler.BotHandler. There is no public sync entry, so the bench
//     would conflate channel + goroutine overhead with routing cost. Skipped.
//   - echotron uses a chat-ID-keyed Dispatcher that fans out to per-chat Bot
//     instances; it's a different paradigm (stateful per-chat bot loop), so
//     not directly comparable to "match this update against N handlers".
package benchmarks

import (
	"context"
	"fmt"
	"testing"

	"github.com/lukaszraczylo/go-telegram/api"
	"github.com/lukaszraczylo/go-telegram/client"
	"github.com/lukaszraczylo/go-telegram/dispatch"

	gobot "github.com/go-telegram/bot"
	gobotmodels "github.com/go-telegram/bot/models"
	tele "gopkg.in/telebot.v3"
)

const dispatchN = 20

// matchCmd is the command the LAST-registered handler matches.
const matchCmd = "/cmd19"

func BenchmarkDispatch_ours(b *testing.B) {
	r := dispatch.New(client.New(benchToken))
	noop := func(c *dispatch.Context, m *api.Message) error { return nil }
	for i := 0; i < dispatchN; i++ {
		r.OnCommand(fmt.Sprintf("/cmd%d", i), noop)
	}
	u := &api.Update{
		UpdateID: 1,
		Message: &api.Message{
			MessageID: 1,
			Date:      0,
			Chat:      api.Chat{ID: 42, Type: api.ChatTypePrivate},
			Text:      matchCmd,
			Entities: []api.MessageEntity{
				{Type: api.MessageEntityTypeBotCommand, Offset: 0, Length: int64(len(matchCmd))},
			},
		},
	}
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if err := r.Process(ctx, u); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDispatch_telebot(b *testing.B) {
	bot, err := tele.NewBot(tele.Settings{Token: benchToken, Synchronous: true, Offline: true})
	if err != nil {
		b.Fatal(err)
	}
	noop := func(c tele.Context) error { return nil }
	for i := 0; i < dispatchN; i++ {
		bot.Handle(fmt.Sprintf("/cmd%d", i), noop)
	}
	u := tele.Update{
		ID: 1,
		Message: &tele.Message{
			ID:     1,
			Chat:   &tele.Chat{ID: 42, Type: tele.ChatPrivate},
			Sender: &tele.User{ID: 42},
			Text:   matchCmd,
			Entities: []tele.MessageEntity{
				{Type: tele.EntityCommand, Offset: 0, Length: len(matchCmd)},
			},
		},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		bot.ProcessUpdate(u)
	}
}

func BenchmarkDispatch_gobot(b *testing.B) {
	srvless := func(ctx context.Context, b *gobot.Bot, update *gobotmodels.Update) {}
	bot, err := gobot.New(benchToken, gobot.WithSkipGetMe(), gobot.WithDefaultHandler(srvless))
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < dispatchN; i++ {
		bot.RegisterHandler(gobot.HandlerTypeMessageText, fmt.Sprintf("/cmd%d", i), gobot.MatchTypeExact, srvless)
	}
	u := &gobotmodels.Update{
		ID: 1,
		Message: &gobotmodels.Message{
			ID:   1,
			Chat: gobotmodels.Chat{ID: 42, Type: gobotmodels.ChatTypePrivate},
			Text: matchCmd,
		},
	}
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		bot.ProcessUpdate(ctx, u)
	}
}
