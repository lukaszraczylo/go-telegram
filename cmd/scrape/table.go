package main

import (
	"strings"

	"golang.org/x/net/html"

	"github.com/lukaszraczylo/go-telegram/internal/spec"
)

// parseFieldsTable walks a <table> for an object-type definition.
// Columns: Field, Type, Description (optional column orders are not
// supported; Telegram's docs use a stable layout).
//
// Optional fields are detected via the "Optional." prefix in the
// description text, which is the documented convention.
func parseFieldsTable(t *html.Node) []spec.Field {
	rows := tableRows(t)
	if len(rows) == 0 {
		return nil
	}
	var fields []spec.Field
	for _, row := range rows[1:] { // skip header
		cells := rowCells(row)
		if len(cells) < 3 {
			continue
		}
		jname := strings.TrimSpace(textOf(cells[0]))
		typeText := strings.TrimSpace(textOf(cells[1]))
		desc := strings.TrimSpace(textOf(cells[2]))

		required := !strings.HasPrefix(desc, "Optional.")
		tref := parseTypeRef(typeText)
		var enumVals []string
		if tref.Kind == spec.KindPrimitive && tref.Name == "string" {
			enumVals = extractEnumValues(jname, desc)
		}
		fields = append(fields, spec.Field{
			Name:       goName(jname),
			JSONName:   jname,
			Type:       tref,
			Required:   required,
			Doc:        desc,
			EnumValues: enumVals,
		})
	}
	return fields
}

// parseParamsTable walks a <table> for a method definition.
// Columns: Parameter, Type, Required, Description.
func parseParamsTable(t *html.Node) []spec.Field {
	rows := tableRows(t)
	if len(rows) == 0 {
		return nil
	}
	var params []spec.Field
	for _, row := range rows[1:] {
		cells := rowCells(row)
		if len(cells) < 4 {
			continue
		}
		jname := strings.TrimSpace(textOf(cells[0]))
		typeText := strings.TrimSpace(textOf(cells[1]))
		req := strings.EqualFold(strings.TrimSpace(textOf(cells[2])), "Yes")
		desc := strings.TrimSpace(textOf(cells[3]))

		tref := parseTypeRef(typeText)
		var enumVals []string
		if tref.Kind == spec.KindPrimitive && tref.Name == "string" {
			enumVals = extractEnumValues(jname, desc)
		}
		params = append(params, spec.Field{
			Name:       goName(jname),
			JSONName:   jname,
			Type:       tref,
			Required:   req,
			Doc:        desc,
			EnumValues: enumVals,
		})
	}
	return params
}

// tableRows returns the <tr> children of a <table>, skipping over
// any <thead>/<tbody> wrappers.
func tableRows(t *html.Node) []*html.Node {
	var rows []*html.Node
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			rows = append(rows, n)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(t)
	return rows
}

// rowCells returns the <td> (or <th>) children of a <tr>.
func rowCells(tr *html.Node) []*html.Node {
	var cells []*html.Node
	for c := tr.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && (c.Data == "td" || c.Data == "th") {
			cells = append(cells, c)
		}
	}
	return cells
}

// goName converts a snake_case JSON identifier to PascalCase.
// Special-cases common acronyms used in the Telegram docs.
func goName(s string) string {
	if s == "" {
		return ""
	}
	parts := strings.Split(s, "_")
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		switch p {
		case "id":
			b.WriteString("ID")
		case "url":
			b.WriteString("URL")
		case "ip":
			b.WriteString("IP")
		case "https":
			b.WriteString("HTTPS")
		case "json":
			b.WriteString("JSON")
		case "html":
			b.WriteString("HTML")
		default:
			if p[0] >= 'a' && p[0] <= 'z' {
				b.WriteByte(p[0] - 'a' + 'A')
				b.WriteString(p[1:])
			} else {
				b.WriteString(p)
			}
		}
	}
	return b.String()
}

// parseTypeRef decodes the type-cell text into a spec.TypeRef.
//
// Recognised shapes:
//
//	"Integer"               → primitive int64
//	"String"                → primitive string
//	"Boolean" / "True"      → primitive bool
//	"Float" / "Float number"→ primitive float64
//	"Array of X"            → array of (parseTypeRef of X)
//	"Array of Array of X"   → array of array of X
//	"Foo"                   → named Foo
//	"Foo or Bar"            → oneOf {Foo, Bar}
//	"InputFile or String"   → oneOf (caller may translate to InputFile)
//
// parseTypeRef decodes the type-cell text into a spec.TypeRef.
//
// Recognised shapes:
//
//	"Integer"               → primitive int64
//	"String"                → primitive string
//	"Boolean" / "True"      → primitive bool
//	"Float" / "Float number"→ primitive float64
//	"Array of X"            → array of (parseTypeRef of X)
//	"Array of Array of X"   → array of array of X
//	"Foo"                   → named Foo
//	"Foo or Bar"            → oneOf {Foo, Bar}
//	"Foo, Bar and Baz"      → oneOf {Foo, Bar, Baz} (Telegram's comma+and union form)
//	"InputFile or String"   → oneOf (caller may translate to InputFile)
func parseTypeRef(s string) spec.TypeRef {
	s = strings.TrimSpace(s)
	// Array prefix.
	if rest, ok := strings.CutPrefix(s, "Array of "); ok {
		elem := parseTypeRef(rest)
		return spec.TypeRef{Kind: spec.KindArray, ElemType: &elem}
	}
	// Comma-and union ("X, Y, Z and W") — used by Telegram for ≥3-variant unions.
	if strings.Contains(s, ", ") && strings.Contains(s, " and ") {
		parts := splitCommaAnd(s)
		variants := make([]string, 0, len(parts))
		for _, p := range parts {
			variants = append(variants, primitiveOrNamed(strings.TrimSpace(p)).Name)
		}
		return spec.TypeRef{Kind: spec.KindOneOf, Variants: variants}
	}
	// "X or Y" union (the 2-variant form).
	if strings.Contains(s, " or ") {
		parts := strings.Split(s, " or ")
		variants := make([]string, 0, len(parts))
		for _, p := range parts {
			variants = append(variants, primitiveOrNamed(strings.TrimSpace(p)).Name)
		}
		return spec.TypeRef{Kind: spec.KindOneOf, Variants: variants}
	}
	return primitiveOrNamed(s)
}

// splitCommaAnd splits "A, B, C and D" → ["A", "B", "C", "D"].
func splitCommaAnd(s string) []string {
	// Replace " and " with ", " then split on ", ".
	s = strings.ReplaceAll(s, " and ", ", ")
	parts := strings.Split(s, ", ")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// primitiveOrNamed maps a single-word type cell to either a primitive
// or a named TypeRef. Unrecognised words are treated as named types.
func primitiveOrNamed(s string) spec.TypeRef {
	switch s {
	case "Integer", "Int":
		return spec.TypeRef{Kind: spec.KindPrimitive, Name: "int64"}
	case "String":
		return spec.TypeRef{Kind: spec.KindPrimitive, Name: "string"}
	case "Boolean", "Bool", "True", "False":
		return spec.TypeRef{Kind: spec.KindPrimitive, Name: "bool"}
	case "Float", "Float number":
		return spec.TypeRef{Kind: spec.KindPrimitive, Name: "float64"}
	default:
		return spec.TypeRef{Kind: spec.KindNamed, Name: s}
	}
}
