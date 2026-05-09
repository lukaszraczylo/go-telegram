package transport

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lukaszraczylo/go-telegram/client"
	"github.com/stretchr/testify/require"
)

func TestWebhook_DeliversUpdate(t *testing.T) {
	b := client.New("t")
	w := NewWebhookServer(b)
	w.SecretToken = "secret"

	srv := httptest.NewServer(w)
	t.Cleanup(srv.Close)

	body := `{"update_id":1,"message":{"message_id":1,"date":0,"chat":{"id":1,"type":"private"},"text":"hi"}}`
	req, _ := http.NewRequest(http.MethodPost, srv.URL, strings.NewReader(body))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "secret")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	select {
	case u := <-w.Updates():
		require.Equal(t, int64(1), u.UpdateID)
	case <-time.After(time.Second):
		t.Fatal("update not delivered")
	}
}

func TestWebhook_RejectsBadSecret(t *testing.T) {
	b := client.New("t")
	w := NewWebhookServer(b)
	w.SecretToken = "secret"

	srv := httptest.NewServer(w)
	t.Cleanup(srv.Close)

	req, _ := http.NewRequest(http.MethodPost, srv.URL, strings.NewReader(`{}`))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "wrong")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestWebhook_RejectsNonPOST(t *testing.T) {
	w := NewWebhookServer(client.New("t"))
	srv := httptest.NewServer(w)
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL)
	require.NoError(t, err)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
}

func TestWebhook_RejectsBadJSON(t *testing.T) {
	w := NewWebhookServer(client.New("t"))
	srv := httptest.NewServer(w)
	t.Cleanup(srv.Close)

	resp, err := http.Post(srv.URL, "application/json", bytes.NewBufferString("not json"))
	require.NoError(t, err)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestWebhook_StopExitsRun(t *testing.T) {
	w := NewWebhookServer(client.New("t"))

	done := make(chan struct{})
	go func() { _ = w.Run(context.Background()); close(done) }()

	require.NoError(t, w.Stop(context.Background()))
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not exit after Stop")
	}
}

// TestWebhook_ConcurrentStopNoPanic fires many concurrent requests while
// simultaneously calling Stop, and asserts no panic (send on closed channel).
// Run under -race to verify mutex and WaitGroup correctness.
func TestWebhook_ConcurrentStopNoPanic(t *testing.T) {
	body := `{"update_id":1,"message":{"message_id":1,"date":0,"chat":{"id":1,"type":"private"},"text":"hi"}}`

	for range 20 {
		w := NewWebhookServer(client.New("t"), WithBufferSize(256))
		srv := httptest.NewServer(w)

		// Drain updates so ServeHTTP doesn't block on a full channel.
		go func() {
			for range w.Updates() {
			}
		}()

		// Run in background.
		go func() { _ = w.Run(context.Background()) }()

		// Fire concurrent requests.
		const goroutines = 20
		ready := make(chan struct{})
		var wg sync.WaitGroup
		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-ready
				for j := 0; j < 5; j++ {
					req, _ := http.NewRequest(http.MethodPost, srv.URL, strings.NewReader(body))
					resp, err := http.DefaultClient.Do(req)
					if err == nil {
						_ = resp.Body.Close()
					}
				}
			}()
		}

		close(ready)
		time.Sleep(5 * time.Millisecond) // let some requests land before Stop
		srv.Close()
		_ = w.Stop(context.Background())
		wg.Wait()
	}
}
