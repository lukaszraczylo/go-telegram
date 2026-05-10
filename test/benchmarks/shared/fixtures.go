// Package shared holds JSON fixtures and an httptest mock server reused by
// every per-library benchmark. Keeping fixtures here guarantees that all
// libraries decode the same bytes and that round-trip benches hit the same
// canned response.
package shared

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
)

// SmallUpdateJSON is a minimal text-message update — what a typical bot sees
// most often. Used by the dispatcher and webhook benches.
const SmallUpdateJSON = `{
  "update_id": 123456789,
  "message": {
    "message_id": 1,
    "date": 1715000000,
    "chat": {"id": 42, "type": "private", "first_name": "Alice"},
    "from": {"id": 42, "is_bot": false, "first_name": "Alice", "language_code": "en"},
    "text": "/start"
  }
}`

// LargeUpdateJSON exercises union/discriminator decoding: text + entities,
// reply markup with an inline keyboard, and a 3-size photo array.
const LargeUpdateJSON = `{
  "update_id": 987654321,
  "message": {
    "message_id": 17,
    "date": 1715000123,
    "chat": {"id": -100123456789, "type": "supergroup", "title": "Devs"},
    "from": {"id": 42, "is_bot": false, "first_name": "Alice", "username": "alice"},
    "text": "see https://example.com and @bob too",
    "entities": [
      {"type": "url", "offset": 4, "length": 19},
      {"type": "mention", "offset": 28, "length": 4},
      {"type": "bold", "offset": 0, "length": 3}
    ],
    "reply_markup": {
      "inline_keyboard": [
        [{"text": "ok", "callback_data": "ok:1"}, {"text": "no", "callback_data": "no:1"}, {"text": "more", "callback_data": "more:1"}],
        [{"text": "left", "callback_data": "p:l"}, {"text": "right", "callback_data": "p:r"}]
      ]
    },
    "photo": [
      {"file_id": "AgAD1", "file_unique_id": "u1", "width": 90, "height": 67, "file_size": 1234},
      {"file_id": "AgAD2", "file_unique_id": "u2", "width": 320, "height": 240, "file_size": 12345},
      {"file_id": "AgAD3", "file_unique_id": "u3", "width": 800, "height": 600, "file_size": 123456}
    ]
  }
}`

// SendMessageOKResponse is the canned `{"ok":true,"result":Message}` body
// returned by the mock server for SendMessage round-trips.
const SendMessageOKResponse = `{"ok":true,"result":{"message_id":1,"date":1715000000,"chat":{"id":42,"type":"private","first_name":"Alice"},"from":{"id":7,"is_bot":true,"first_name":"Bot","username":"benchbot"},"text":"hello"}}`

// GetMeOKResponse is the canned getMe reply some libraries call eagerly during
// constructor (telebot, echotron). Lets us avoid a real network hop in setup.
const GetMeOKResponse = `{"ok":true,"result":{"id":7,"is_bot":true,"first_name":"Bot","username":"benchbot","can_join_groups":true,"can_read_all_group_messages":false,"supports_inline_queries":false}}`

// NewMockServer returns an httptest.Server that responds to every Bot API
// path with a canned `ok:true` body chosen by suffix:
//   - /sendMessage → SendMessageOKResponse
//   - /getMe       → GetMeOKResponse
//   - anything else → SendMessageOKResponse (safe default for benches)
//
// Caller is responsible for Close().
func NewMockServer() *httptest.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Drain body so the client sees a complete request/response cycle.
		_, _ = io.Copy(io.Discard, r.Body)
		_ = r.Body.Close()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/getMe"):
			_, _ = io.WriteString(w, GetMeOKResponse)
		default:
			_, _ = io.WriteString(w, SendMessageOKResponse)
		}
	})
	return httptest.NewServer(handler)
}
