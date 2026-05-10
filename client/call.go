package client

import (
	"bytes"
	"context"
	"errors"
	"github.com/goccy/go-json"
	"io"
	"net/http"
	"reflect"
	"sync"
)

var (
	headerJSONValue = []string{"application/json"}
	rawOKTrueBody   = []byte(`{"ok":true,"result":true}`)
	rawOKFalseBody  = []byte(`{"ok":true,"result":false}`)

	// respBufPool reuses *bytes.Buffer for response body reads. Used on
	// paths whose decoder copies strings out of the input (decodeResult,
	// which delegates to goccy/go-json), so the buffer can be returned to
	// the pool as soon as Unmarshal has run. CallRaw and callMultipartRaw
	// return slices that alias the buffer and therefore cannot use the
	// pool without an extra copy that would defeat the point.
	respBufPool = sync.Pool{New: func() any { return new(bytes.Buffer) }}

	// reqBufPool reuses *bytes.Buffer for request body marshalling on the
	// JSON path. Only used when the configured Codec satisfies BodyEncoder
	// so we can stream-encode into the buffer instead of allocating an
	// intermediate []byte. The buffer is safe to return to the pool once
	// http.Client.Do (or RetryDoer, which io.ReadAlls the body up front)
	// has consumed it.
	reqBufPool = sync.Pool{New: func() any { return new(bytes.Buffer) }}
)

// maxPooledBufCap caps the buffer size returned to either pool. Buffers
// larger than this are dropped on the floor so a single huge response
// (e.g. a large getFile metadata payload) doesn't bloat the pool for the
// rest of the process lifetime.
const maxPooledBufCap = 64 * 1024

func putRespBuf(buf *bytes.Buffer) {
	if buf.Cap() > maxPooledBufCap {
		return
	}
	respBufPool.Put(buf)
}

func putReqBuf(buf *bytes.Buffer) {
	if buf.Cap() > maxPooledBufCap {
		return
	}
	reqBufPool.Put(buf)
}

// Call is the single point through which every Telegram Bot API method
// invocation flows. It marshals the request, signs the URL with the bot
// token, dispatches via HTTPDoer, decodes the Result envelope, and
// translates non-OK responses into typed errors.
//
// It is generic over both request and response types. Methods with no
// parameters may pass a nil Req; the helper sends "{}" in that case so
// Telegram receives a syntactically valid empty object.
//
// Call is exported because the api package (which lives outside this one)
// invokes it from generated method wrappers. User code should not normally
// call it directly — use the typed wrappers in package api instead.
func Call[Req any, Resp any](ctx context.Context, b *Bot, method string, req Req) (Resp, error) {
	var zero Resp

	if mp, ok := any(req).(multipartRequest); ok {
		if mp == nil {
			return zero, &ParseError{Err: errors.New("client: nil multipart request")}
		}
		if mp.HasFile() {
			return callMultipart[Resp](ctx, b, method, mp)
		}
	}

	body, pooledReqBuf, err := encodeJSONBody(b.codec, req)
	if err != nil {
		return zero, err
	}
	if pooledReqBuf != nil {
		defer putReqBuf(pooledReqBuf)
	}

	httpReq, err := b.buildRequest(ctx, method, body)
	if err != nil {
		return zero, &NetworkError{Err: err}
	}

	resp, err := b.http.Do(httpReq)
	if err != nil {
		// Surface ctx errors faithfully so callers can errors.Is(err, ctx.Err()).
		if ctxErr := ctx.Err(); ctxErr != nil {
			return zero, ctxErr
		}
		return zero, &NetworkError{Err: err}
	}
	defer func() { _ = resp.Body.Close() }()

	buf := respBufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer putRespBuf(buf)
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return zero, &NetworkError{Err: err}
	}

	return decodeResult[Resp](b.codec, buf.Bytes())
}

