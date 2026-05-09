package client

// Option configures a Bot at construction time. Per-call configuration is
// expressed via typed parameter structs (e.g. SendMessageParams), not options.
type Option func(*Bot)

// WithHTTPClient overrides the HTTP transport. Pass any HTTPDoer
// implementation (e.g. an *http.Client wrapping a custom RoundTripper, or
// a fasthttp adapter).
func WithHTTPClient(c HTTPDoer) Option { return func(b *Bot) { b.http = c } }

// WithCodec overrides the JSON codec. Pass goccy/go-json, sonic, or any
// type implementing Codec to swap out encoding/json.
func WithCodec(c Codec) Option { return func(b *Bot) { b.codec = c } }

// WithBaseURL overrides the API base URL. Useful for testing against a
// local httptest.Server, or for self-hosted Bot API servers.
func WithBaseURL(url string) Option { return func(b *Bot) { b.base = url } }

// WithLogger sets the logger used for diagnostic events. Passing nil
// silently disables logging.
func WithLogger(l Logger) Option {
	return func(b *Bot) {
		if l == nil {
			l = NoopLogger{}
		}
		b.logger = l
	}
}
