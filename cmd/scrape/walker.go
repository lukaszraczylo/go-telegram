package main

import (
	"strings"

	"golang.org/x/net/html"
)

// section is an h4-anchored block of the docs page. Title is the
// heading text (e.g. "User" or "sendMessage"). Description is the
// concatenation of immediately-following <p> paragraphs (until the
// next h4 / h3 / table / list). Tables and Lists hold raw nodes for
// later parsing by the table/oneof extractors.
type section struct {
	Title       string
	Description string
	Tables      []*html.Node // <table> nodes
	Lists       []*html.Node // <ul> nodes (used for oneof variant lists)
}

// walk parses the page and returns sections in document order.
// Sections whose title contains a space (e.g. "Bot API 7.10") are
// included; later passes ignore them or treat them specially.
func walk(doc *html.Node) []section {
	var (
		sections []section
		current  *section
	)

	var visit func(n *html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "h4":
				if current != nil {
					sections = append(sections, *current)
				}
				current = &section{Title: textOf(n)}
				// Don't recurse into the heading; we already have its text.
				return
			case "h3":
				// h3 (e.g. "Available methods") delimits a section;
				// flush the current h4 section but do not start a new one.
				if current != nil {
					sections = append(sections, *current)
					current = nil
				}
				return
			case "p":
				if current != nil {
					if current.Description != "" {
						current.Description += "\n"
					}
					current.Description += strings.TrimSpace(textOf(n))
				}
				return
			case "table":
				if current != nil {
					current.Tables = append(current.Tables, n)
				}
				return
			case "ul":
				if current != nil {
					current.Lists = append(current.Lists, n)
				}
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(doc)
	if current != nil {
		sections = append(sections, *current)
	}
	return sections
}

// textOf returns the concatenated text content of n and descendants,
// with adjacent whitespace collapsed to single spaces.
func textOf(n *html.Node) string {
	var sb strings.Builder
	var w func(*html.Node)
	w = func(n *html.Node) {
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			w(c)
		}
	}
	w(n)
	return collapseWS(sb.String())
}

func collapseWS(s string) string {
	var b strings.Builder
	prevSpace := false
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if !prevSpace {
				b.WriteByte(' ')
			}
			prevSpace = true
			continue
		}
		prevSpace = false
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

// isMethodTitle returns true for headings that look like method names
// (camelCase starting with a lowercase letter; e.g. "sendMessage").
func isMethodTitle(s string) bool {
	if s == "" {
		return false
	}
	r := s[0]
	return r >= 'a' && r <= 'z'
}

// isTypeTitle returns true for headings that look like type names
// (PascalCase; e.g. "Message"). Allows a leading-uppercase only;
// excludes spaces (which would denote a header like "Bot API 7.10").
func isTypeTitle(s string) bool {
	if s == "" {
		return false
	}
	r := s[0]
	if r < 'A' || r > 'Z' {
		return false
	}
	return !strings.Contains(s, " ")
}
