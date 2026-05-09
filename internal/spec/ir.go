// Package spec defines the intermediate representation produced by the
// Telegram Bot API scraper (cmd/scrape) and consumed by the code generator
// (cmd/genapi). It is committed as internal/spec/api.json so PR diffs read
// as a Telegram changelog.
package spec

import "fmt"

// API is the top-level IR document.
type API struct {
	// Version is the Telegram Bot API version parsed from the "Recent changes" section of the docs page.
	Version string `json:"version"`
	// Types lists all object types in declaration order.
	Types []TypeDecl `json:"types"`
	// Methods lists all API methods in declaration order.
	Methods []MethodDecl `json:"methods"`
}

// TypeDecl describes a Telegram object type.
type TypeDecl struct {
	Name   string  `json:"name"`
	Doc    string  `json:"doc,omitempty"`
	Fields []Field `json:"fields,omitempty"`
	// OneOf, when non-empty, indicates this type is a union and lists the concrete variant type names.
	// Variants are emitted as concrete structs implementing a sealed interface.
	OneOf []string `json:"one_of,omitempty"`
}

// MethodDecl describes a Telegram API method.
type MethodDecl struct {
	Name    string  `json:"name"`
	Doc     string  `json:"doc,omitempty"`
	Params  []Field `json:"params,omitempty"`
	Returns TypeRef `json:"returns"`
	// HasFiles is true when any parameter accepts an InputFile, requiring a multipart/form-data request.
	HasFiles bool `json:"has_files,omitempty"`
}

// Field describes a single field on a type or a single parameter on a method.
type Field struct {
	// Name is the Go-style identifier (e.g. "ChatID").
	Name string `json:"name"`
	// JSONName is the wire name (e.g. "chat_id").
	JSONName string  `json:"json_name"`
	Type     TypeRef `json:"type"`
	Required bool    `json:"required,omitempty"`
	Doc      string  `json:"doc,omitempty"`
}

// Kind enumerates TypeRef shapes.
type Kind int

const (
	// KindPrimitive: int64, string, bool, float64.
	KindPrimitive Kind = iota
	// KindNamed: a TypeDecl by name.
	KindNamed
	// KindArray: ElemType is the element type.
	KindArray
	// KindOneOf: Variants lists discriminant union members.
	KindOneOf
)

// String returns a stable, lowercase representation suitable for serialisation.
func (k Kind) String() string {
	switch k {
	case KindPrimitive:
		return "primitive"
	case KindNamed:
		return "named"
	case KindArray:
		return "array"
	case KindOneOf:
		return "oneOf"
	default:
		return "unknown"
	}
}

// MarshalText / UnmarshalText keep JSON output human-readable.
func (k Kind) MarshalText() ([]byte, error) { return []byte(k.String()), nil }

func (k *Kind) UnmarshalText(b []byte) error {
	switch string(b) {
	case "primitive":
		*k = KindPrimitive
	case "named":
		*k = KindNamed
	case "array":
		*k = KindArray
	case "oneOf":
		*k = KindOneOf
	default:
		return fmt.Errorf("unknown Kind: %q", string(b))
	}
	return nil
}

// TypeRef is a structural reference used wherever a Field type is expressed.
type TypeRef struct {
	Kind     Kind     `json:"kind"`
	Name     string   `json:"name,omitempty"`
	ElemType *TypeRef `json:"elem_type,omitempty"`
	Variants []string `json:"variants,omitempty"`
}