// CallRaw is like Call but returns the raw JSON of the result field
// instead of decoding it into a typed value. Generated method wrappers
// for sealed-interface return types (ChatMember, MenuButton, etc.) use
// this helper, then dispatch through the union's UnmarshalXxx function.
//
// CallRaw still translates non-OK responses into *APIError just like Call.
func CallRaw[Req any](ctx context.Context, b *Bot, method string, req Req) (json.RawMessage, error) {
	if mp, ok := any(req).(multipartRequest); ok {
		if mp == nil {
			return nil, &ParseError{Err: errors.New("client: nil multipart request")}
		}
		if mp.HasFile() {
			return callMultipartRaw(ctx, b, method, mp)
		}
	}

	body, pooledReqBuf, err := encodeJSONBody(b.codec, req)
	if err != nil {
		return nil, err
	}
	if pooledReqBuf != nil {
		defer putReqBuf(pooledReqBuf)
	}

	httpReq, err := b.buildRequest(ctx, method, body)
	if err != nil {
		return nil, &NetworkError{Err: err}
	}

	resp, err := b.http.Do(httpReq)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, &NetworkError{Err: err}
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &NetworkError{Err: err}
	}

	return decodeResultRaw(b.codec, raw)
}

// decodeResultRaw is decodeResult's sibling that returns the raw result
// field instead of typing it.
func decodeResultRaw(codec Codec, raw []byte) (json.RawMessage, error) {
	var env Result[json.RawMessage]
	if err := codec.Unmarshal(raw, &env); err != nil {
		return nil, &ParseError{Err: err, Body: copyBody(raw)}
	}
	if !env.OK {
		return nil, mapAPIError(env.ErrorCode, env.Description, env.Parameters)
	}
	return env.Result, nil
}

// buildRequest constructs the *http.Request for an API call. When the bot
// has a cached parsed base URL (the common path), the request is built
// manually so that net/url.Parse and net/http.NewRequestWithContext's
// internal book-keeping are skipped — saving allocations on every call.
//
// ContentLength and GetBody are populated from the body's concrete type
// in bodyToReadCloser so RetryDoer can replay the body across attempts.
func (b *Bot) buildRequest(ctx context.Context, method string, body io.Reader) (*http.Request, error) {
	if b.baseURL == nil {
		// Slow path: WithBaseURL configured an unparsable URL (or New ran
		// before pre-parse for some reason). Fall back to the stdlib
		// constructor so we still produce a valid request.
		url := b.base + b.pathPrefix + method
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
		if err != nil {
			return nil, err
		}
		req.Header["Content-Type"] = headerJSONValue
		req.Header["Accept"] = headerJSONValue
		return req, nil
	}

	// Fast path: clone the cached *url.URL by value, set the per-method
	// path. Constructing &http.Request{} directly avoids the Header,
	// URL-parse, and ContentLength bookkeeping that NewRequestWithContext
	// runs unconditionally.
	u := *b.baseURL
	u.Path = b.pathPrefix + method
	u.RawPath = ""

	rc, contentLength, getBody := bodyToReadCloser(body)
	req := &http.Request{
		Method:        http.MethodPost,
		URL:           &u,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        http.Header{"Content-Type": headerJSONValue, "Accept": headerJSONValue},
		Body:          rc,
		GetBody:       getBody,
		ContentLength: contentLength,
		Host:          u.Host,
	}
	return req.WithContext(ctx), nil
}

// bufferReadCloser exposes a *bytes.Buffer as io.ReadCloser without going
// through io.NopCloser. Keeping the concrete *bytes.Buffer accessible lets
// alternative HTTPDoers (e.g. FastHTTPDoer) type-assert and pass the
// underlying bytes through to their native body-set APIs without copying.
type bufferReadCloser struct {
	*bytes.Buffer
}

func (bufferReadCloser) Close() error { return nil }

// readerReadCloser is the equivalent wrapper for *bytes.Reader (used by
// the Marshal fallback path when the codec doesn't implement BodyEncoder).
type readerReadCloser struct {
	*bytes.Reader
}

func (readerReadCloser) Close() error { return nil }

