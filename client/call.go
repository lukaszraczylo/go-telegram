package client

import (
	"bytes"
	"context"
	"errors"
	"github.com/goccy/go-json"
	"io"
	"net/http"
	"reflect"
)

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

	body, err := encodeJSONBody(b.codec, req)
	if err != nil {
		return zero, err
	}

	url := b.base + "/bot" + b.token + "/" + method
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return zero, &NetworkError{Err: err}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := b.http.Do(httpReq)
	if err != nil {
		// Surface ctx errors faithfully so callers can errors.Is(err, ctx.Err()).
		if ctxErr := ctx.Err(); ctxErr != nil {
			return zero, ctxErr
		}
		return zero, &NetworkError{Err: err}
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return zero, &NetworkError{Err: err}
	}

	return decodeResult[Resp](b.codec, raw)
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

	body, err := encodeJSONBody(b.codec, req)
	if err != nil {
		return nil, err
	}

	url := b.base + "/bot" + b.token + "/" + method
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, &NetworkError{Err: err}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

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

// encodeJSONBody marshals req to a JSON body. A nil interface or nil
// pointer req yields "{}" so Telegram receives a valid empty object.
func encodeJSONBody(codec Codec, req any) (io.Reader, error) {
	if req == nil || isNilPointer(req) {
		return bytes.NewBufferString("{}"), nil
	}
	data, err := codec.Marshal(req)
	if err != nil {
		return nil, &ParseError{Err: err}
	}
	return bytes.NewReader(data), nil
}

// decodeResult unmarshals raw into Result[Resp] and translates non-OK
// responses into *APIError.
func decodeResult[Resp any](codec Codec, raw []byte) (Resp, error) {
	var zero Resp
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
