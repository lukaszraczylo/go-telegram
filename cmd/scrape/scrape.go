package main

import (
	"bytes"
	"fmt"
	"github.com/goccy/go-json"
	"os"

	"golang.org/x/net/html"

	"github.com/lukaszraczylo/go-telegram/internal/spec"
)

// scrape (the package-level implementation overriding the stub in main.go;
// remove the stub from main.go in this task) parses the docs HTML into IR.
func scrape(htmlBytes []byte) (*spec.API, error) {
	doc, err := html.Parse(bytes.NewReader(htmlBytes))
	if err != nil {
		return nil, fmt.Errorf("html parse: %w", err)
	}
	sections := walk(doc)
	api := &spec.API{Version: extractVersion(sections)}
	for _, s := range sections {
		switch {
		case isMethodTitle(s.Title):
			api.Methods = append(api.Methods, methodFromSection(s))
		case isTypeTitle(s.Title):
			api.Types = append(api.Types, typeFromSection(s))
		}
	}
	return api, nil
}

func typeFromSection(s section) spec.TypeDecl {
	td := spec.TypeDecl{Name: s.Title, Doc: s.Description}
	if len(s.Tables) > 0 {
		td.Fields = parseFieldsTable(s.Tables[0])
	} else if len(s.Lists) > 0 {
		// Union: extract variant names from <li><a>...</a></li>.
		td.OneOf = extractListLinks(s.Lists[0])
	}
	return td
}

func methodFromSection(s section) spec.MethodDecl {
	md := spec.MethodDecl{Name: s.Title, Doc: s.Description, Returns: extractReturn(s.Description)}
	if len(s.Tables) > 0 {
		md.Params = parseParamsTable(s.Tables[0])
	}
	md.HasFiles = hasFilesParams(md.Params)
	return md
}

// extractListLinks pulls anchor texts out of a <ul>: each <li><a>X</a></li>
// contributes "X" to the result. Used for union variant lists.
func extractListLinks(ul *html.Node) []string {
	var names []string
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			names = append(names, textOf(n))
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(ul)
	return names
}

// writeJSON marshals the IR with stable, human-readable formatting and
// writes it to path. Marshalling is deterministic: types and methods
// preserve scrape order; struct fields use IR-defined order.
func writeJSON(path string, api *spec.API) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(api); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}
