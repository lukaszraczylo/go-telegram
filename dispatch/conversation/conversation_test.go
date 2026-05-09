package conversation_test

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/lukaszraczylo/go-telegram/api"
	"github.com/lukaszraczylo/go-telegram/client"
	"github.com/lukaszraczylo/go-telegram/dispatch"
	"github.com/lukaszraczylo/go-telegram/dispatch/conversation"
	"github.com/stretchr/testify/require"
)

// ---- helpers ---------------------------------------------------------------

func msgUpd(userID, chatID int64, text string) api.Update {
	return api.Update{
		UpdateID: 1,
		Message: &api.Message{
			MessageID: 1,
			From:      &api.User{ID: userID},
			Chat:      api.Chat{ID: chatID},
			Text:      text,
		},
	}
}

func makeCtx(u *api.Update) *dispatch.Context {
	return dispatch.NewContext(context.Background(), client.New("t"), u)
}

// anyMsg matches any update that has a Message.
var anyMsg = func(u *api.Update) bool { return u.Message != nil }

// hasPrefix returns a filter matching updates whose Message.Text has prefix p.
func hasPrefix(p string) dispatch.Filter[*api.Update] {
	return func(u *api.Update) bool {
		return u.Message != nil && strings.HasPrefix(u.Message.Text, p)
	}
}

// fakeUpdater feeds a fixed set of updates then closes (mirrors router_test.go).
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

// ---- Storage tests ---------------------------------------------------------

func TestStorage_ErrKeyNotFound(t *testing.T) {
	s := conversation.NewMemoryStorage()
	_, err := s.Get(context.Background(), "missing")
	require.ErrorIs(t, err, conversation.ErrKeyNotFound)
}

func TestStorage_SetAndGet(t *testing.T) {
	ctx := context.Background()
	s := conversation.NewMemoryStorage()
	require.NoError(t, s.Set(ctx, "k", "state_a"))
	v, err := s.Get(ctx, "k")
	require.NoError(t, err)
	require.Equal(t, conversation.State("state_a"), v)
}

func TestStorage_Delete(t *testing.T) {
	ctx := context.Background()
	s := conversation.NewMemoryStorage()
	require.NoError(t, s.Set(ctx, "k", "state_a"))
	require.NoError(t, s.Delete(ctx, "k"))
	_, err := s.Get(ctx, "k")
	require.ErrorIs(t, err, conversation.ErrKeyNotFound)
}

func TestStorage_DeleteNonExistentIsNoop(t *testing.T) {
	require.NoError(t, conversation.NewMemoryStorage().Delete(context.Background(), "gone"))
}

// ---- Key strategy tests ----------------------------------------------------

func TestKeyByUser_Variants(t *testing.T) {
	t.Run("message", func(t *testing.T) {
		u := msgUpd(42, 100, "hi")
		require.Equal(t, "u:42", conversation.KeyByUser(&u))
	})
	t.Run("edited_message", func(t *testing.T) {
		u := api.Update{EditedMessage: &api.Message{From: &api.User{ID: 7}, Chat: api.Chat{ID: 1}}}
		require.Equal(t, "u:7", conversation.KeyByUser(&u))
	})
	t.Run("callback_query", func(t *testing.T) {
		u := api.Update{CallbackQuery: &api.CallbackQuery{From: api.User{ID: 99}}}
		require.Equal(t, "u:99", conversation.KeyByUser(&u))
	})
	t.Run("inline_query", func(t *testing.T) {
		u := api.Update{InlineQuery: &api.InlineQuery{From: api.User{ID: 5}}}
		require.Equal(t, "u:5", conversation.KeyByUser(&u))
	})
	t.Run("empty", func(t *testing.T) {
		require.Equal(t, "", conversation.KeyByUser(&api.Update{}))
	})
}

func TestKeyByChat_Variants(t *testing.T) {
	t.Run("message", func(t *testing.T) {
		u := msgUpd(1, 200, "")
		require.Equal(t, "c:200", conversation.KeyByChat(&u))
	})
	t.Run("inline_has_no_chat", func(t *testing.T) {
		u := api.Update{InlineQuery: &api.InlineQuery{From: api.User{ID: 5}}}
		require.Equal(t, "", conversation.KeyByChat(&u))
	})
}

func TestKeyByUserAndChat(t *testing.T) {
	u := msgUpd(42, 100, "")
	require.Equal(t, "uc:100:42", conversation.KeyByUserAndChat(&u))
}

// ---- Handler / state machine tests -----------------------------------------

