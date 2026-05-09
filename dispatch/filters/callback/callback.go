// Package callback provides Filter helpers for *api.CallbackQuery payloads.
package callback

import (
	"regexp"
	"strings"

	"github.com/lukaszraczylo/go-telegram/api"
	"github.com/lukaszraczylo/go-telegram/dispatch"
)

// Data returns a Filter that matches callback queries whose Data matches
// pattern (regex). Panics at registration time on an invalid pattern.
func Data(pattern string) dispatch.Filter[*api.CallbackQuery] {
	re := regexp.MustCompile(pattern)
	return func(q *api.CallbackQuery) bool {
		return q != nil && re.MatchString(q.Data)
	}
}

// DataEquals returns a Filter that matches callback queries whose Data equals
// s exactly.
func DataEquals(s string) dispatch.Filter[*api.CallbackQuery] {
	return func(q *api.CallbackQuery) bool {
		return q != nil && q.Data == s
	}
}

// DataPrefix returns a Filter that matches callback queries whose Data starts
// with prefix.
func DataPrefix(prefix string) dispatch.Filter[*api.CallbackQuery] {
	return func(q *api.CallbackQuery) bool {
		return q != nil && strings.HasPrefix(q.Data, prefix)
	}
}

// FromUser returns a Filter that matches callback queries whose From.ID equals
// userID.
func FromUser(userID int64) dispatch.Filter[*api.CallbackQuery] {
	return func(q *api.CallbackQuery) bool {
		return q != nil && q.From.ID == userID
	}
}
