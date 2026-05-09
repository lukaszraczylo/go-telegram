package inline_test

import (
	"testing"

	"github.com/lukaszraczylo/go-telegram/api"
	ilfilter "github.com/lukaszraczylo/go-telegram/dispatch/filters/inline"
	"github.com/stretchr/testify/require"
)

func iq(query string) *api.InlineQuery {
	return &api.InlineQuery{ID: "i", From: api.User{ID: 1}, Query: query}
}

func TestQuery(t *testing.T) {
	f := ilfilter.Query(`^find`)
	require.True(t, f(iq("find me")))
	require.False(t, f(iq("search me")))
	require.False(t, f(nil))
}

func TestQuery_PanicsOnBadPattern(t *testing.T) {
	require.Panics(t, func() { ilfilter.Query(`[bad`) })
}

func TestQueryEquals(t *testing.T) {
	f := ilfilter.QueryEquals("exact")
	require.True(t, f(iq("exact")))
	require.False(t, f(iq("exact match")))
	require.False(t, f(nil))
}

func TestQueryPrefix(t *testing.T) {
	f := ilfilter.QueryPrefix("@user")
	require.True(t, f(iq("@username")))
	require.False(t, f(iq("no prefix")))
	require.False(t, f(nil))
}

func TestComposedInlineFilters(t *testing.T) {
	f := ilfilter.QueryPrefix("find").Or(ilfilter.QueryEquals("help"))
	require.True(t, f(iq("find me")))
	require.True(t, f(iq("help")))
	require.False(t, f(iq("other")))
}