func buildConv() *conversation.Conversation {
	return &conversation.Conversation{
		EntryPoints: []conversation.Step{{
			Filter: hasPrefix("/start"),
			Handler: func(c *dispatch.Context, u *api.Update) error {
				return conversation.Next("await_name")
			},
		}},
		States: map[conversation.State][]conversation.Step{
			"await_name": {{
				Filter:  anyMsg,
				Handler: func(c *dispatch.Context, u *api.Update) error { return conversation.Next("await_age") },
			}},
			"await_age": {{
				Filter:  anyMsg,
				Handler: func(c *dispatch.Context, u *api.Update) error { return conversation.End() },
			}},
		},
		Exits: []conversation.Step{{
			Filter:  hasPrefix("/cancel"),
			Handler: func(c *dispatch.Context, u *api.Update) error { return conversation.End() },
		}},
	}
}

func TestConversation_FullFlow(t *testing.T) {
	conv := buildConv()

	var downstream int
	noop := dispatch.Handler[*api.Update](func(_ *dispatch.Context, _ *api.Update) error {
		downstream++
		return nil
	})
	mw := conv.Dispatch(noop)

	key := "uc:1:42"

	// 1. /start → enters, state = await_name
	u1 := msgUpd(42, 1, "/start")
	require.NoError(t, mw(makeCtx(&u1), &u1))
	v, err := conv.Storage.Get(context.Background(), key)
	require.NoError(t, err)
	require.Equal(t, conversation.State("await_name"), v)
	require.Equal(t, 0, downstream, "entry claimed update")

	// 2. name → state = await_age
	u2 := msgUpd(42, 1, "Alice")
	require.NoError(t, mw(makeCtx(&u2), &u2))
	v, err = conv.Storage.Get(context.Background(), key)
	require.NoError(t, err)
	require.Equal(t, conversation.State("await_age"), v)

	// 3. age → End, key deleted
	u3 := msgUpd(42, 1, "30")
	require.NoError(t, mw(makeCtx(&u3), &u3))
	_, err = conv.Storage.Get(context.Background(), key)
	require.ErrorIs(t, err, conversation.ErrKeyNotFound)
}

func TestConversation_ExitsCancelMidFlow(t *testing.T) {
	conv := buildConv()
	noop := dispatch.Handler[*api.Update](func(_ *dispatch.Context, _ *api.Update) error { return nil })
	mw := conv.Dispatch(noop)

	// Start conversation.
	u1 := msgUpd(42, 1, "/start")
	require.NoError(t, mw(makeCtx(&u1), &u1))
	_, err := conv.Storage.Get(context.Background(), "uc:1:42")
	require.NoError(t, err)

	// Cancel mid-flow.
	u2 := msgUpd(42, 1, "/cancel")
	require.NoError(t, mw(makeCtx(&u2), &u2))
	_, err = conv.Storage.Get(context.Background(), "uc:1:42")
	require.ErrorIs(t, err, conversation.ErrKeyNotFound, "exit should clear state")
}

func TestConversation_FallbackFiresWhenNoStateStepMatches(t *testing.T) {
	fallbackHit := false
	conv := &conversation.Conversation{
		EntryPoints: []conversation.Step{{
			Filter:  hasPrefix("/start"),
			Handler: func(c *dispatch.Context, u *api.Update) error { return conversation.Next("waiting") },
		}},
		States: map[conversation.State][]conversation.Step{
			// No steps for "waiting" that match a callback query.
			"waiting": {},
		},
		Fallbacks: []conversation.Step{{
			Filter: anyMsg,
			Handler: func(c *dispatch.Context, u *api.Update) error {
				fallbackHit = true
				return nil
			},
		}},
	}

	noop := dispatch.Handler[*api.Update](func(_ *dispatch.Context, _ *api.Update) error { return nil })
	mw := conv.Dispatch(noop)

	u1 := msgUpd(42, 1, "/start")
	require.NoError(t, mw(makeCtx(&u1), &u1))

	u2 := msgUpd(42, 1, "unexpected text")
	require.NoError(t, mw(makeCtx(&u2), &u2))
	require.True(t, fallbackHit, "fallback should have fired")
}

func TestConversation_NoActiveConv_PassesToDownstream(t *testing.T) {
	conv := buildConv()
	downstreamHit := false
	downstream := dispatch.Handler[*api.Update](func(_ *dispatch.Context, _ *api.Update) error {
		downstreamHit = true
		return nil
	})
	mw := conv.Dispatch(downstream)

	// Random message that doesn't match /start
	u := msgUpd(42, 1, "hello")
	require.NoError(t, mw(makeCtx(&u), &u))
	require.True(t, downstreamHit, "unmatched update should reach downstream")
}

