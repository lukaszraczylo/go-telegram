// Package client provides HTTP client primitives for the Telegram Bot API.
package client

import "github.com/goccy/go-json"

// Codec encodes/decodes JSON payloads exchanged with the Telegram Bot API.
// The default implementation wraps goccy/go-json. Users may plug in
// bytedance/sonic or any compatible encoder by passing
// WithCodec to New.
type Codec interface {
	Marshal(v any) ([]byte, error)
	Unmarshal(data []byte, v any) error
}

// DefaultCodec wraps goccy/go-json. It is the zero-value safe default.
type DefaultCodec struct{}

// Marshal calls json.Marshal.
func (DefaultCodec) Marshal(v any) ([]byte, error) { return json.Marshal(v) }

// Unmarshal calls json.Unmarshal.
func (DefaultCodec) Unmarshal(data []byte, v any) error { return json.Unmarshal(data, v) }
