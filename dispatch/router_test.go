package dispatch

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lukaszraczylo/go-telegram/api"
	"github.com/lukaszraczylo/go-telegram/client"
	"github.com/stretchr/testify/require"
)

// fakeUpdater feeds a fixed slice of updates then closes.
type fakeUpdater struct{ ch chan api.Update }

func newFake(ups ...api.Update) *fakeUpdater {
	ch := make(chan api.Update, len(ups))
	for _, u := range ups {
		ch <- u
	}
	close(ch)
	return &fakeUpdater{ch: ch}
}

func (f *fakeUpdater) Updates() <-chan api.Update     { return f.ch }
func (f *fakeUpdater) Run(ctx context.Context) error  { <-ctx.Done(); return ctx.Err() }
func (f *fakeUpdater) Stop(ctx context.Context) error { return nil }

func cmdMessage(text string) api.Update {
	return api.Update{
		UpdateID: 1,
		Message: &api.Message{
			MessageID: 1, Date: 0, Chat: api.Chat{ID: 1, Type: api.ChatTypePrivate},
			Text:     text,
			Entities: []api.MessageEntity{{Type: api.MessageEntityTypeBotCommand, Offset: 0, Length: int64(indexEnd(text))}},
		},
	}
}

func indexEnd(text string) int {
	for i, r := range text {
		if r == ' ' {
			return i
		}
	}
	return len(text)
}

