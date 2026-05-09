package client

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockDoer struct{ mock.Mock }

func (m *mockDoer) Do(r *http.Request) (*http.Response, error) {
	args := m.Called(r)
	if v := args.Get(0); v != nil {
		return v.(*http.Response), args.Error(1)
	}
	return nil, args.Error(1)
}

func newResp(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}

type echoReq struct {
	ChatID int64  `json:"chat_id"`
	Text   string `json:"text"`
}
type echoResp struct {
	MessageID int64 `json:"message_id"`
}

func TestCall_Success(t *testing.T) {
	m := &mockDoer{}
	m.On("Do", mock.MatchedBy(func(r *http.Request) bool {
		if !strings.HasSuffix(r.URL.Path, "/bot123:abc/sendEcho") {
			return false
		}
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(r.Body)
		return strings.Contains(buf.String(), `"chat_id":42`)
	})).Return(newResp(200, `{"ok":true,"result":{"message_id":7}}`), nil)

	b := New("123:abc", WithHTTPClient(m))
	out, err := Call[*echoReq, *echoResp](context.Background(), b, "sendEcho", &echoReq{ChatID: 42, Text: "hi"})
	require.NoError(t, err)
	require.Equal(t, int64(7), out.MessageID)
	m.AssertExpectations(t)
}

func TestCall_APIError(t *testing.T) {
	m := &mockDoer{}
	m.On("Do", mock.Anything).Return(
		newResp(200, `{"ok":false,"error_code":429,"description":"Too Many Requests: retry after 3","parameters":{"retry_after":3}}`), nil)

	b := New("t", WithHTTPClient(m))
	_, err := Call[*echoReq, *echoResp](context.Background(), b, "x", &echoReq{})
	require.Error(t, err)
	var ae *APIError
	require.ErrorAs(t, err, &ae)
	require.Equal(t, 429, ae.Code)
	require.True(t, ae.IsRetryable())
	require.True(t, errors.Is(err, ErrTooManyRequests))
}

func TestCall_NetworkError(t *testing.T) {
	m := &mockDoer{}
	m.On("Do", mock.Anything).Return(nil, errors.New("dial timeout"))

	b := New("t", WithHTTPClient(m))
	_, err := Call[*echoReq, *echoResp](context.Background(), b, "x", &echoReq{})
	require.Error(t, err)
	var ne *NetworkError
	require.ErrorAs(t, err, &ne)
}

func TestCall_ParseError(t *testing.T) {
	m := &mockDoer{}
	m.On("Do", mock.Anything).Return(newResp(200, `not json`), nil)

	b := New("t", WithHTTPClient(m))
	_, err := Call[*echoReq, *echoResp](context.Background(), b, "x", &echoReq{})
	require.Error(t, err)
	var pe *ParseError
	require.ErrorAs(t, err, &pe)
}

func TestCall_ContextCanceled(t *testing.T) {
	m := &mockDoer{}
	m.On("Do", mock.Anything).Return(nil, context.Canceled).Maybe()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	b := New("t", WithHTTPClient(m))
	_, err := Call[*echoReq, *echoResp](ctx, b, "x", &echoReq{})
	require.ErrorIs(t, err, context.Canceled)
}

func TestCall_NilRequest(t *testing.T) {
	// Methods with no params (e.g. getMe) may pass a nil Req value.
	m := &mockDoer{}
	m.On("Do", mock.MatchedBy(func(r *http.Request) bool {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(r.Body)
		return buf.String() == "{}"
	})).Return(newResp(200, `{"ok":true,"result":{"message_id":0}}`), nil)

	b := New("t", WithHTTPClient(m))
	_, err := Call[*echoReq, *echoResp](context.Background(), b, "x", nil)
	require.NoError(t, err)
}
