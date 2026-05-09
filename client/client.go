package client

const defaultBaseURL = "https://api.telegram.org"

// Bot is the Telegram Bot API client. Construct via New. All API methods
// (declared in package api) hang off *Bot via thin wrappers around call.
type Bot struct {
	token  string
	base   string
	http   HTTPDoer
	codec  Codec
	logger Logger
}

// Token returns the bot token. Exposed for advanced use cases (custom
// transports, manual URL building); ordinary code does not need it.
func (b *Bot) Token() string { return b.token }

// BaseURL returns the configured Telegram API base URL.
func (b *Bot) BaseURL() string { return b.base }

// HTTP returns the underlying HTTPDoer. Exposed for adapters that need
// to share connection pools or for diagnostic checks.
func (b *Bot) HTTP() HTTPDoer { return b.http }

// Codec returns the configured Codec.
func (b *Bot) Codec() Codec { return b.codec }

// Logger returns the configured Logger.
func (b *Bot) Logger() Logger { return b.logger }

// New constructs a Bot with the given token and optional configuration.
// The default HTTP client is tuned for long-poll workloads (see
// NewDefaultHTTPDoer); the default codec wraps encoding/json; the default
// logger discards records.
func New(token string, opts ...Option) *Bot {
	b := &Bot{
		token:  token,
		base:   defaultBaseURL,
		http:   NewDefaultHTTPDoer(),
		codec:  DefaultCodec{},
		logger: NoopLogger{},
	}
	for _, o := range opts {
		o(b)
	}
	return b
}
