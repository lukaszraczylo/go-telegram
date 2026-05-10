// Large-Update unmarshal benchmarks: decode a realistic Update with text +
// entities + a 2x3 inline keyboard + a 3-size photo array. Stresses each
// library's union/discriminator decoding (entities, reply markup variants).
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

var largeUpdateBytes = []byte(shared.LargeUpdateJSON)

func BenchmarkLargeUnmarshal_ours(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		var u api.Update
		if err := json.Unmarshal(largeUpdateBytes, &u); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLargeUnmarshal_gotba(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		var u tgbotapi.Update
		if err := json.Unmarshal(largeUpdateBytes, &u); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLargeUnmarshal_telebot(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		var u tele.Update
		if err := json.Unmarshal(largeUpdateBytes, &u); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLargeUnmarshal_gobot(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		var u gobotmodels.Update
		if err := json.Unmarshal(largeUpdateBytes, &u); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLargeUnmarshal_telego(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		var u telego.Update
		if err := json.Unmarshal(largeUpdateBytes, &u); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLargeUnmarshal_echotron(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		var u echotron.Update
		if err := json.Unmarshal(largeUpdateBytes, &u); err != nil {
			b.Fatal(err)
		}
	}
}
