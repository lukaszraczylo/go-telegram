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
// Values is a per-update bag matchers populate. Conventional keys:
//
//	"command":      string, the matched bot command (e.g. "/start")
//	"command_args": string, everything after the command
//	"regex_match":  []string, regex sub-matches when OnText matches
type Context struct {
	Ctx    context.Context
	Bot    *client.Bot
	Update *api.Update
	Values map[string]any
}

// NewContext constructs a Context. Used by Router internally; exposed for
// custom test harnesses.
func NewContext(ctx context.Context, b *client.Bot, u *api.Update) *Context {
	return &Context{Ctx: ctx, Bot: b, Update: u, Values: map[string]any{}}
}
