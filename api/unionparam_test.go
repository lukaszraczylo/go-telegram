package api

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/lukaszraczylo/go-telegram/client"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestSetMyCommands_BotCommandScope_NoPointerToInterface is a regression
// test for the bug where sealed-interface union types without an
// auto-decode discriminator (BotCommandScope, InputMedia, etc.) were
// emitted as `*<Union>` (pointer-to-interface) when used as optional
// fields. Pointer-to-interface is a Go anti-pattern: the interface is
// already nil-able, and callers were forced to write
// `Scope: &someConcreteScope` instead of `Scope: someConcreteScope`.
//
// This test confirms the field is now bare-interface-typed: a concrete
// variant assigns directly, and a nil scope omits the field via
// omitempty.
func TestSetMyCommands_BotCommandScope_NoPointerToInterface(t *testing.T) {
	var captured string
	m := &mockDoer{}
	m.On("Do", mock.MatchedBy(func(r *http.Request) bool {
		if r.Body != nil {
			b, _ := io.ReadAll(r.Body)
			captured = string(b)
		}
		return true
	})).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(`{"ok":true,"result":true}`)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}, nil)

	bot := client.New("test:token", client.WithHTTPClient(m))

	// Direct assignment of a concrete variant — only possible when Scope
	// is `BotCommandScope` (interface), not `*BotCommandScope`.
	ok, err := SetMyCommands(context.Background(), bot, &SetMyCommandsParams{
		Commands: []BotCommand{{Command: "start", Description: "begin"}},
		Scope:    &BotCommandScopeAllPrivateChats{},
	})
	require.NoError(t, err)
	require.True(t, ok)
	require.Contains(t, captured, `"scope":{"type":"all_private_chats"}`)
}

// TestSetMyCommands_NilScope_OmitsField confirms omitempty works on the
// bare-interface field when the caller doesn't supply a scope.
func TestSetMyCommands_NilScope_OmitsField(t *testing.T) {
	var captured string
	m := &mockDoer{}
	m.On("Do", mock.MatchedBy(func(r *http.Request) bool {
		if r.Body != nil {
			b, _ := io.ReadAll(r.Body)
			captured = string(b)
		}
		return true
	})).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(`{"ok":true,"result":true}`)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}, nil)

	bot := client.New("test:token", client.WithHTTPClient(m))
	_, err := SetMyCommands(context.Background(), bot, &SetMyCommandsParams{
		Commands: []BotCommand{{Command: "start", Description: "begin"}},
	})
	require.NoError(t, err)
	require.NotContains(t, captured, `"scope"`, "nil scope must be omitted from JSON")
}
