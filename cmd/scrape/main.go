// Command scrape parses the Telegram Bot API HTML page into the IR
// (internal/spec.API) and writes it to internal/spec/api.json.
//
// Usage:
//
//	scrape -input <file>            (read HTML from local file)
//	scrape -url   <url>             (fetch HTML from URL; default: live docs)
//	scrape -output <file>           (output path; default: internal/spec/api.json)
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/lukaszraczylo/go-telegram/internal/spec"
)

const defaultURL = "https://core.telegram.org/bots/api"

func main() {
	input := flag.String("input", "", "local HTML file (overrides -url)")
	url := flag.String("url", defaultURL, "URL to fetch HTML from")
	output := flag.String("output", "internal/spec/api.json", "output path")
	overridesPath := flag.String("overrides", "internal/spec/overrides.json", "path to overrides JSON")
	flag.Parse()

	if err := run(*input, *url, *output, *overridesPath); err != nil {
		fmt.Fprintln(os.Stderr, "scrape:", err)
		os.Exit(1)
	}
}

func run(input, url, output, overridesPath string) error {
	htmlBytes, err := readHTML(input, url)
	if err != nil {
		return fmt.Errorf("read html: %w", err)
	}

	api, err := scrape(htmlBytes)
	if err != nil {
		return fmt.Errorf("scrape: %w", err)
	}

	overrides, err := spec.LoadOverrides(overridesPath)
	if err != nil {
		return fmt.Errorf("load overrides: %w", err)
	}
	overrides.Apply(api)

	return writeJSON(output, api)
}

func readHTML(input, url string) ([]byte, error) {
	if input != "" {
		return os.ReadFile(input)
	}
	c := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "go-telegram codegen scraper")
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(resp.Status)
	}
	return io.ReadAll(resp.Body)
}
