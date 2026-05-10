// Package client provides HTTP client primitives for the Telegram Bot API.
package client

import (
	"io"

	"github.com/goccy/go-json"
)

// Codec encodes/decodes JSON payloads exchanged with the Telegram Bot API.
// The default implementation wraps goccy/go-json. Users may plug in
// bytedance/sonic or any compatible encoder by passing
// WithCodec to New.
type Codec interface {
	Marshal(v any) ([]byte, error)
	Unmarshal(data []byte, v any) error
}

// BodyEncoder is an optional Codec extension that encodes directly into
// an io.Writer, skipping the intermediate []byte that Marshal returns.
// Call uses this when present to avoid allocating the marshal result and
// the bytes.Reader that wraps it; codecs without it fall through to
// Marshal + bytes.NewReader.
type BodyEncoder interface {
	MarshalTo(w io.Writer, v any) error
}

// DefaultCodec wraps goccy/go-json. It is the zero-value safe default.
type DefaultCodec struct{}

// Marshal calls json.Marshal.
func (DefaultCodec) Marshal(v any) ([]byte, error) { return json.Marshal(v) }

// Unmarshal calls json.Unmarshal.
func (DefaultCodec) Unmarshal(data []byte, v any) error { return json.Unmarshal(data, v) }

// MarshalTo encodes v into w via goccy/go-json's streaming encoder. The
// trailing newline that Encoder appends is valid JSON whitespace and is
// accepted by Telegram's parser.
func (DefaultCodec) MarshalTo(w io.Writer, v any) error {
	return json.NewEncoder(w).Encode(v)
}
