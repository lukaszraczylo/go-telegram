package dispatch

import (
	"fmt"
	"runtime/debug"

	"github.com/lukaszraczylo/go-telegram/api"
)

// Recovery returns middleware that recovers from panics in downstream
// handlers, converting them into a returned error and logging via the
// bot's configured logger. Registered automatically by NewRouter.
func Recovery() Middleware[*api.Update] {
	return func(next Handler[*api.Update]) Handler[*api.Update] {
		return func(c *Context, u *api.Update) (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("panic in handler: %v\n%s", r, debug.Stack())
					if c.Bot != nil {
						c.Bot.Logger().Error("dispatch recovered panic", "err", err)
					}
				}
			}()
			return next(c, u)
		}
	}
}