func TestRouter_OnCommandMatches(t *testing.T) {
	b := client.New("t")
	r := New(b)
	hit := make(chan string, 1)
	r.OnCommand("/start", func(c *Context, m *api.Message) error {
		hit <- c.Values["command"].(string)
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	go func() { _ = r.Run(ctx, newFake(cmdMessage("/start"))) }()

	require.Equal(t, "/start", <-hit)
}

func TestRouter_OnCommandStripsBotName(t *testing.T) {
	r := New(client.New("t"))
	hit := make(chan string, 1)
	r.OnCommand("/start", func(c *Context, m *api.Message) error {
		hit <- "matched"
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	go func() { _ = r.Run(ctx, newFake(cmdMessage("/start@MyBot hello"))) }()

	require.Equal(t, "matched", <-hit)
}

func TestRouter_OnText(t *testing.T) {
	r := New(client.New("t"))
	hit := make(chan []string, 1)
	r.OnText(`^hello (\w+)$`, func(c *Context, m *api.Message) error {
		hit <- c.Values["regex_match"].([]string)
		return nil
	})

	u := api.Update{UpdateID: 1, Message: &api.Message{
		MessageID: 1, Chat: api.Chat{ID: 1, Type: "private"}, Text: "hello world",
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	go func() { _ = r.Run(ctx, newFake(u)) }()

	subs := <-hit
	require.Equal(t, "world", subs[1])
}

func TestRouter_OnCallback(t *testing.T) {
	r := New(client.New("t"))
	hit := make(chan string, 1)
	r.OnCallback(`^like:(\d+)$`, func(c *Context, q *api.CallbackQuery) error {
		hit <- q.Data
		return nil
	})

	u := api.Update{UpdateID: 1, CallbackQuery: &api.CallbackQuery{
		ID: "x", From: api.User{ID: 1}, ChatInstance: "y", Data: "like:42",
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	go func() { _ = r.Run(ctx, newFake(u)) }()

	require.Equal(t, "like:42", <-hit)
}

func TestRouter_NoMatch(t *testing.T) {
	r := New(client.New("t"))
	called := false
	r.OnCommand("/start", func(c *Context, m *api.Message) error {
		called = true
		return nil
	})
	u := api.Update{UpdateID: 1, Message: &api.Message{Text: "no command"}}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_ = r.Run(ctx, newFake(u))
	require.False(t, called)
}

func TestRouter_PanicRecovery(t *testing.T) {
	r := New(client.New("t"))
	r.OnCommand("/boom", func(c *Context, m *api.Message) error {
		panic("kaboom")
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	// Should not propagate panic to Run.
	require.NotPanics(t, func() { _ = r.Run(ctx, newFake(cmdMessage("/boom"))) })
}

// TestRouter_NonASCIICommand verifies that UTF-16 entity offsets are used
// correctly when the command contains non-ASCII runes. "/старт" is 6 runes,
// each a BMP code point, so UTF-16 length == 6.
func TestRouter_NonASCIICommand(t *testing.T) {
	const text = "/старт аргумент"
	// "/старт" = 1 + 5 runes, all BMP → UTF-16 length 6
	const cmdU16Len = int64(6)
	u := api.Update{
		UpdateID: 1,
		Message: &api.Message{
			MessageID: 1,
			Chat:      api.Chat{ID: 1, Type: api.ChatTypePrivate},
			Text:      text,
			Entities: []api.MessageEntity{
				{Type: api.MessageEntityTypeBotCommand, Offset: 0, Length: cmdU16Len},
			},
		},
	}

	r := New(client.New("t"))
	hit := make(chan [2]string, 1)
	r.OnCommand("/старт", func(c *Context, m *api.Message) error {
		hit <- [2]string{
			c.Values["command"].(string),
			c.Values["command_args"].(string),
		}
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	go func() { _ = r.Run(ctx, newFake(u)) }()

	got := <-hit
	require.Equal(t, "/старт", got[0])
	require.Equal(t, "аргумент", got[1])
}

// TestRouter_CommandValuesNotLeakedOnNoMatch verifies that c.Values["command"]
// is not set when a command entity is present but no route matches, so a
// subsequent text handler doesn't see stale values.
func TestRouter_CommandValuesNotLeakedOnNoMatch(t *testing.T) {
	r := New(client.New("t"))
	// Register a text handler that should fire as fallback.
	leaked := make(chan bool, 1)
	r.OnText(`.*`, func(c *Context, m *api.Message) error {
		_, hasCmd := c.Values["command"]
		leaked <- hasCmd
		return nil
	})
	// No OnCommand registered, so the command entity won't match any route.
	u := api.Update{UpdateID: 1, Message: &api.Message{
		MessageID: 1, Chat: api.Chat{ID: 1, Type: "private"},
		Text:     "/unknown",
		Entities: []api.MessageEntity{{Type: api.MessageEntityTypeBotCommand, Offset: 0, Length: 8}},
	}}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	go func() { _ = r.Run(ctx, newFake(u)) }()

	require.False(t, <-leaked, "command value must not leak into text handler")
}

func TestRouter_MiddlewareOrder(t *testing.T) {
	r := New(client.New("t"))
	var order []string
	r.Use(func(next Handler[*api.Update]) Handler[*api.Update] {
		return func(c *Context, u *api.Update) error {
			order = append(order, "before-1")
			err := next(c, u)
			order = append(order, "after-1")
			return err
		}
	})
	r.Use(func(next Handler[*api.Update]) Handler[*api.Update] {
		return func(c *Context, u *api.Update) error {
			order = append(order, "before-2")
			err := next(c, u)
			order = append(order, "after-2")
			return err
		}
	})
	r.OnCommand("/x", func(c *Context, m *api.Message) error {
		order = append(order, "handler")
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_ = r.Run(ctx, newFake(cmdMessage("/x")))

	require.Equal(t,
		[]string{"before-1", "before-2", "handler", "after-2", "after-1"},
		order)
}
func TestRouter_OnChannelPost(t *testing.T) {
	r := New(client.New("t"))
	hit := make(chan int64, 1)
	r.OnChannelPost(func(c *Context, m *api.Message) error {
		hit <- m.MessageID
		return nil
	})

	u := api.Update{UpdateID: 1, ChannelPost: &api.Message{
		MessageID: 99, Chat: api.Chat{ID: -100, Type: api.ChatTypeChannel},
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	go func() { _ = r.Run(ctx, newFake(u)) }()

	require.Equal(t, int64(99), <-hit)
}

func TestRouter_RunsAllHandlersForEditedMessage(t *testing.T) {
	r := New(client.New("t"))
	var hits []string
	r.OnEditedMessage(func(c *Context, m *api.Message) error {
		hits = append(hits, "first")
		return nil
	})
	r.OnEditedMessage(func(c *Context, m *api.Message) error {
		hits = append(hits, "second")
		return nil
	})

	u := api.Update{UpdateID: 1, EditedMessage: &api.Message{
		MessageID: 1, Chat: api.Chat{ID: 1, Type: "private"}, Text: "edited",
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_ = r.Run(ctx, newFake(u))

	require.Equal(t, []string{"first", "second"}, hits)
}

// ---------------------------------------------------------------------------
// Filter-route tests
// ---------------------------------------------------------------------------

func TestRouter_OnMessageFilter_Matches(t *testing.T) {
	r := New(client.New("t"))
	hit := make(chan string, 1)
	r.OnMessageFilter(
		Filter[*api.Message](func(m *api.Message) bool { return m != nil && m.Text == "ping" }),
		func(c *Context, m *api.Message) error { hit <- m.Text; return nil },
	)

	u := api.Update{UpdateID: 1, Message: &api.Message{
		MessageID: 1, Chat: api.Chat{ID: 1, Type: "private"}, Text: "ping",
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	go func() { _ = r.Run(ctx, newFake(u)) }()

	require.Equal(t, "ping", <-hit)
}

func TestRouter_OnMessageFilter_NoMatch(t *testing.T) {
	r := New(client.New("t"))
	called := false
	r.OnMessageFilter(
		Filter[*api.Message](func(m *api.Message) bool { return false }),
		func(c *Context, m *api.Message) error { called = true; return nil },
	)

	u := api.Update{UpdateID: 1, Message: &api.Message{
		MessageID: 1, Chat: api.Chat{ID: 1, Type: "private"}, Text: "any",
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_ = r.Run(ctx, newFake(u))
	require.False(t, called)
}

// Command routes must take priority over filter routes.
func TestRouter_OnMessageFilter_CommandWins(t *testing.T) {
	r := New(client.New("t"))
	var winner string
	r.OnCommand("/start", func(c *Context, m *api.Message) error { winner = "command"; return nil })
	r.OnMessageFilter(
		Filter[*api.Message](func(m *api.Message) bool { return true }),
		func(c *Context, m *api.Message) error { winner = "filter"; return nil },
	)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_ = r.Run(ctx, newFake(cmdMessage("/start")))

	require.Equal(t, "command", winner)
}

func TestRouter_OnCallbackFilter_Matches(t *testing.T) {
	r := New(client.New("t"))
	hit := make(chan string, 1)
	r.OnCallbackFilter(
		Filter[*api.CallbackQuery](func(q *api.CallbackQuery) bool { return q != nil && q.Data == "yes" }),
		func(c *Context, q *api.CallbackQuery) error { hit <- q.Data; return nil },
	)

	u := api.Update{UpdateID: 1, CallbackQuery: &api.CallbackQuery{
		ID: "x", From: api.User{ID: 1}, ChatInstance: "y", Data: "yes",
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	go func() { _ = r.Run(ctx, newFake(u)) }()

	require.Equal(t, "yes", <-hit)
}

// Pattern-based OnCallback wins over OnCallbackFilter when both match.
func TestRouter_OnCallbackFilter_PatternWins(t *testing.T) {
	r := New(client.New("t"))
	var winner string
	r.OnCallback(`^yes$`, func(c *Context, q *api.CallbackQuery) error { winner = "pattern"; return nil })
	r.OnCallbackFilter(
		Filter[*api.CallbackQuery](func(q *api.CallbackQuery) bool { return true }),
		func(c *Context, q *api.CallbackQuery) error { winner = "filter"; return nil },
	)

	u := api.Update{UpdateID: 1, CallbackQuery: &api.CallbackQuery{
		ID: "x", From: api.User{ID: 1}, ChatInstance: "y", Data: "yes",
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_ = r.Run(ctx, newFake(u))

	require.Equal(t, "pattern", winner)
}

func TestRouter_OnInlineQueryFilter_Matches(t *testing.T) {
	r := New(client.New("t"))
	hit := make(chan string, 1)
	r.OnInlineQueryFilter(
		Filter[*api.InlineQuery](func(q *api.InlineQuery) bool { return q != nil && q.Query == "find" }),
		func(c *Context, q *api.InlineQuery) error { hit <- q.Query; return nil },
	)

	u := api.Update{UpdateID: 1, InlineQuery: &api.InlineQuery{
		ID: "i", From: api.User{ID: 1}, Query: "find",
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	go func() { _ = r.Run(ctx, newFake(u)) }()

	require.Equal(t, "find", <-hit)
}

func TestRouter_FilterChain_Composition(t *testing.T) {
	// Filter: private chat AND text contains "hello"
	privateChat := Filter[*api.Message](func(m *api.Message) bool {
		return m != nil && m.Chat.Type == api.ChatTypePrivate
	})
	hasHello := Filter[*api.Message](func(m *api.Message) bool {
		return m != nil && len(m.Text) > 0 && containsStr(m.Text, "hello")
	})
	combined := privateChat.And(hasHello)

	r := New(client.New("t"))
	hit := make(chan string, 1)
	r.OnMessageFilter(combined, func(c *Context, m *api.Message) error { hit <- m.Text; return nil })

	match := api.Update{UpdateID: 1, Message: &api.Message{
		MessageID: 1, Chat: api.Chat{ID: 1, Type: api.ChatTypePrivate}, Text: "say hello",
	}}
	noMatch := api.Update{UpdateID: 2, Message: &api.Message{
		MessageID: 2, Chat: api.Chat{ID: 2, Type: api.ChatTypeGroup}, Text: "say hello",
	}}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	go func() { _ = r.Run(ctx, newFake(match, noMatch)) }()

	require.Equal(t, "say hello", <-hit)
}

// containsStr is a helper to avoid importing strings in test file unnecessarily.
func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsSubstr(s, sub))
}

func containsSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Concurrent dispatch tests
// ---------------------------------------------------------------------------

// fakeSlowUpdater feeds n updates then blocks until ctx cancel.
type fakeSlowUpdater struct {
	ch chan api.Update
}

func newSlowFake(ups ...api.Update) *fakeSlowUpdater {
	ch := make(chan api.Update, len(ups))
	for _, u := range ups {
		ch <- u
	}
	close(ch)
	return &fakeSlowUpdater{ch: ch}
}

func (f *fakeSlowUpdater) Updates() <-chan api.Update     { return f.ch }
func (f *fakeSlowUpdater) Run(ctx context.Context) error  { <-ctx.Done(); return ctx.Err() }
func (f *fakeSlowUpdater) Stop(ctx context.Context) error { return nil }

func TestRouter_ConcurrentDispatch_AllHandlersFire(t *testing.T) {
	const n = 100
	var fired atomic.Int64

	ups := make([]api.Update, n)
	for i := range ups {
		ups[i] = api.Update{UpdateID: int64(i + 1), Message: &api.Message{
			MessageID: int64(i + 1),
			Chat:      api.Chat{ID: 1, Type: api.ChatTypePrivate},
			Text:      "hi",
		}}
	}

	r := New(client.New("t"), WithMaxConcurrency(20))
	r.OnMessageFilter(
		Filter[*api.Message](func(m *api.Message) bool { return true }),
		func(c *Context, m *api.Message) error { fired.Add(1); return nil },
	)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = r.Run(ctx, newSlowFake(ups...))

	require.Equal(t, int64(n), fired.Load())
}

func TestRouter_ConcurrentDispatch_SemaphoreBoundsConcurrency(t *testing.T) {
	const limit = 5
	const n = 30

	var inFlight atomic.Int64
	var maxSeen atomic.Int64
	ready := make(chan struct{})   // signals handler to proceed
	started := make(chan struct{}) // first handler signals it's running

	ups := make([]api.Update, n)
	for i := range ups {
		ups[i] = api.Update{UpdateID: int64(i + 1), Message: &api.Message{
			MessageID: int64(i + 1),
			Chat:      api.Chat{ID: 1, Type: api.ChatTypePrivate},
			Text:      "hi",
		}}
	}

	once := atomic.Bool{}
	r := New(client.New("t"), WithMaxConcurrency(limit))
	r.OnMessageFilter(
		Filter[*api.Message](func(m *api.Message) bool { return true }),
		func(c *Context, m *api.Message) error {
			cur := inFlight.Add(1)
			for {
				old := maxSeen.Load()
				if cur <= old || maxSeen.CompareAndSwap(old, cur) {
					break
				}
			}
			if once.CompareAndSwap(false, true) {
				close(started)
			}
			<-ready
			inFlight.Add(-1)
			return nil
		},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() { _ = r.Run(ctx, newSlowFake(ups...)) }()

	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal("timed out waiting for first handler")
	}
	// Give the pool a moment to fill up.
	time.Sleep(50 * time.Millisecond)
	close(ready)

	// Wait for Run to drain by cancelling context after a short wait.
	time.Sleep(200 * time.Millisecond)
	cancel()

	require.LessOrEqual(t, maxSeen.Load(), int64(limit),
		"in-flight goroutines exceeded semaphore limit")
}

func TestRouter_ConcurrentDispatch_WaitsForInFlight(t *testing.T) {
	unblock := make(chan struct{})
	done := make(chan struct{})

	r := New(client.New("t"), WithMaxConcurrency(10))
	r.OnMessageFilter(
		Filter[*api.Message](func(m *api.Message) bool { return true }),
		func(c *Context, m *api.Message) error {
			<-unblock
			return nil
		},
	)

	u := api.Update{UpdateID: 1, Message: &api.Message{
		MessageID: 1, Chat: api.Chat{ID: 1, Type: api.ChatTypePrivate}, Text: "hi",
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go func() {
		_ = r.Run(ctx, newSlowFake(u))
		close(done)
	}()

	// Give Run time to pick up the update and launch the goroutine.
	time.Sleep(30 * time.Millisecond)
	cancel() // trigger Run to exit its loop

	// Run should not return until handler unblocks.
	select {
	case <-done:
		t.Fatal("Run returned before in-flight handler finished")
	case <-time.After(50 * time.Millisecond):
	}

	close(unblock)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after handler finished")
	}
}

func TestRouter_SerialMode_NoRace(t *testing.T) {
	// WithMaxConcurrency(0) — serial; shared slice is safe without a mutex.
	var order []int64

	const n = 20
	ups := make([]api.Update, n)
	for i := range ups {
		ups[i] = api.Update{UpdateID: int64(i + 1), Message: &api.Message{
			MessageID: int64(i + 1),
			Chat:      api.Chat{ID: 1, Type: api.ChatTypePrivate},
			Text:      "hi",
		}}
	}

	r := New(client.New("t"), WithMaxConcurrency(0))
	r.OnMessageFilter(
		Filter[*api.Message](func(m *api.Message) bool { return true }),
		func(c *Context, m *api.Message) error {
			order = append(order, m.MessageID)
			return nil
		},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = r.Run(ctx, newSlowFake(ups...))

	require.Len(t, order, n)
	for i, v := range order {
		require.Equal(t, int64(i+1), v)
	}
}

// liveUpdater is an updater whose channel stays open until stopCh is closed.
type liveUpdater struct {
	ch     chan api.Update
	stopCh chan struct{}
}

func newLiveUpdater() *liveUpdater {
	return &liveUpdater{ch: make(chan api.Update, 8), stopCh: make(chan struct{})}
}

func (l *liveUpdater) Send(u api.Update)              { l.ch <- u }
func (l *liveUpdater) Close()                         { close(l.stopCh) }
func (l *liveUpdater) Updates() <-chan api.Update     { return l.ch }
func (l *liveUpdater) Stop(ctx context.Context) error { return nil }
func (l *liveUpdater) Run(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-l.stopCh:
		return nil
	}
}

// ---------------------------------------------------------------------------
// Typed handler tests (Feature 1)
// ---------------------------------------------------------------------------

func TestRouter_OnMyChatMember(t *testing.T) {
	r := New(client.New("t"))
	hit := make(chan int64, 1)
	r.OnMyChatMember(func(c *Context, u *api.ChatMemberUpdated) error { hit <- u.From.ID; return nil })

	upd := api.Update{UpdateID: 1, MyChatMember: &api.ChatMemberUpdated{
		From: api.User{ID: 42},
		Chat: api.Chat{ID: 1},
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	go func() { _ = r.Run(ctx, newFake(upd)) }()
	require.Equal(t, int64(42), <-hit)
}

func TestRouter_OnMyChatMemberFilter(t *testing.T) {
	r := New(client.New("t"))
	hit := make(chan int64, 1)
	f := Filter[*api.ChatMemberUpdated](func(u *api.ChatMemberUpdated) bool { return u.From.ID == 99 })
	r.OnMyChatMemberFilter(f, func(c *Context, u *api.ChatMemberUpdated) error { hit <- u.From.ID; return nil })

	match := api.Update{UpdateID: 1, MyChatMember: &api.ChatMemberUpdated{From: api.User{ID: 99}}}
	noMatch := api.Update{UpdateID: 2, MyChatMember: &api.ChatMemberUpdated{From: api.User{ID: 1}}}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	go func() { _ = r.Run(ctx, newFake(noMatch, match)) }()
	require.Equal(t, int64(99), <-hit)
}

func TestRouter_OnChatMember(t *testing.T) {
	r := New(client.New("t"))
	hit := make(chan int64, 1)
	r.OnChatMember(func(c *Context, u *api.ChatMemberUpdated) error { hit <- u.Chat.ID; return nil })

	upd := api.Update{UpdateID: 1, ChatMember: &api.ChatMemberUpdated{
		From: api.User{ID: 1},
		Chat: api.Chat{ID: 77},
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	go func() { _ = r.Run(ctx, newFake(upd)) }()
	require.Equal(t, int64(77), <-hit)
}

func TestRouter_OnChatMemberFilter(t *testing.T) {
	r := New(client.New("t"))
	hit := make(chan int64, 1)
	f := Filter[*api.ChatMemberUpdated](func(u *api.ChatMemberUpdated) bool { return u.Chat.ID == 55 })
	r.OnChatMemberFilter(f, func(c *Context, u *api.ChatMemberUpdated) error { hit <- u.Chat.ID; return nil })

	upd := api.Update{UpdateID: 1, ChatMember: &api.ChatMemberUpdated{Chat: api.Chat{ID: 55}}}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	go func() { _ = r.Run(ctx, newFake(upd)) }()
	require.Equal(t, int64(55), <-hit)
}

func TestRouter_OnChatJoinRequest(t *testing.T) {
	r := New(client.New("t"))
	hit := make(chan int64, 1)
	r.OnChatJoinRequest(func(c *Context, req *api.ChatJoinRequest) error { hit <- req.From.ID; return nil })

	upd := api.Update{UpdateID: 1, ChatJoinRequest: &api.ChatJoinRequest{
		From: api.User{ID: 11},
		Chat: api.Chat{ID: 1},
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	go func() { _ = r.Run(ctx, newFake(upd)) }()
	require.Equal(t, int64(11), <-hit)
}

func TestRouter_OnChatJoinRequestFilter(t *testing.T) {
	r := New(client.New("t"))
	hit := make(chan int64, 1)
	f := Filter[*api.ChatJoinRequest](func(req *api.ChatJoinRequest) bool { return req.Chat.ID == 22 })
	r.OnChatJoinRequestFilter(f, func(c *Context, req *api.ChatJoinRequest) error { hit <- req.Chat.ID; return nil })

	upd := api.Update{UpdateID: 1, ChatJoinRequest: &api.ChatJoinRequest{
		From: api.User{ID: 1},
		Chat: api.Chat{ID: 22},
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	go func() { _ = r.Run(ctx, newFake(upd)) }()
	require.Equal(t, int64(22), <-hit)
}

func TestRouter_OnPreCheckoutQuery(t *testing.T) {
	r := New(client.New("t"))
	hit := make(chan string, 1)
	r.OnPreCheckoutQuery(func(c *Context, q *api.PreCheckoutQuery) error { hit <- q.Currency; return nil })

	upd := api.Update{UpdateID: 1, PreCheckoutQuery: &api.PreCheckoutQuery{
		ID: "q1", From: api.User{ID: 1}, Currency: "USD",
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	go func() { _ = r.Run(ctx, newFake(upd)) }()
	require.Equal(t, "USD", <-hit)
}

func TestRouter_OnPreCheckoutQueryFilter(t *testing.T) {
	r := New(client.New("t"))
	hit := make(chan string, 1)
	f := Filter[*api.PreCheckoutQuery](func(q *api.PreCheckoutQuery) bool { return q.Currency == "EUR" })
	r.OnPreCheckoutQueryFilter(f, func(c *Context, q *api.PreCheckoutQuery) error { hit <- q.Currency; return nil })

	upd := api.Update{UpdateID: 1, PreCheckoutQuery: &api.PreCheckoutQuery{
		ID: "q1", From: api.User{ID: 1}, Currency: "EUR",
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	go func() { _ = r.Run(ctx, newFake(upd)) }()
	require.Equal(t, "EUR", <-hit)
}

func TestRouter_OnShippingQuery(t *testing.T) {
	r := New(client.New("t"))
	hit := make(chan string, 1)
	r.OnShippingQuery(func(c *Context, q *api.ShippingQuery) error { hit <- q.ID; return nil })

	upd := api.Update{UpdateID: 1, ShippingQuery: &api.ShippingQuery{ID: "sq1", From: api.User{ID: 1}}}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	go func() { _ = r.Run(ctx, newFake(upd)) }()
	require.Equal(t, "sq1", <-hit)
}

func TestRouter_OnPoll(t *testing.T) {
	r := New(client.New("t"))
	hit := make(chan string, 1)
	r.OnPoll(func(c *Context, p *api.Poll) error { hit <- p.ID; return nil })

	upd := api.Update{UpdateID: 1, Poll: &api.Poll{ID: "poll1"}}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	go func() { _ = r.Run(ctx, newFake(upd)) }()
	require.Equal(t, "poll1", <-hit)
}

func TestRouter_OnPollAnswer(t *testing.T) {
	r := New(client.New("t"))
	hit := make(chan string, 1)
	r.OnPollAnswer(func(c *Context, a *api.PollAnswer) error { hit <- a.PollID; return nil })

	upd := api.Update{UpdateID: 1, PollAnswer: &api.PollAnswer{PollID: "p1", OptionIds: []int64{0}}}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	go func() { _ = r.Run(ctx, newFake(upd)) }()
	require.Equal(t, "p1", <-hit)
}

func TestRouter_OnChosenInlineResult(t *testing.T) {
	r := New(client.New("t"))
	hit := make(chan string, 1)
	r.OnChosenInlineResult(func(c *Context, res *api.ChosenInlineResult) error { hit <- res.ResultID; return nil })

	upd := api.Update{UpdateID: 1, ChosenInlineResult: &api.ChosenInlineResult{ResultID: "r1", From: api.User{ID: 1}}}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	go func() { _ = r.Run(ctx, newFake(upd)) }()
	require.Equal(t, "r1", <-hit)
}

func TestRouter_OnMessageReaction(t *testing.T) {
	r := New(client.New("t"))
	hit := make(chan int64, 1)
	r.OnMessageReaction(func(c *Context, u *api.MessageReactionUpdated) error { hit <- u.Chat.ID; return nil })

	upd := api.Update{UpdateID: 1, MessageReaction: &api.MessageReactionUpdated{Chat: api.Chat{ID: 33}}}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	go func() { _ = r.Run(ctx, newFake(upd)) }()
	require.Equal(t, int64(33), <-hit)
}

func TestRouter_OnMessageReactionCount(t *testing.T) {
	r := New(client.New("t"))
	hit := make(chan int64, 1)
	r.OnMessageReactionCount(func(c *Context, u *api.MessageReactionCountUpdated) error { hit <- u.Chat.ID; return nil })

	upd := api.Update{UpdateID: 1, MessageReactionCount: &api.MessageReactionCountUpdated{Chat: api.Chat{ID: 44}}}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	go func() { _ = r.Run(ctx, newFake(upd)) }()
	require.Equal(t, int64(44), <-hit)
}

func TestRouter_OnChatBoost(t *testing.T) {
	r := New(client.New("t"))
	hit := make(chan int64, 1)
	r.OnChatBoost(func(c *Context, u *api.ChatBoostUpdated) error { hit <- u.Chat.ID; return nil })

	upd := api.Update{UpdateID: 1, ChatBoost: &api.ChatBoostUpdated{Chat: api.Chat{ID: 55}}}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	go func() { _ = r.Run(ctx, newFake(upd)) }()
	require.Equal(t, int64(55), <-hit)
}

func TestRouter_OnRemovedChatBoost(t *testing.T) {
	r := New(client.New("t"))
	hit := make(chan int64, 1)
	r.OnRemovedChatBoost(func(c *Context, u *api.ChatBoostRemoved) error { hit <- u.Chat.ID; return nil })

	upd := api.Update{UpdateID: 1, RemovedChatBoost: &api.ChatBoostRemoved{Chat: api.Chat{ID: 66}}}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	go func() { _ = r.Run(ctx, newFake(upd)) }()
	require.Equal(t, int64(66), <-hit)
}

func TestRouter_OnBusinessConnection(t *testing.T) {
	r := New(client.New("t"))
	hit := make(chan string, 1)
	r.OnBusinessConnection(func(c *Context, bc *api.BusinessConnection) error { hit <- bc.ID; return nil })

	upd := api.Update{UpdateID: 1, BusinessConnection: &api.BusinessConnection{ID: "bc1", UserChatID: 1, User: api.User{ID: 1}}}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	go func() { _ = r.Run(ctx, newFake(upd)) }()
	require.Equal(t, "bc1", <-hit)
}

func TestRouter_OnPurchasedPaidMedia(t *testing.T) {
	r := New(client.New("t"))
	hit := make(chan string, 1)
	r.OnPurchasedPaidMedia(func(c *Context, p *api.PaidMediaPurchased) error { hit <- p.PaidMediaPayload; return nil })

	upd := api.Update{UpdateID: 1, PurchasedPaidMedia: &api.PaidMediaPurchased{From: api.User{ID: 1}, PaidMediaPayload: "payload1"}}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	go func() { _ = r.Run(ctx, newFake(upd)) }()
	require.Equal(t, "payload1", <-hit)
}

func TestRouter_ContextCancel_UnblocksWaitingAcquire(t *testing.T) {
	// Fill the semaphore with slow handlers, send one more update, then cancel
	// ctx. Run must unblock from the semaphore-acquire select and return.
	const limit = 2
	unblock := make(chan struct{})

	slowHandler := func(c *Context, m *api.Message) error {
		<-unblock
		return nil
	}

	lu := newLiveUpdater()
	r := New(client.New("t"), WithMaxConcurrency(limit))
	r.OnMessageFilter(Filter[*api.Message](func(m *api.Message) bool { return true }), slowHandler)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runDone := make(chan error, 1)
	go func() { runDone <- r.Run(ctx, lu) }()

	// Send enough updates to fill semaphore.
	for i := range limit {
		lu.Send(api.Update{UpdateID: int64(i + 1), Message: &api.Message{
			MessageID: int64(i + 1),
			Chat:      api.Chat{ID: 1, Type: api.ChatTypePrivate},
			Text:      "hi",
		}})
	}

	// Give goroutines time to acquire all semaphore slots.
	time.Sleep(50 * time.Millisecond)

	// Send one more update — Run will block trying to acquire the full semaphore.
	lu.Send(api.Update{UpdateID: int64(limit + 1), Message: &api.Message{
		MessageID: int64(limit + 1),
		Chat:      api.Chat{ID: 1, Type: api.ChatTypePrivate},
		Text:      "extra",
	}})

	// Give Run a moment to reach the semaphore-acquire select.
	time.Sleep(30 * time.Millisecond)
	cancel()

	// Unblock handlers so wg.Wait() inside Run can complete, allowing Run to
	// return (and write to runDone).
	close(unblock)

	select {
	case err := <-runDone:
		require.Error(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not unblock after context cancel")
	}
}
