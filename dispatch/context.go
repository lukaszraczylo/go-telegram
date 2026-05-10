// Package dispatch provides a typed router for Telegram updates. It
// consumes any transport.Updater and dispatches updates to handlers
// registered by command, regex, or update-payload kind.
package dispatch

import (
	"context"

	"github.com/lukaszraczylo/go-telegram/api"
	"github.com/lukaszraczylo/go-telegram/client"
)

// Context bundles the per-update state every handler receives.
//
// Ctx is the request context propagated from Router.Run; cancelling the
// run cancels every handler.
//
// Bot is the API client. Handlers reply by calling api.SendMessage(c.Ctx,
// c.Bot, ...) etc.
//
// Update is the raw update; payload-typed handlers also receive a
// narrowed pointer to one of its sub-fields.
//
// Command, CommandArgs and RegexMatch are populated by the router for
// the matching route kind; they replace the previous "command",
// "command_args" and "regex_match" entries in Values, which were the
// only conventional keys. Values remains for user-defined custom keys.
//
// Command is the matched bot command (e.g. "/start"); empty when the
// route is not a command match.
//
// CommandArgs is everything after the command; empty when no command
// matched or the command had no trailing text.
//
// RegexMatch is the regex sub-matches when an OnText/OnCallback regex
// route matched; nil otherwise.
//
// Values is lazily allocated for user-defined keys. Handlers that don't
// write pay no allocation. Reads against a nil map return the zero
// value. Writers must use Set instead of indexing the map directly.
type Context struct {
	Ctx         context.Context
	Bot         *client.Bot
	Update      *api.Update
	Command     string
	CommandArgs string
	RegexMatch  []string
	Values      map[string]any
}

// NewContext constructs a Context. Used by Router internally; exposed for
// custom test harnesses.
func NewContext(ctx context.Context, b *client.Bot, u *api.Update) *Context {
	return &Context{Ctx: ctx, Bot: b, Update: u}
}

// Set writes key/val into Values, allocating the map on first use. Use
// this instead of `c.Values[k] = v` so the no-write path stays
// allocation-free.
func (c *Context) Set(key string, val any) {
	if c.Values == nil {
		c.Values = make(map[string]any, 2)
	}
	c.Values[key] = val
}