func TestConversation_EmptyKey_PassesThrough(t *testing.T) {
	// InlineQuery has no chatID → KeyByUserAndChat returns "" → pass through.
	conv := buildConv()
	downstreamHit := false
	downstream := dispatch.Handler[*api.Update](func(_ *dispatch.Context, _ *api.Update) error {
		downstreamHit = true
		return nil
	})
	mw := conv.Dispatch(downstream)

	u := api.Update{InlineQuery: &api.InlineQuery{From: api.User{ID: 5}}}
	require.NoError(t, mw(makeCtx(&u), &u))
	require.True(t, downstreamHit)
}

func TestConversation_AllowReEntry(t *testing.T) {
	conv := buildConv()
	conv.AllowReEntry = true

	noop := dispatch.Handler[*api.Update](func(_ *dispatch.Context, _ *api.Update) error { return nil })
	mw := conv.Dispatch(noop)

	// Start.
	u1 := msgUpd(42, 1, "/start")
	require.NoError(t, mw(makeCtx(&u1), &u1))
	v, _ := conv.Storage.Get(context.Background(), "uc:1:42")
	require.Equal(t, conversation.State("await_name"), v)

	// Advance once.
	u2 := msgUpd(42, 1, "Alice")
	require.NoError(t, mw(makeCtx(&u2), &u2))
	v, _ = conv.Storage.Get(context.Background(), "uc:1:42")
	require.Equal(t, conversation.State("await_age"), v)

	// Re-enter with /start — should restart to await_name even though mid-flow.
	u3 := msgUpd(42, 1, "/start")
	require.NoError(t, mw(makeCtx(&u3), &u3))
	v, _ = conv.Storage.Get(context.Background(), "uc:1:42")
	require.Equal(t, conversation.State("await_name"), v, "AllowReEntry should restart")
}

func TestConversation_NoReEntry_EntryIgnoredWhenActive(t *testing.T) {
	conv := buildConv()
	conv.AllowReEntry = false

	noop := dispatch.Handler[*api.Update](func(_ *dispatch.Context, _ *api.Update) error { return nil })
	mw := conv.Dispatch(noop)

	// Start.
	u1 := msgUpd(42, 1, "/start")
	require.NoError(t, mw(makeCtx(&u1), &u1))

	// Advance to await_age.
	u2 := msgUpd(42, 1, "Alice")
	require.NoError(t, mw(makeCtx(&u2), &u2))
	v, _ := conv.Storage.Get(context.Background(), "uc:1:42")
	require.Equal(t, conversation.State("await_age"), v)

	// /start again — should NOT restart; state should stay await_age since
	// /start matches the state step filter (anyMsg) and advances.
	// Actually /start is handled by "await_age" anyMsg step → End().
	u3 := msgUpd(42, 1, "/start")
	require.NoError(t, mw(makeCtx(&u3), &u3))
	// State ended (End() called by await_age step).
	_, err := conv.Storage.Get(context.Background(), "uc:1:42")
	require.ErrorIs(t, err, conversation.ErrKeyNotFound, "state step should have consumed /start when AllowReEntry=false")
}

func TestConversation_StayInState_NilReturn(t *testing.T) {
	// Handler returning nil keeps state unchanged.
	stored := false
	conv := &conversation.Conversation{
		EntryPoints: []conversation.Step{{
			Filter: hasPrefix("/start"),
			Handler: func(c *dispatch.Context, u *api.Update) error {
				return conversation.Next("waiting")
			},
		}},
		States: map[conversation.State][]conversation.Step{
			"waiting": {{
				Filter: anyMsg,
				Handler: func(c *dispatch.Context, u *api.Update) error {
					stored = true
					return nil // stay in current state
				},
			}},
		},
	}

	noop := dispatch.Handler[*api.Update](func(_ *dispatch.Context, _ *api.Update) error { return nil })
	mw := conv.Dispatch(noop)

	u1 := msgUpd(42, 1, "/start")
	require.NoError(t, mw(makeCtx(&u1), &u1))

	u2 := msgUpd(42, 1, "something")
	require.NoError(t, mw(makeCtx(&u2), &u2))
	require.True(t, stored)
	v, _ := conv.Storage.Get(context.Background(), "uc:1:42")
	require.Equal(t, conversation.State("waiting"), v, "nil return should leave state unchanged")
}

