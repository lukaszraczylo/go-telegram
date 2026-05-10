package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/valyala/fasthttp"
)

// FastHTTPDoer is an HTTPDoer backed by github.com/valyala/fasthttp. It
// trades net/http compatibility (and HTTP/2 support) for substantially
// fewer allocations per request — fasthttp pools its Request and Response
// objects and uses a zero-allocation HTTP/1.1 parser.
//
// Use it for high-throughput bots when GC pressure matters and you don't
// need HTTP/2 or any net/http-only middleware (RoundTripper composition,
// the OpenTelemetry httptrace family, etc.):
//
//	bot := client.New(token, client.WithHTTPClient(client.NewFastHTTPDoer()))
//
// Wrap with RetryDoer the same way you would the default doer.
type FastHTTPDoer struct {
	client *fasthttp.Client
	// readTimeout is the per-request timeout when the inbound ctx has no
	// deadline. Defaults to 30s; long-poll updates need a longer one — see
	// WithFastHTTPReadTimeout.
	readTimeout time.Duration
}

// FastHTTPDoerOption configures a FastHTTPDoer.
type FastHTTPDoerOption func(*FastHTTPDoer)

// WithFastHTTPClient swaps in a pre-configured *fasthttp.Client.
// Useful for sharing a connection pool across multiple bots or applying
// custom dial / TLS configuration.
func WithFastHTTPClient(c *fasthttp.Client) FastHTTPDoerOption {
	return func(d *FastHTTPDoer) { d.client = c }
}

// WithFastHTTPReadTimeout sets the per-request fallback timeout used when
// the inbound context has no deadline. Long-poll callers should pass a
// value larger than the long-poll timeout.
func WithFastHTTPReadTimeout(t time.Duration) FastHTTPDoerOption {
	return func(d *FastHTTPDoer) { d.readTimeout = t }
}

// NewFastHTTPDoer constructs a FastHTTPDoer with sensible defaults.
func NewFastHTTPDoer(opts ...FastHTTPDoerOption) *FastHTTPDoer {
	d := &FastHTTPDoer{
		client: &fasthttp.Client{
			ReadTimeout:         90 * time.Second,
			WriteTimeout:        30 * time.Second,
			MaxIdleConnDuration: 90 * time.Second,
		},
		readTimeout: 30 * time.Second,
	}
	for _, o := range opts {
		o(d)
	}
	return d
}

// Do satisfies HTTPDoer by translating req into a pooled fasthttp.Request,
// dispatching it, and returning a *http.Response whose Body releases the
// pooled fasthttp.Response when Close is called.
//
// The conversion is intentionally minimal: URL goes via req.URL.RequestURI()
// + Host (avoids re-parsing), header values move byte-for-byte, and the
// body is taken straight from req.Body. *bytes.Buffer / *bytes.Reader are
// recognised so we can pass the underlying bytes without io.ReadAll.
func (d *FastHTTPDoer) Do(req *http.Request) (*http.Response, error) {
	if req == nil {
		return nil, errors.New("client: nil http.Request")
	}

	fReq := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(fReq)

	fReq.SetRequestURI(req.URL.String())
	fReq.Header.SetMethod(req.Method)
	if req.Host != "" {
		fReq.Header.SetHost(req.Host)
	}
	for name, values := range req.Header {
		for _, v := range values {
			fReq.Header.Set(name, v)
		}
	}

	if err := setFastHTTPBody(fReq, req); err != nil {
		return nil, err
	}

	fResp := fasthttp.AcquireResponse()
	// fResp is released by fasthttpResponseBody.Close — caller is
	// expected to defer resp.Body.Close() per net/http contract.

	deadline, hasDeadline := req.Context().Deadline()
	var err error
	if hasDeadline {
		err = d.client.DoDeadline(fReq, fResp, deadline)
	} else {
		err = d.client.DoTimeout(fReq, fResp, d.readTimeout)
	}
	if err != nil {
		fasthttp.ReleaseResponse(fResp)
		// Map fasthttp's timeout error to ctx.Err semantics so callers
		// can errors.Is(err, context.DeadlineExceeded).
		if hasDeadline && errors.Is(err, fasthttp.ErrTimeout) {
			return nil, context.DeadlineExceeded
		}
		return nil, err
	}

	httpResp := &http.Response{
		StatusCode:    fResp.StatusCode(),
		Status:        strconv.Itoa(fResp.StatusCode()) + " " + fastHTTPStatusText(fResp.StatusCode()),
		Header:        make(http.Header, fResp.Header.Len()),
		ContentLength: int64(fResp.Header.ContentLength()),
		Body:          &fasthttpResponseBody{resp: fResp, body: fResp.Body()},
		Request:       req,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
	}
	for k, v := range fResp.Header.All() {
		httpResp.Header.Add(string(k), string(v))
	}
	return httpResp, nil
}

