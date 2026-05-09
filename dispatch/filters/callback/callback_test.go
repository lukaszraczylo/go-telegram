package callback_test

import (
	"testing"

	"github.com/lukaszraczylo/go-telegram/api"
	cbfilter "github.com/lukaszraczylo/go-telegram/dispatch/filters/callback"
	"github.com/stretchr/testify/require"
)

func cq(data string, userID int64) *api.CallbackQuery {
	return &api.CallbackQuery{
		ID:   "q",
		From: api.User{ID: userID},
		Data: data,
	}
}

func TestData(t *testing.T) {
	f := cbfilter.Data(`^like:\d+$`)
	require.True(t, f(cq("like:42", 1)))
	require.False(t, f(cq("dislike:42", 1)))
	require.False(t, f(nil))
}

func TestData_PanicsOnBadPattern(t *testing.T) {
	require.Panics(t, func() { cbfilter.Data(`[bad`) })
}

func TestDataEquals(t *testing.T) {
	f := cbfilter.DataEquals("yes")
	require.True(t, f(cq("yes", 1)))
	require.False(t, f(cq("yes please", 1)))
	require.False(t, f(nil))
}

func TestDataPrefix(t *testing.T) {
	f := cbfilter.DataPrefix("vote:")
	require.True(t, f(cq("vote:up", 1)))
	require.False(t, f(cq("novote:up", 1)))
	require.False(t, f(nil))
}

func TestFromUser(t *testing.T) {
	f := cbfilter.FromUser(7)
	require.True(t, f(cq("data", 7)))
	require.False(t, f(cq("data", 8)))
	require.False(t, f(nil))
}

func TestComposedCallbackFilters(t *testing.T) {
	f := cbfilter.DataPrefix("vote:").And(cbfilter.FromUser(7))
	require.True(t, f(cq("vote:up", 7)))
	require.False(t, f(cq("vote:up", 8)))
	require.False(t, f(cq("other", 7)))
}
