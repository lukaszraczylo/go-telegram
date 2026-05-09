package api

import (
	"bytes"
	"io"
	"net/http"

	"github.com/stretchr/testify/mock"
)

// mockDoer is a testify-mock HTTPDoer shared by hand-written tests.
type mockDoer struct{ mock.Mock }

func (m *mockDoer) Do(r *http.Request) (*http.Response, error) {
	args := m.Called(r)
	if v := args.Get(0); v != nil {
		return v.(*http.Response), args.Error(1)
	}
	return nil, args.Error(1)
}

// newJSONResp constructs an *http.Response with a JSON body.
func newJSONResp(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}
