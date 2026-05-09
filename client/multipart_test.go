package client

import (
	"context"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type fakeMultipartReq struct {
	chatID int64
	body   string
}

func (f *fakeMultipartReq) HasFile() bool { return true }
func (f *fakeMultipartReq) MultipartFields() map[string]string {
	return map[string]string{"chat_id": "42"}
}
func (f *fakeMultipartReq) MultipartFiles() []MultipartFile {
	return []MultipartFile{{
		FieldName: "document",
		Filename:  "hello.txt",
		Reader:    strings.NewReader(f.body),
	}}
}

type fileResp struct {
	MessageID int64 `json:"message_id"`
}

func TestCallMultipart_Success(t *testing.T) {
	m := &mockDoer{}
	m.On("Do", mock.MatchedBy(func(r *http.Request) bool {
		ct := r.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "multipart/form-data") {
			return false
		}
		_, params, err := mime.ParseMediaType(ct)
		if err != nil {
			return false
		}
		mr := multipart.NewReader(r.Body, params["boundary"])
		seenChat := false
		seenFile := false
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return false
			}
			switch p.FormName() {
			case "chat_id":
				body, _ := io.ReadAll(p)
				seenChat = string(body) == "42"
			case "document":
				body, _ := io.ReadAll(p)
				seenFile = string(body) == "hello world"
			}
		}
		return seenChat && seenFile
	})).Return(newResp(200, `{"ok":true,"result":{"message_id":99}}`), nil)

	b := New("t", WithHTTPClient(m))
	out, err := Call[*fakeMultipartReq, *fileResp](context.Background(), b, "sendDocument", &fakeMultipartReq{chatID: 42, body: "hello world"})
	require.NoError(t, err)
	require.Equal(t, int64(99), out.MessageID)
}

func TestCallMultipart_NoGoroutineLeakOnError(t *testing.T) {
	m := &mockDoer{}
	m.On("Do", mock.Anything).Return(nil, errors.New("dial timeout"))

	b := New("t", WithHTTPClient(m))
	before := runtime.NumGoroutine()

	for i := 0; i < 50; i++ {
		_, _ = Call[*fakeMultipartReq, *fileResp](
			context.Background(), b, "sendDocument",
			&fakeMultipartReq{chatID: 42, body: strings.Repeat("x", 1<<14)},
		)
	}

	// Allow goroutines to finish exiting after Close propagates.
	time.Sleep(50 * time.Millisecond)
	runtime.GC()
	after := runtime.NumGoroutine()

	// A small drift is normal (timers, finalizers); 5 is generous.
	if after-before > 5 {
		t.Fatalf("goroutine leak: before=%d after=%d", before, after)
	}
}
