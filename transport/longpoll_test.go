package transport

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lukaszraczylo/go-telegram/client"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockDoer struct{ mock.Mock }

func (m *mockDoer) Do(r *http.Request) (*http.Response, error) {
	args := m.Called(r)
	if v := args.Get(0); v != nil {
		return v.(*http.Response), args.Error(1)
	}
	return nil, args.Error(1)
}

func resp(body string) *http.Response {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}

func TestLongPoller_DeliversUpdatesAndAdvancesOffset(t *testing.T) {
	m := &mockDoer{}
	m.On("Do", mock.Anything).Return(
		resp(`{"ok":true,"result":[{"update_id":10,"message":{"message_id":1,"date":0,"chat":{"id":1,"type":"private"},"text":"hi"}}]}`),
		nil,
	).Once()
	m.On("Do", mock.Anything).Return(
		resp(`{"ok":true,"result":[{"update_id":11,"message":{"message_id":2,"date":0,"chat":{"id":1,"type":"private"},"text":"there"}}]}`),
		nil,
	).Once()
	m.On("Do", mock.Anything).Return(
		resp(`{"ok":true,"result":[]}`),
		nil,
	).Maybe()

	b := client.New("t", client.WithHTTPClient(m))
	p := NewLongPoller(b)
	p.Timeout = 0

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	go func() { _ = p.Run(ctx) }()

	u1 := <-p.Updates()
	require.Equal(t, int64(10), u1.UpdateID)
	u2 := <-p.Updates()
	require.Equal(t, int64(11), u2.UpdateID)
}

func TestLongPoller_BackoffOnNetworkError(t *testing.T) {
	m := &mockDoer{}
	var attempts atomic.Int32
	m.On("Do", mock.Anything).Run(func(args mock.Arguments) {
		attempts.Add(1)
	}).Return(nil, io.ErrUnexpectedEOF).Maybe()

	b := client.New("t", client.WithHTTPClient(m))
	p := NewLongPoller(b)
	p.Timeout = 0
	p.Backoff = &ExponentialBackoff{Base: 5 * time.Millisecond, Max: 5 * time.Millisecond}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_ = p.Run(ctx)
	require.GreaterOrEqual(t, attempts.Load(), int32(2), "should retry at least once")
}

func TestLongPoller_StopCloses(t *testing.T) {
	m := &mockDoer{}
	m.On("Do", mock.Anything).Return(resp(`{"ok":true,"result":[]}`), nil).Maybe()

	b := client.New("t", client.WithHTTPClient(m))
	p := NewLongPoller(b)
	p.Timeout = 0

	ctx := context.Background()
	done := make(chan struct{})
	go func() { _ = p.Run(ctx); close(done) }()

	require.NoError(t, p.Stop(ctx))
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not exit after Stop")
	}

	// Channel must be closed.
	_, ok := <-p.Updates()
	require.False(t, ok, "expected closed channel after Stop")
}
func TestLongPoller_HonoursRetryAfterOn429(t *testing.T) {
	m := &mockDoer{}
	var requestTimes []time.Time
	var mu sync.Mutex

	record := func(args mock.Arguments) {
		mu.Lock()
		requestTimes = append(requestTimes, time.Now())
		mu.Unlock()
	}

	// First call: 429 with retry_after=1.
	m.On("Do", mock.Anything).
		Run(record).
		Return(resp(`{"ok":false,"error_code":429,"description":"Too Many Requests","parameters":{"retry_after":1}}`), nil).
		Once()
	// Subsequent calls: empty success.
	m.On("Do", mock.Anything).
		Run(record).
		Return(resp(`{"ok":true,"result":[]}`), nil).
		Maybe()

	b := client.New("t", client.WithHTTPClient(m))
	p := NewLongPoller(b)
	p.Timeout = 0
	// Backoff base is huge so if it were used we'd see >>1s delay.
	p.Backoff = &ExponentialBackoff{Base: 10 * time.Second, Max: 30 * time.Second}

	ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
	defer cancel()
	_ = p.Run(ctx)

	mu.Lock()
	defer mu.Unlock()
	require.GreaterOrEqual(t, len(requestTimes), 2, "expected at least 2 requests")
	gap := requestTimes[1].Sub(requestTimes[0])
	require.GreaterOrEqual(t, gap, 900*time.Millisecond, "should have waited ~1s per retry_after, got %v", gap)
	require.Less(t, gap, 3*time.Second, "should not have waited backoff base (10s), got %v", gap)
}
