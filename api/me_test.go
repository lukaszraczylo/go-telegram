package api

import (
	"context"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/lukaszraczylo/go-telegram/client"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMeCache_FetchesOnce(t *testing.T) {
	m := &mockDoer{}
	var calls atomic.Int32
	m.On("Do", mock.MatchedBy(func(r *http.Request) bool {
		if strings.HasSuffix(r.URL.Path, "/getMe") {
			calls.Add(1)
			return true
		}
		return false
	})).Return(newJSONResp(200, `{"ok":true,"result":{"id":1,"is_bot":true,"first_name":"echo","username":"echo_bot"}}`), nil)

	bot := client.New("t", client.WithHTTPClient(m))
	var cache MeCache

	me1, err := cache.Get(context.Background(), bot)
	require.NoError(t, err)
	require.Equal(t, "echo_bot", me1.Username)

	me2, err := cache.Get(context.Background(), bot)
	require.NoError(t, err)
	require.Same(t, me1, me2)
	require.Equal(t, int32(1), calls.Load(), "should fetch only once")
}

func TestMeCache_Reset(t *testing.T) {
	var calls atomic.Int32
	m := &mockDoer{}
	m.On("Do", mock.Anything).Run(func(args mock.Arguments) {
		calls.Add(1)
	}).Return(newJSONResp(200, `{"ok":true,"result":{"id":1,"is_bot":true,"first_name":"echo","username":"echo_bot"}}`), nil).Once()
	m.On("Do", mock.Anything).Run(func(args mock.Arguments) {
		calls.Add(1)
	}).Return(newJSONResp(200, `{"ok":true,"result":{"id":1,"is_bot":true,"first_name":"echo","username":"echo_bot"}}`), nil).Once()

	bot := client.New("t", client.WithHTTPClient(m))
	var cache MeCache

	_, err := cache.Get(context.Background(), bot)
	require.NoError(t, err)
	cache.Reset()
	_, err = cache.Get(context.Background(), bot)
	require.NoError(t, err)
	require.Equal(t, int32(2), calls.Load())
}
