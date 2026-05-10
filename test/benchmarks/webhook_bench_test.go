// Webhook decode benchmarks: parse a small text-message Update from a fixed
// JSON payload using each library's typed Update struct. Pure CPU — no
// network. Stresses the JSON codec each library ships with by default.
package benchmarks

import (
	"encoding/json"
	"testing"

	"github.com/lukaszraczylo/go-telegram/api"
	"github.com/lukaszraczylo/go-telegram/test/benchmarks/shared"

	echotron "github.com/NicoNex/echotron/v3"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	gobotmodels "github.com/go-telegram/bot/models"
	telego "github.com/mymmrac/telego"
	tele "gopkg.in/telebot.v3"
)

var smallUpdateBytes = []byte(shared.SmallUpdateJSON)

func BenchmarkWebhook_ours(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		var u api.Update
		if err := json.Unmarshal(smallUpdateBytes, &u); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWebhook_gotba(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		var u tgbotapi.Update
		if err := json.Unmarshal(smallUpdateBytes, &u); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWebhook_telebot(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		var u tele.Update
		if err := json.Unmarshal(smallUpdateBytes, &u); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWebhook_gobot(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		var u gobotmodels.Update
		if err := json.Unmarshal(smallUpdateBytes, &u); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWebhook_telego(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		var u telego.Update
		if err := json.Unmarshal(smallUpdateBytes, &u); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWebhook_echotron(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		var u echotron.Update
		if err := json.Unmarshal(smallUpdateBytes, &u); err != nil {
			b.Fatal(err)
		}
	}
}
