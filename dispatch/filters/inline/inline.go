// Package inline provides Filter helpers for *api.InlineQuery payloads.
package inline

import (
	"regexp"
	"strings"

	"github.com/lukaszraczylo/go-telegram/api"
	"github.com/lukaszraczylo/go-telegram/dispatch"
)

// Query returns a Filter that matches inline queries whose Query field matches
// pattern (regex). Panics at registration time on an invalid pattern.
func Query(pattern string) dispatch.Filter[*api.InlineQuery] {
	re := regexp.MustCompile(pattern)
	return func(q *api.InlineQuery) bool {
		return q != nil && re.MatchString(q.Query)
	}
}

// QueryEquals returns a Filter that matches inline queries whose Query equals
// s exactly.
func QueryEquals(s string) dispatch.Filter[*api.InlineQuery] {
	return func(q *api.InlineQuery) bool {
		return q != nil && q.Query == s
	}
}

// QueryPrefix returns a Filter that matches inline queries whose Query starts
// with prefix.
func QueryPrefix(prefix string) dispatch.Filter[*api.InlineQuery] {
	return func(q *api.InlineQuery) bool {
		return q != nil && strings.HasPrefix(q.Query, prefix)
	}
}
