package client

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
)

// stubDoer returns the same canned response body for every request. It
// is intentionally minimal — testify mock has its own overhead that
// would dominate the per-call cost we want to measure.
type stubDoer struct{ body []byte }

func (s *stubDoer) Do(_ *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(s.body)),
		Header:     http.Header{},
	}, nil
}

type benchSendReq struct {
	ChatID int64  `json:"chat_id"`
	Text   string `json:"text"`
}

type benchMsgResp struct {
	MessageID int64  `json:"message_id"`
	Date      int64  `json:"date"`
	Text      string `json:"text"`
}

func BenchmarkCall_BoolResponse(b *testing.B) {
	d := &stubDoer{body: []byte(`{"ok":true,"result":true}`)}
	bot := New("123:abc", WithHTTPClient(d))
	ctx := context.Background()
	req := &benchSendReq{ChatID: 42, Text: "hi"}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := Call[*benchSendReq, bool](ctx, bot, "setMyCommands", req); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCall_StructResponse(b *testing.B) {
	d := &stubDoer{body: []byte(`{"ok":true,"result":{"message_id":1,"date":0,"text":"ok"}}`)}
	bot := New("123:abc", WithHTTPClient(d))
	ctx := context.Background()
	req := &benchSendReq{ChatID: 42, Text: "hi"}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := Call[*benchSendReq, benchMsgResp](ctx, bot, "sendMessage", req); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEncodeJSONBody(b *testing.B) {
	codec := DefaultCodec{}
	req := &benchSendReq{ChatID: 42, Text: "hello, world"}
	b.ReportAllocs()
	for b.Loop() {
		r, pooled, err := encodeJSONBody(codec, req)
		if err != nil {
			b.Fatal(err)
		}
		_ = r
		if pooled != nil {
			putReqBuf(pooled)
		}
	}
}

func BenchmarkDecodeResult_Bool(b *testing.B) {
	codec := DefaultCodec{}
	raw := []byte(`{"ok":true,"result":true}`)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := decodeResult[bool](codec, raw); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecodeResult_Struct(b *testing.B) {
	codec := DefaultCodec{}
	raw := []byte(`{"ok":true,"result":{"message_id":1,"date":0,"text":"ok"}}`)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := decodeResult[benchMsgResp](codec, raw); err != nil {
			b.Fatal(err)
		}
	}
}