func TestConversation_ActiveNoMatch_Swallows(t *testing.T) {
	// Active conversation with no matching state step and no fallback:
	// update is swallowed (not passed downstream).
	conv := &conversation.Conversation{
		EntryPoints: []conversation.Step{{
			Filter:  hasPrefix("/start"),
			Handler: func(c *dispatch.Context, u *api.Update) error { return conversation.Next("waiting") },
		}},
		States: map[conversation.State][]conversation.Step{
			"waiting": {{
				// Only matches /done specifically.
				Filter:  hasPrefix("/done"),
				Handler: func(c *dispatch.Context, u *api.Update) error { return conversation.End() },
			}},
		},
	}

	downstreamHit := false
	downstream := dispatch.Handler[*api.Update](func(_ *dispatch.Context, _ *api.Update) error {
		downstreamHit = true
		return nil
	})
	mw := conv.Dispatch(downstream)

	u1 := msgUpd(42, 1, "/start")
	require.NoError(t, mw(makeCtx(&u1), &u1))

	// Random text doesn't match /done and there's no fallback → swallowed.
	u2 := msgUpd(42, 1, "random")
	require.NoError(t, mw(makeCtx(&u2), &u2))
	require.False(t, downstreamHit, "active conv with no matching step should swallow")
}

// ---- Via Router.Run --------------------------------------------------------

func TestConversation_ViaRouter(t *testing.T) {
	var steps atomic.Int32
	conv := &conversation.Conversation{
		EntryPoints: []conversation.Step{{
			Filter: hasPrefix("/start"),
			Handler: func(c *dispatch.Context, u *api.Update) error {
				steps.Add(1)
				return conversation.Next("await_name")
			},
		}},
		States: map[conversation.State][]conversation.Step{
			"await_name": {{
				Filter: anyMsg,
				Handler: func(c *dispatch.Context, u *api.Update) error {
					steps.Add(1)
					return conversation.Next("await_age")
				},
			}},
			"await_age": {{
				Filter: anyMsg,
				Handler: func(c *dispatch.Context, u *api.Update) error {
					steps.Add(1)
					return conversation.End()
				},
			}},
		},
	}

	router := dispatch.New(client.New("t"), dispatch.WithMaxConcurrency(0)) // serial
	router.Use(conv.Dispatch)

	ups := []api.Update{
		msgUpd(42, 1, "/start"),
		msgUpd(42, 1, "Alice"),
		msgUpd(42, 1, "30"),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() { errCh <- router.Run(ctx, newFake(ups...)) }()

	// Wait for updater channel to drain (Run returns when closed).
	err := <-errCh
	if err != nil && err != context.Canceled {
		t.Fatalf("Run error: %v", err)
	}

	require.Equal(t, int32(3), steps.Load(), "all three steps should have fired")
}

// ---- Concurrent storage safety ---------------------------------------------

func TestConversation_ConcurrentStorageAccess(t *testing.T) {
	// 15 goroutines each running a full /start → name → age flow against the
	// same shared storage but DIFFERENT keys (one per goroutine). Validates
	// no data races.
	const numUsers = 15

	conv := buildConv()
	noop := dispatch.Handler[*api.Update](func(_ *dispatch.Context, _ *api.Update) error { return nil })
	mw := conv.Dispatch(noop)

	var wg sync.WaitGroup
	wg.Add(numUsers)
	for i := 0; i < numUsers; i++ {
		go func(uid int64) {
			defer wg.Done()
			u1 := msgUpd(uid, uid, "/start")
			_ = mw(makeCtx(&u1), &u1)
			u2 := msgUpd(uid, uid, "Alice")
			_ = mw(makeCtx(&u2), &u2)
			u3 := msgUpd(uid, uid, "30")
			_ = mw(makeCtx(&u3), &u3)
		}(int64(i + 1))
	}
	wg.Wait()
	// Race detector catches bugs; no assertion needed beyond clean finish.
}

func TestConversation_ConcurrentSameKey(t *testing.T) {
	// 12 goroutines hammer the same key concurrently. Storage must not panic
	// or corrupt state. Race detector validates lock discipline.
	const goroutines = 12
	s := conversation.NewMemoryStorage()
	ctx := context.Background()

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(i int) {
			defer wg.Done()
			_ = s.Set(ctx, "shared", conversation.State("step"))
			_, _ = s.Get(ctx, "shared")
			if i%4 == 0 {
				_ = s.Delete(ctx, "shared")
			}
		}(i)
	}
	wg.Wait()
}
