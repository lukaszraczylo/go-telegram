package client

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// client.go option getters
// ---------------------------------------------------------------------------

func TestBot_Getters(t *testing.T) {
	b := New("mytoken",
		WithBaseURL("http://localhost:9999"),
		WithCodec(DefaultCodec{}),
		WithLogger(NoopLogger{}),
	)
	require.Equal(t, "mytoken", b.Token())
	require.Equal(t, "http://localhost:9999", b.BaseURL())
	require.NotNil(t, b.HTTP())
	require.NotNil(t, b.Codec())
	require.NotNil(t, b.Logger())
}

func TestWithLogger_NilBecomesNoop(t *testing.T) {
	b := New("t", WithLogger(nil))
	require.IsType(t, NoopLogger{}, b.Logger())
}

func TestNoopLogger_AllMethods(t *testing.T) {
	l := NoopLogger{}
	// None of these should panic.
	l.Debug("msg")
	l.Info("msg", "k", "v")
	l.Warn("msg")
	l.Error("msg", "err", "oops")
}

// ---------------------------------------------------------------------------
// RetryOption setters
// ---------------------------------------------------------------------------

func TestRetryOptions_Applied(t *testing.T) {
	d := NewRetryDoer(nil,
		WithMaxAttempts(7),
		WithBaseBackoff(1*time.Second),
		WithMaxBackoff(60*time.Second),
		WithBackoffFactor(3.0),
		WithJitter(0.5),
	)
	require.Equal(t, 7, d.maxAttempts)
	require.Equal(t, 1*time.Second, d.base)
	require.Equal(t, 60*time.Second, d.max)
	require.Equal(t, 3.0, d.factor)
	require.Equal(t, 0.5, d.jitter)
}

// ---------------------------------------------------------------------------
// RetryDoer.delay — override path
// ---------------------------------------------------------------------------

func TestRetryDoer_DelayOverride(t *testing.T) {
	d := NewRetryDoer(nil)
	got := d.delay(1, 5*time.Second)
	require.Equal(t, 5*time.Second, got)
}

func TestRetryDoer_DelayExponential(t *testing.T) {
	d := NewRetryDoer(nil,
		WithBaseBackoff(100*time.Millisecond),
		WithMaxBackoff(10*time.Second),
		WithJitter(0), // no jitter for deterministic test
		WithBackoffFactor(2.0),
	)
	d1 := d.delay(1, 0)
	d2 := d.delay(2, 0)
	require.Greater(t, int64(d2), int64(d1), "backoff should grow")
}

func TestRetryDoer_DelayMaxCap(t *testing.T) {
	d := NewRetryDoer(nil,
		WithBaseBackoff(1*time.Second),
		WithMaxBackoff(2*time.Second),
		WithJitter(0),
		WithBackoffFactor(100.0),
	)
	delay := d.delay(10, 0)
	require.LessOrEqual(t, delay, 2*time.Second)
}

// ---------------------------------------------------------------------------
// errors.go — RetryAfter nil parameters + ParseError.Unwrap
// ---------------------------------------------------------------------------

func TestAPIError_RetryAfterNilParams(t *testing.T) {
	e := &APIError{Code: 429, Description: "Too Many Requests", Parameters: nil}
	require.Equal(t, time.Duration(0), e.RetryAfter())
}

func TestParseError_Unwrap(t *testing.T) {
	inner := errors.New("decode error")
	pe := &ParseError{Err: inner, Body: []byte("body")}
	require.ErrorIs(t, pe, inner)
}

func TestParseError_LongBodyTruncated(t *testing.T) {
	body := bytes.Repeat([]byte("x"), 1000)
	pe := &ParseError{Err: errors.New("e"), Body: body}
	msg := pe.Error()
	// Error() truncates body to 256 for display — should not include all 1000 chars
	require.Less(t, len(msg), 800, "should truncate body in Error()")
}

func TestNetworkError_Unwrap(t *testing.T) {
	inner := errors.New("tcp error")
	ne := &NetworkError{Err: inner}
	require.ErrorIs(t, ne, inner)
}

// ---------------------------------------------------------------------------
// mapAPIError — missing sentinel branches (generic 400, unmapped 500)
// ---------------------------------------------------------------------------

func TestMapAPIError_Generic400(t *testing.T) {
	e := mapAPIError(400, "Bad Request: some unknown thing", nil)
	require.True(t, errors.Is(e, ErrBadRequest))
}

func TestMapAPIError_Unmapped500(t *testing.T) {
	e := mapAPIError(500, "Internal Server Error", nil)
	require.Nil(t, e.sentinel)
	require.Equal(t, 500, e.Code)
}

func TestMapAPIError_403(t *testing.T) {
	e := mapAPIError(403, "Forbidden: bot was blocked", nil)
	require.True(t, errors.Is(e, ErrForbidden))
}

// ---------------------------------------------------------------------------
// callMultipart — ctx cancelled
// ---------------------------------------------------------------------------

func TestCallMultipart_ContextCancelled(t *testing.T) {
	// A doer that blocks then returns context error.
	blocker := &extraBlockingDoer{done: make(chan struct{})}

	b := New("t", WithHTTPClient(blocker))
	ctx, cancel := context.WithCancel(context.Background())

	mp := &extraFakeMultipartReq{
		fields: map[string]string{"chat_id": "1"},
		files: []MultipartFile{
			{FieldName: "document", Filename: "f.txt", Reader: bytes.NewReader([]byte("data"))},
		},
	}

	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
		close(blocker.done)
	}()

	_, err := callMultipart[*struct{}](ctx, b, "sendDocument", mp)
	require.Error(t, err)
}

type extraBlockingDoer struct{ done chan struct{} }

func (b *extraBlockingDoer) Do(r *http.Request) (*http.Response, error) {
	<-b.done
	return nil, r.Context().Err()
}

type extraFakeMultipartReq struct {
	fields map[string]string
	files  []MultipartFile
}

func (f *extraFakeMultipartReq) HasFile() bool                      { return len(f.files) > 0 }
func (f *extraFakeMultipartReq) MultipartFiles() []MultipartFile    { return f.files }
func (f *extraFakeMultipartReq) MultipartFields() map[string]string { return f.fields }

// ---------------------------------------------------------------------------
// copyBody size cap
// ---------------------------------------------------------------------------

func TestCopyBody_LargeBodyCapped(t *testing.T) {
	big := bytes.Repeat([]byte("a"), 8000)
	out := copyBody(big)
	require.Len(t, out, 4096)
}

func TestCopyBody_SmallBody(t *testing.T) {
	small := []byte("hello")
	out := copyBody(small)
	require.Equal(t, small, out)
}

// ---------------------------------------------------------------------------
// Call — 5xx non-200 HTTP status (transport level)
// ---------------------------------------------------------------------------

func TestCall_5xxHTTPStatus(t *testing.T) {
	m := &mockDoer{}
	m.On("Do", mock.Anything).Return(&http.Response{
		StatusCode: 500,
		Body:       io.NopCloser(bytes.NewBufferString(`{"ok":false,"error_code":500,"description":"Internal"}`)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}, nil)

	b := New("t", WithHTTPClient(m))
	_, err := Call[*echoReq, *echoResp](context.Background(), b, "x", &echoReq{})
	require.Error(t, err)
	var ae *APIError
	require.ErrorAs(t, err, &ae)
	require.Equal(t, 500, ae.Code)
}
