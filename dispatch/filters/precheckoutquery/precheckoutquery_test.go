package precheckoutquery_test

import (
	"testing"

	"github.com/lukaszraczylo/go-telegram/api"
	pcqfilter "github.com/lukaszraczylo/go-telegram/dispatch/filters/precheckoutquery"
	"github.com/stretchr/testify/require"
)

func pcq(currency string, fromID int64) *api.PreCheckoutQuery {
	return &api.PreCheckoutQuery{
		ID:       "q",
		Currency: currency,
		From:     api.User{ID: fromID},
	}
}

func TestCurrency_Matches(t *testing.T) {
	f := pcqfilter.Currency("USD")
	require.True(t, f(pcq("USD", 1)))
	require.False(t, f(pcq("EUR", 1)))
	require.False(t, f(nil))
}

func TestFromUser_Matches(t *testing.T) {
	f := pcqfilter.FromUser(5)
	require.True(t, f(pcq("USD", 5)))
	require.False(t, f(pcq("USD", 9)))
	require.False(t, f(nil))
}

func TestComposedFilters(t *testing.T) {
	f := pcqfilter.Currency("XTR").And(pcqfilter.FromUser(42))
	require.True(t, f(pcq("XTR", 42)))
	require.False(t, f(pcq("XTR", 99)))
	require.False(t, f(pcq("USD", 42)))
}