// setFastHTTPBody copies req.Body into fReq with the cheapest path that
// preserves correctness. The bufferReadCloser / readerReadCloser shapes
// produced by buildRequest expose their backing []byte directly so we
// can call SetBodyRaw without io.ReadAll. Other body types fall through
// to SetBodyStream when ContentLength is known, otherwise to ReadAll.
func setFastHTTPBody(fReq *fasthttp.Request, req *http.Request) error {
	if req.Body == nil {
		return nil
	}
	switch v := req.Body.(type) {
	case bufferReadCloser:
		fReq.SetBodyRaw(v.Bytes())
		return nil
	case readerReadCloser:
		// *bytes.Reader.Bytes() returns the unread portion.
		size := v.Len()
		buf := make([]byte, size)
		_, err := v.Read(buf)
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		fReq.SetBodyRaw(buf)
		return nil
	default:
		if req.ContentLength > 0 {
			fReq.SetBodyStream(v, int(req.ContentLength))
		} else {
			body, err := io.ReadAll(v)
			if err != nil {
				return err
			}
			fReq.SetBodyRaw(body)
		}
		return nil
	}
}

// fasthttpResponseBody adapts a pooled *fasthttp.Response so it satisfies
// io.ReadCloser. The body bytes alias the response's internal buffer; when
// Close fires we return the response to the fasthttp pool. Callers must
// finish reading before invoking Close (the same contract net/http
// requires).
type fasthttpResponseBody struct {
	resp *fasthttp.Response
	body []byte
	pos  int
}

func (b *fasthttpResponseBody) Read(p []byte) (int, error) {
	if b.pos >= len(b.body) {
		return 0, io.EOF
	}
	n := copy(p, b.body[b.pos:])
	b.pos += n
	return n, nil
}

func (b *fasthttpResponseBody) Close() error {
	if b.resp != nil {
		fasthttp.ReleaseResponse(b.resp)
		b.resp = nil
		b.body = nil
	}
	return nil
}

// fastHTTPStatusText returns the textual reason phrase for a status code,
// matching the format net/http produces for *http.Response.Status. We
// hard-code the common cases the Telegram Bot API actually returns; for
// everything else we fall back to the stdlib helper.
func fastHTTPStatusText(code int) string {
	switch code {
	case http.StatusOK:
		return "OK"
	case http.StatusBadRequest:
		return "Bad Request"
	case http.StatusUnauthorized:
		return "Unauthorized"
	case http.StatusForbidden:
		return "Forbidden"
	case http.StatusNotFound:
		return "Not Found"
	case http.StatusTooManyRequests:
		return "Too Many Requests"
	case http.StatusInternalServerError:
		return "Internal Server Error"
	case http.StatusBadGateway:
		return "Bad Gateway"
	case http.StatusServiceUnavailable:
		return "Service Unavailable"
	case http.StatusGatewayTimeout:
		return "Gateway Timeout"
	default:
		return http.StatusText(code)
	}
}
