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

// msgUpdate builds a simple private message update.
func msgUpdate(id int64, text string) api.Update {
	return api.Update{
		UpdateID: id,
		Message: &api.Message{
			MessageID: id,
			Chat:      api.Chat{ID: 1, Type: api.ChatTypePrivate},
			Text:      text,
		},
	}
}

// cmdUpdate builds a command message update.
func cmdUpdate(id int64, cmd string) api.Update {
	return api.Update{
		UpdateID: id,
		Message: &api.Message{
			MessageID: id,
			Chat:      api.Chat{ID: 1, Type: api.ChatTypePrivate},
			Text:      cmd,
			Entities: []api.MessageEntity{
				{Type: api.MessageEntityTypeBotCommand, Offset: 0, Length: int64(len(cmd))},
			},
		},
	}
}

// runSingle fires one update through the router and waits for it to complete.
func runSingle(t *testing.T, r *Router, up api.Update) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_ = r.Run(ctx, newFake(up))
}

// TestGroup_Order verifies group 0 fires before group 1.
func TestGroup_Order(t *testing.T) {
	r := New(client.New("t"))
	var order []int

	r.Group(0).OnText(`.*`, func(c *Context, m *api.Message) error {
		order = append(order, 0)
		return ErrContinueGroups // let group 1 also run
	})
	r.Group(1).OnText(`.*`, func(c *Context, m *api.Message) error {
		order = append(order, 1)
		return nil
	})

	runSingle(t, r, msgUpdate(1, "hello"))
	require.Equal(t, []int{0, 1}, order)
}

// TestGroup_FirstMatchWins verifies group 0 match stops group 1 by default.
func TestGroup_FirstMatchWins(t *testing.T) {
	r := New(client.New("t"))
	var fired []int

	r.Group(0).OnText(`.*`, func(c *Context, m *api.Message) error {
		fired = append(fired, 0)
		return nil // matched — group 1 must NOT run
	})
	r.Group(1).OnText(`.*`, func(c *Context, m *api.Message) error {
		fired = append(fired, 1)
		return nil
	})

	runSingle(t, r, msgUpdate(1, "hello"))
	require.Equal(t, []int{0}, fired)
}

// TestGroup_ErrContinueGroups lets group 1 run when group 0 returns ErrContinueGroups.
func TestGroup_ErrContinueGroups(t *testing.T) {
	r := New(client.New("t"))
	g1Hit := make(chan struct{}, 1)

	r.Group(0).OnText(`.*`, func(c *Context, m *api.Message) error {
		return ErrContinueGroups
	})
	r.Group(1).OnText(`.*`, func(c *Context, m *api.Message) error {
		g1Hit <- struct{}{}
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	go func() { _ = r.Run(ctx, newFake(msgUpdate(1, "ping"))) }()

	select {
	case <-g1Hit:
	case <-ctx.Done():
		t.Fatal("group 1 handler never fired")
	}
}

// TestGroup_ErrEndGroups stops all further groups.
func TestGroup_ErrEndGroups(t *testing.T) {
	r := New(client.New("t"))
	var fired []int

	r.Group(0).OnText(`.*`, func(c *Context, m *api.Message) error {
		fired = append(fired, 0)
		return ErrEndGroups
	})
	r.Group(1).OnText(`.*`, func(c *Context, m *api.Message) error {
		fired = append(fired, 1)
		return nil
	})

	runSingle(t, r, msgUpdate(1, "hello"))
	require.Equal(t, []int{0}, fired)
}

// TestGroup_NonSentinelError propagates error and stops further groups.
func TestGroup_NonSentinelError(t *testing.T) {
	r := New(client.New("t"), WithMaxConcurrency(0))
	var fired []int

	r.Group(0).OnText(`.*`, func(c *Context, m *api.Message) error {
		fired = append(fired, 0)
		return context.DeadlineExceeded // non-sentinel real error
	})
	r.Group(1).OnText(`.*`, func(c *Context, m *api.Message) error {
		fired = append(fired, 1)
		return nil
	})

	runSingle(t, r, msgUpdate(1, "hello"))
	// group 1 must not fire
	require.Equal(t, []int{0}, fired)
}

// TestGroup_Command verifies OnCommand in a group works.
func TestGroup_Command(t *testing.T) {
	r := New(client.New("t"))
	hit := make(chan string, 1)

	r.Group(0).OnCommand("/start", func(c *Context, m *api.Message) error {
		hit <- "g0-start"
		return nil
	})
	r.Group(1).OnCommand("/start", func(c *Context, m *api.Message) error {
		hit <- "g1-start"
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	go func() { _ = r.Run(ctx, newFake(cmdUpdate(1, "/start"))) }()

	got := <-hit
	require.Equal(t, "g0-start", got)
}

// TestGroup_MessageFilter verifies OnMessageFilter in a group works.
func TestGroup_MessageFilter(t *testing.T) {
	r := New(client.New("t"))
	hit := make(chan bool, 1)

	r.Group(0).OnMessageFilter(
		Filter[*api.Message](func(m *api.Message) bool { return m != nil && m.Text == "ok" }),
		func(c *Context, m *api.Message) error {
			hit <- true
			return nil
		},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	go func() { _ = r.Run(ctx, newFake(msgUpdate(1, "ok"))) }()

	require.True(t, <-hit)
}

// TestGroup_ErrContinueGroups_WithCommand verifies ErrContinueGroups works for commands across groups.
func TestGroup_ErrContinueGroups_WithCommand(t *testing.T) {
	r := New(client.New("t"))
	var count atomic.Int32

	r.Group(0).OnCommand("/ping", func(c *Context, m *api.Message) error {
		count.Add(1)
		return ErrContinueGroups
	})
	r.Group(1).OnCommand("/ping", func(c *Context, m *api.Message) error {
		count.Add(10)
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	go func() { _ = r.Run(ctx, newFake(cmdUpdate(1, "/ping"))) }()
	time.Sleep(100 * time.Millisecond)
	cancel()

	require.Equal(t, int32(11), count.Load())
}
