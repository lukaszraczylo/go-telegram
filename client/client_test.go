package client

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNew_Defaults(t *testing.T) {
	b := New("123:abc")
	require.Equal(t, "123:abc", b.token)
	require.Equal(t, defaultBaseURL, b.base)
	require.NotNil(t, b.http)
	require.NotNil(t, b.codec)
	require.NotNil(t, b.logger)
}

func TestNew_OptionsApplied(t *testing.T) {
	custom := &http.Client{}
	type fakeCodec struct{ DefaultCodec }
	c := fakeCodec{}

	b := New("t",
		WithHTTPClient(custom),
		WithCodec(c),
		WithBaseURL("https://example.test"),
		WithLogger(NoopLogger{}),
	)
	require.Same(t, custom, b.http)
	require.Equal(t, c, b.codec)
	require.Equal(t, "https://example.test", b.base)
}

func TestResultRoundTrip(t *testing.T) {
	in := Result[int64]{OK: true, Result: 42}
	data, err := DefaultCodec{}.Marshal(in)
	require.NoError(t, err)
	var out Result[int64]
	require.NoError(t, DefaultCodec{}.Unmarshal(data, &out))
	require.Equal(t, in, out)
}