// bodyToReadCloser wraps body for assignment to *http.Request.Body. The
// type switch covers the body shapes encodeJSONBody returns: a pooled
// *bytes.Buffer (BodyEncoder path or {} fast path) or a *bytes.Reader
// (Marshal fallback for codecs that don't implement BodyEncoder). Both
// cases populate ContentLength and GetBody so RetryDoer can replay the
// body across retry attempts without buffering it again.
func bodyToReadCloser(body io.Reader) (io.ReadCloser, int64, func() (io.ReadCloser, error)) {
	switch v := body.(type) {
	case *bytes.Buffer:
		buf := v.Bytes()
		length := int64(len(buf))
		return bufferReadCloser{v}, length, func() (io.ReadCloser, error) {
			return readerReadCloser{bytes.NewReader(buf)}, nil
		}
	case *bytes.Reader:
		length := int64(v.Len())
		// Snapshot the reader's current data so GetBody returns a fresh one.
		snapshot := *v
		return readerReadCloser{v}, length, func() (io.ReadCloser, error) {
			s := snapshot
			return readerReadCloser{&s}, nil
		}
	default:
		// Unknown reader: no length, no replay. Should not happen with the
		// current encodeJSONBody body shapes but kept for forward safety.
		return io.NopCloser(body), -1, nil
	}
}

// encodeJSONBody marshals req into a JSON body. It returns the body
// reader plus, when the codec satisfies BodyEncoder, the pooled buffer
// that backs it — callers MUST return that buffer to the pool via
// putReqBuf once the request is done. The buffer return is exposed
// directly (instead of a closure) so encodeJSONBody allocates nothing
// on the pooled path beyond the codec's own internal allocations.
//
// The {} fast path used for nil/nil-pointer requests bypasses the pool
// entirely; the 2-byte literal isn't worth the contention overhead.
func encodeJSONBody(codec Codec, req any) (io.Reader, *bytes.Buffer, error) {
	if req == nil || isNilPointer(req) {
		return bytes.NewBufferString("{}"), nil, nil
	}
	if enc, ok := codec.(BodyEncoder); ok {
		buf := reqBufPool.Get().(*bytes.Buffer)
		buf.Reset()
		if err := enc.MarshalTo(buf, req); err != nil {
			putReqBuf(buf)
			return nil, nil, &ParseError{Err: err}
		}
		return buf, buf, nil
	}
	data, err := codec.Marshal(req)
	if err != nil {
		return nil, nil, &ParseError{Err: err}
	}
	return bytes.NewReader(data), nil, nil
}

// decodeResult unmarshals raw into Result[Resp] and translates non-OK
// responses into *APIError.
//
// Bool fast path: ~60% of Telegram methods return bool. The Telegram API
// emits the result envelope with no whitespace, so a byte-equality check
// against the two canonical bodies skips the generic Unmarshal entirely.
// Anything that doesn't match exactly (e.g. responses with extra fields,
// errors) falls through to the slow path.
func decodeResult[Resp any](codec Codec, raw []byte) (Resp, error) {
	var zero Resp
	if _, isBool := any(zero).(bool); isBool {
		switch {
		case bytes.Equal(raw, rawOKTrueBody):
			return any(true).(Resp), nil
		case bytes.Equal(raw, rawOKFalseBody):
			return any(false).(Resp), nil
		}
	}
	var env Result[Resp]
	if err := codec.Unmarshal(raw, &env); err != nil {
		return zero, &ParseError{Err: err, Body: copyBody(raw)}
	}
	if !env.OK {
		return zero, mapAPIError(env.ErrorCode, env.Description, env.Parameters)
	}
	return env.Result, nil
}

// isNilPointer returns true when v is a typed nil pointer (the interface
// itself is non-nil because it carries a type, but the underlying value
// is nil). One reflect call per request; not on a hot path that demands
// allocation-freedom.
func isNilPointer(v any) bool {
	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Ptr && rv.IsNil()
}

func copyBody(b []byte) []byte {
	const max = 4096
	if len(b) > max {
		b = b[:max]
	}
	out := make([]byte, len(b))
	copy(out, b)
	return out
}
