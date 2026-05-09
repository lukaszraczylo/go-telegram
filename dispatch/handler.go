package dispatch

// Handler is a generic handler over update payload type T. T is typically
// *api.Message, *api.CallbackQuery, *api.InlineQuery, or *api.Update for
// global middleware.
type Handler[T any] func(ctx *Context, payload T) error

// Middleware wraps a Handler[T] with cross-cutting behaviour (logging,
// recovery, auth). Middleware composition is left-to-right: Use(a,b,c)
// runs as a(b(c(handler))).
type Middleware[T any] func(Handler[T]) Handler[T]

// Chain composes a slice of middleware into a single Middleware[T].
func Chain[T any](mws ...Middleware[T]) Middleware[T] {
	return func(h Handler[T]) Handler[T] {
		for i := len(mws) - 1; i >= 0; i-- {
			h = mws[i](h)
		}
		return h
	}
}
