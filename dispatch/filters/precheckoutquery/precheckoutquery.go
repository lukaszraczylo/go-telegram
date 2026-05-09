// Package precheckoutquery provides Filter helpers for *api.PreCheckoutQuery payloads.
package precheckoutquery

import (
	"github.com/lukaszraczylo/go-telegram/api"
	"github.com/lukaszraczylo/go-telegram/dispatch"
)

// Currency returns a Filter that matches pre-checkout queries with the given
// ISO 4217 currency code (e.g. "USD", "EUR", "XTR").
func Currency(c string) dispatch.Filter[*api.PreCheckoutQuery] {
	return func(q *api.PreCheckoutQuery) bool {
		return q != nil && q.Currency == c
	}
}

// FromUser returns a Filter that matches pre-checkout queries sent by the
// user with the given ID.
func FromUser(uid int64) dispatch.Filter[*api.PreCheckoutQuery] {
	return func(q *api.PreCheckoutQuery) bool {
		return q != nil && q.From.ID == uid
	}
}
