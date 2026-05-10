// Round-trip benchmarks: build a SendMessage request, POST it to a local
// httptest.Server returning a canned `{"ok":true,"result":Message}` body,
// decode the response. Measures marshal + transport + unmarshal end-to-end.
//
// Each library's idiomatic "send a text message" call path is exercised
// through its public API. The mock server replies identically for every path,
// so any difference comes from serialization, HTTP plumbing, or response
// decoding inside each library.
package benchmarks

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/lukaszraczylo/go-telegram/api"
	"github.com/lukaszraczylo/go-telegram/client"
	"github.com/lukaszraczylo/go-telegram/test/benchmarks/shared"

	echotron "github.com/NicoNex/echotron/v3"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	gobot "github.com/go-telegram/bot"
	telego "github.com/mymmrac/telego"
	tele "gopkg.in/telebot.v3"
)

// Telegram-format token (digits:[\w-]{35}). telego enforces this format on construction.
const benchToken = "1234567890:ABCDEFGHIJKLMNOPQRSTUVWXYZ_ab123456"

// BenchmarkCall_ours — lukaszraczylo/go-telegram.
func BenchmarkCall_ours(b *testing.B) {
	srv := shared.NewMockServer()
	defer srv.Close()
	bot := client.New(benchToken, client.WithBaseURL(srv.URL))
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, err := api.SendMessage(ctx, bot, &api.SendMessageParams{
			ChatID: api.ChatIDFromInt(42),
			Text:   "hello",
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCall_gotba — go-telegram-bot-api/telegram-bot-api/v5.
func BenchmarkCall_gotba(b *testing.B) {
	srv := shared.NewMockServer()
	defer srv.Close()
	// Endpoint is sprintf'd with token + method.
	endpoint := srv.URL + "/bot%s/%s"
	bot, err := tgbotapi.NewBotAPIWithClient(benchToken, endpoint, &http.Client{})
	if err != nil {
		b.Fatal(err)
	}
	msg := tgbotapi.NewMessage(42, "hello")
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := bot.Send(msg); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCall_telebot — gopkg.in/telebot.v3 (tucnak).
func BenchmarkCall_telebot(b *testing.B) {
	srv := shared.NewMockServer()
	defer srv.Close()
	bot, err := tele.NewBot(tele.Settings{
		Token:       benchToken,
		URL:         srv.URL,
		Synchronous: true,
		Offline:     true, // skip eager getMe call; we test sending
	})
	if err != nil {
		b.Fatal(err)
	}
	chat := &tele.Chat{ID: 42}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := bot.Send(chat, "hello"); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCall_gobot — go-telegram/bot.
func BenchmarkCall_gobot(b *testing.B) {
	srv := shared.NewMockServer()
	defer srv.Close()
	bot, err := gobot.New(benchToken,
		gobot.WithServerURL(srv.URL),
		gobot.WithSkipGetMe(),
	)
	if err != nil {
		b.Fatal(err)
	}
	ctx := context.Background()
	params := &gobot.SendMessageParams{ChatID: int64(42), Text: "hello"}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := bot.SendMessage(ctx, params); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCall_telego — mymmrac/telego.
func BenchmarkCall_telego(b *testing.B) {
	srv := shared.NewMockServer()
	defer srv.Close()
	bot, err := telego.NewBot(benchToken, telego.WithAPIServer(srv.URL))
	if err != nil {
		b.Fatal(err)
	}
	ctx := context.Background()
	params := &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: 42},
		Text:   "hello",
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := bot.SendMessage(ctx, params); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCall_echotron — NicoNex/echotron/v3.
//
// echotron expects a base URL ending in /bot<token>/ and ships built-in
// dual-level rate limiting (global 30/s, per-chat 20/min) on its unexported
// lclient field. The setters (SetGlobalRequestLimit / SetChatRequestLimit)
// are methods on the unexported type and have no public accessor through
// the API value, so the rate limiter cannot be disabled from outside the
// package without monkey-patching.
//
// Running this bench against the real path produces ~3s/op driven entirely
// by the per-chat token bucket — measuring rate limiting, not the library.
// We skip rather than publish a misleading number; the rate limiter is a
// feature of echotron and is documented as a caveat in the report.
func BenchmarkCall_echotron(b *testing.B) {
	b.Skip("echotron has built-in rate limiting that cannot be disabled via the public API; see comment")
	srv := shared.NewMockServer()
	defer srv.Close()
	base := srv.URL + "/bot" + benchToken + "/"
	api := echotron.CustomAPI(base, benchToken)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := api.SendMessage("hello", 42, nil); err != nil {
			b.Fatal(err)
		}
	}
}

// silence unused-import if a build tag strips a lib.
var _ = strings.NewReader
