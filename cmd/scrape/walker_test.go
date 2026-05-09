package main

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

func parse(t *testing.T, path string) *html.Node {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()
	doc, err := html.Parse(f)
	require.NoError(t, err)
	return doc
}

func TestWalk_FixtureSections(t *testing.T) {
	doc := parse(t, "../../testdata/html/small_fixture.html")
	sections := walk(doc)

	titles := make([]string, 0, len(sections))
	for _, s := range sections {
		titles = append(titles, s.Title)
	}

	require.Contains(t, titles, "User")
	require.Contains(t, titles, "ChatMember")
	require.Contains(t, titles, "getMe")
	require.Contains(t, titles, "sendMessage")
	require.Contains(t, titles, "sendDocument")
	require.Contains(t, titles, "getUpdates")
	require.Contains(t, titles, "Bot API 7.10")
}

func TestIsMethodTitle(t *testing.T) {
	require.True(t, isMethodTitle("sendMessage"))
	require.True(t, isMethodTitle("getMe"))
	require.False(t, isMethodTitle("Message"))
	require.False(t, isMethodTitle(""))
	require.False(t, isMethodTitle("Bot API 7.10"))
}

func TestIsTypeTitle(t *testing.T) {
	require.True(t, isTypeTitle("Message"))
	require.True(t, isTypeTitle("ChatMember"))
	require.False(t, isTypeTitle("sendMessage"))
	require.False(t, isTypeTitle("Bot API 7.10"))
	require.False(t, isTypeTitle(""))
}

func TestSection_DescriptionAndTables(t *testing.T) {
	doc := parse(t, "../../testdata/html/small_fixture.html")
	sections := walk(doc)
	var sm *section
	for i, s := range sections {
		if s.Title == "sendMessage" {
			sm = &sections[i]
			break
		}
	}
	require.NotNil(t, sm)
	require.True(t, strings.Contains(sm.Description, "Use this method to send text messages"))
	require.Len(t, sm.Tables, 1)
}
