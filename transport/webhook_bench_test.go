package transport

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lukaszraczylo/go-telegram/client"
)

const benchUpdateBody = `{"update_id":12345,"message":{"message_id":1,"date":1700000000,"chat":{"id":42,"type":"private"},"from":{"id":42,"is_bot":false,"first_name":"User"},"text":"hello world"}}`

func BenchmarkWebhook_ServeHTTP(b *testing.B) {
	w := NewWebhookServer(client.New("t"), WithBufferSize(1024))
	body := []byte(benchUpdateBody)

	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-w.Updates():
			case <-done:
				return
			}
		}
	}()
	defer close(done)

	b.ReportAllocs()
	for b.Loop() {
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		w.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("status %d", rec.Code)
		}
	}
}
