package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "update golden files")

func TestScrape_Golden_SmallFixture(t *testing.T) {
	htmlBytes, err := os.ReadFile("../../testdata/html/small_fixture.html")
	require.NoError(t, err)

	api, err := scrape(htmlBytes)
	require.NoError(t, err)

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	require.NoError(t, enc.Encode(api))

	goldenPath := "../../testdata/golden/api_small_fixture.json"
	if *update {
		require.NoError(t, os.WriteFile(goldenPath, buf.Bytes(), 0o644))
		return
	}
	expected, err := os.ReadFile(goldenPath)
	require.NoError(t, err, "missing golden; run `go test -update ./cmd/scrape/...` to create")
	require.Equal(t, string(expected), buf.String())
}
