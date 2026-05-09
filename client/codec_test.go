package client

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultCodec_RoundTrip(t *testing.T) {
	c := DefaultCodec{}
	type payload struct {
		Name string `json:"name"`
		N    int    `json:"n"`
	}
	in := payload{Name: "x", N: 7}
	data, err := c.Marshal(in)
	require.NoError(t, err)
	require.JSONEq(t, `{"name":"x","n":7}`, string(data))

	var out payload
	require.NoError(t, c.Unmarshal(data, &out))
	require.Equal(t, in, out)
}

func TestDefaultCodec_UnmarshalError(t *testing.T) {
	var v map[string]any
	err := DefaultCodec{}.Unmarshal([]byte(`not json`), &v)
	require.Error(t, err)
}
