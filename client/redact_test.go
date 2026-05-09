package client

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRedactToken(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain bot URL", "https://api.telegram.org/bot123456789:ABCdefGHIjklMNOpqrSTUvwxYZ0123456789/getMe",
			"https://api.telegram.org/bot<REDACTED>/getMe"},
		{"in net/http error", `Post "https://api.telegram.org/bot987654321:Z9YxWvUtSrQpOnMlKjIhGfEdCbA9876543210/sendMessage": dial tcp: lookup api.telegram.org: no such host`,
			`Post "https://api.telegram.org/bot<REDACTED>/sendMessage": dial tcp: lookup api.telegram.org: no such host`},
		{"no token", "regular error message", "regular error message"},
		{"underscore + dash in token", "/bot123456789:abc-def_ghi-jkl_mno-pqr_stu-vwx_yz/sendDocument",
			"/bot<REDACTED>/sendDocument"},
		{"too short id (no match)", "/bot123:abc/getMe", "/bot123:abc/getMe"},
		{"too short key (no match)", "/bot123456789:short/getMe", "/bot123456789:short/getMe"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.want, redactToken(c.in))
		})
	}
}

func TestNetworkError_RedactsToken(t *testing.T) {
	inner := errors.New(`Post "https://api.telegram.org/bot1234567890:ABCdefGHIjklMNOpqrSTUvwxYZ0123456789/getMe": dial tcp: timeout`)
	e := &NetworkError{Err: inner}
	require.NotContains(t, e.Error(), "ABCdefGHIjklMNOpqrSTUvwxYZ")
	require.Contains(t, e.Error(), "<REDACTED>")
}

func TestParseError_RedactsToken(t *testing.T) {
	inner := fmt.Errorf(`unexpected response from /bot1234567890:ABCdefGHIjklMNOpqrSTUvwxYZ0123456789/getMe`)
	e := &ParseError{Err: inner, Body: []byte("garbage")}
	require.NotContains(t, e.Error(), "ABCdefGHI")
	require.Contains(t, e.Error(), "<REDACTED>")
}
