package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFastHTTPDoer_BasicRoundTrip(t *testing.T) {
	got := make(chan struct{ method, ct, body string }, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got <- struct{ method, ct, body string }{r.Method, r.Header.Get("Content-Type"), string(body)}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ok":true,"result":42}`)
	}))
	defer srv.Close()

	d := NewFastHTTPDoer()
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/sendMessage", strings.NewReader(`{"chat_id":1,"text":"hi"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.Background())

	resp, err := d.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		t.Fatalf("status: got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != `{"ok":true,"result":42}` {
		t.Fatalf("body: got %q", body)
	}

	rec := <-got
	if rec.method != http.MethodPost {
		t.Fatalf("method: got %q", rec.method)
	}
	if rec.ct != "application/json" {
		t.Fatalf("content-type: got %q", rec.ct)
	}
	if rec.body != `{"chat_id":1,"text":"hi"}` {
		t.Fatalf("body: got %q", rec.body)
	}
}

func TestFastHTTPDoer_HonoursContextDeadline(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		_, _ = io.WriteString(w, "ok")
	}))
	defer srv.Close()

	d := NewFastHTTPDoer(WithFastHTTPReadTimeout(time.Hour))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	req = req.WithContext(ctx)

	_, err := d.Do(req)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestFastHTTPDoer_IntegratesWithBot(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"ok":true,"result":{"message_id":7,"date":0,"text":"hi"}}`)
	}))
	defer srv.Close()

	bot := New("123:abc",
		WithBaseURL(srv.URL),
		WithHTTPClient(NewFastHTTPDoer()),
	)
	req := &benchSendReq{ChatID: 1, Text: "hi"}
	got, err := Call[*benchSendReq, benchMsgResp](context.Background(), bot, "sendMessage", req)
	if err != nil {
		t.Fatal(err)
	}
	if got.MessageID != 7 || got.Text != "hi" {
		t.Fatalf("got %+v", got)
	}
}
