package client

import (
	"net"
	"net/http"
	"time"
)

// HTTPDoer abstracts the HTTP transport. The default is a net/http client
// tuned for Telegram's long-poll usage. Users may plug in valyala/fasthttp
// (via an adapter), or any custom retry/circuit-breaker client by passing
// WithHTTPClient to New.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// NewDefaultHTTPDoer returns an *http.Client with sensible defaults for
// Telegram Bot API usage:
//   - 60s overall timeout (longer than typical long-poll Timeout=30s).
//   - Connection pooling sized for a small number of long-lived hosts.
//   - HTTP/2 enabled (default in net/http).
func NewDefaultHTTPDoer() *http.Client {
	t := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          16,
		MaxIdleConnsPerHost:   8,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
	}
	return &http.Client{
		Transport: t,
		Timeout:   60 * time.Second,
	}
}
