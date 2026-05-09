package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"github.com/goccy/go-json"
	"go/format"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"text/template"

	"github.com/lukaszraczylo/go-telegram/internal/spec"
)

// Discriminator-value extractors. The curly form ("always “X”") is
// authoritative because Telegram quotes wire literals with curly quotes
// throughout the docs; the bare form ("must be X") is the looser
// non-quoted variant used for BotCommandScope, InputMedia, etc.
var (
	discCurlyRE = regexp.MustCompile(`(?:must be|always)\s+“([^”]+)”`)
	discBareRE  = regexp.MustCompile(`must be\s+([A-Za-z0-9_]+)(?:[\s.,]|$)`)
)

//go:embed types.tmpl
var typesTmpl string

//go:embed methods.tmpl
var methodsTmpl string

//go:embed enums.tmpl
var enumsTmpl string

//go:embed tests.tmpl
var testsTmpl string

// runtimeTypes lists types that are intentionally hand-coded and must not be
// emitted by the code generator. Skipping them prevents collisions between
// generated and hand-coded definitions.
var runtimeTypes = map[string]bool{
	"InputFile":          true,
	"ResponseParameters": true,
	"ChatID":             true,
	"MessageOrBool":      true,
}

// discriminatorSpec describes how to decode a sealed-interface union by
// peeking at a single JSON field.
type discriminatorSpec struct {
	Field    string            // JSON field name to peek at
	Variants map[string]string // discriminator value → concrete Go type name
}

// knownDiscriminators maps parent union name → discriminator spec.
// Used by the template helpers hasDiscriminator / discriminatorField /
// discriminatorMap to emit UnmarshalXxx helpers.
var knownDiscriminators = map[string]discriminatorSpec{
	"ChatMember": {
		Field: "status",
		Variants: map[string]string{
			"creator":       "ChatMemberOwner",
			"administrator": "ChatMemberAdministrator",
			"member":        "ChatMemberMember",
			"restricted":    "ChatMemberRestricted",
			"left":          "ChatMemberLeft",
			"kicked":        "ChatMemberBanned",
		},
	},
	"MessageOrigin": {
		Field: "type",
		Variants: map[string]string{
			"user":        "MessageOriginUser",
			"hidden_user": "MessageOriginHiddenUser",
			"chat":        "MessageOriginChat",
			"channel":     "MessageOriginChannel",
		},
	},
	"ReactionType": {
		Field: "type",
		Variants: map[string]string{
			"emoji":        "ReactionTypeEmoji",
			"custom_emoji": "ReactionTypeCustomEmoji",
			"paid":         "ReactionTypePaid",
		},
	},
	"PaidMedia": {
		Field: "type",
		Variants: map[string]string{
			"preview": "PaidMediaPreview",
			"photo":   "PaidMediaPhoto",
			"video":   "PaidMediaVideo",
		},
	},
	"BackgroundType": {
		Field: "type",
		Variants: map[string]string{
			"fill":       "BackgroundTypeFill",
			"wallpaper":  "BackgroundTypeWallpaper",
			"pattern":    "BackgroundTypePattern",
			"chat_theme": "BackgroundTypeChatTheme",
		},
	},
	"BackgroundFill": {
		Field: "type",
		Variants: map[string]string{
			"solid":             "BackgroundFillSolid",
			"gradient":          "BackgroundFillGradient",
			"freeform_gradient": "BackgroundFillFreeformGradient",
		},
	},
	"ChatBoostSource": {
		Field: "source",
		Variants: map[string]string{
			"premium":   "ChatBoostSourcePremium",
			"gift_code": "ChatBoostSourceGiftCode",
			"giveaway":  "ChatBoostSourceGiveaway",
		},
	},
	"RevenueWithdrawalState": {
		Field: "type",
		Variants: map[string]string{
			"pending":   "RevenueWithdrawalStatePending",
			"succeeded": "RevenueWithdrawalStateSucceeded",
			"failed":    "RevenueWithdrawalStateFailed",
		},
	},
	"TransactionPartner": {
		Field: "type",
		Variants: map[string]string{
			"fragment":     "TransactionPartnerFragment",
			"user":         "TransactionPartnerUser",
			"telegram_ads": "TransactionPartnerTelegramAds",
			"telegram_api": "TransactionPartnerTelegramApi",
			"other":        "TransactionPartnerOther",
		},
	},
	"MenuButton": {
		Field: "type",
		Variants: map[string]string{
			"commands": "MenuButtonCommands",
			"web_app":  "MenuButtonWebApp",
			"default":  "MenuButtonDefault",
		},
	},
	"OwnedGift": {
		Field: "type",
		Variants: map[string]string{
			"regular": "OwnedGiftRegular",
			"unique":  "OwnedGiftUnique",
		},
	},
	"StoryAreaType": {
		Field: "type",
		Variants: map[string]string{
			"location":           "StoryAreaTypeLocation",
			"suggested_reaction": "StoryAreaTypeSuggestedReaction",
			"link":               "StoryAreaTypeLink",
			"weather":            "StoryAreaTypeWeather",
			"unique_gift":        "StoryAreaTypeUniqueGift",
		},
	},
	// MaybeInaccessibleMessage uses an integer discriminator (date field).
	// Variants is nil — the standard template block is skipped; a
	// hand-coded UnmarshalMaybeInaccessibleMessage is emitted instead.
	"MaybeInaccessibleMessage": {
		Field:    "",
		Variants: nil,
	},
}

// emitter renders Go source from a spec.API IR.
type emitter struct {
	api    *spec.API
	outDir string
	enums  *enumPlan
	// variantDiscs maps a concrete variant type name (e.g.
	// "BotCommandScopeAllPrivateChats") to its discriminator wire-field
	// + value. Populated once at construction; consulted by the types
	// template to emit per-variant MarshalJSON that hardcodes the
	// discriminator so callers don't have to set it by hand.
	variantDiscs map[string]variantDiscriminator
}

func newEmitter(api *spec.API, outDir string) *emitter {
	knownInterfaceTypes = buildUnionTypeSet(api)
	return &emitter{
		api:          api,
		outDir:       outDir,
		enums:        planEnums(api),
		variantDiscs: variantDiscriminators(api),
	}
}

// variantDiscriminator describes the JSON field+value that identifies a
// concrete variant of a sealed-interface union on the wire.
type variantDiscriminator struct {
	JSONField string // wire field name, e.g. "type" or "source"
	GoField   string // Go struct field name, e.g. "Type" or "Source"
	Value     string // the wire value, e.g. "all_private_chats"
}

// variantDiscriminators returns variantTypeName → discriminator for every
// concrete struct that participates in a sealed-interface union and has
// a string-typed first field whose doc fixes its value (the canonical
// "must be X" / "always “X”" patterns Telegram uses).
//
// Resolution order:
//
//  1. knownDiscriminators reverse-lookup (the 13 auto-decode unions).
//     This guarantees parity with UnmarshalXxx dispatch for the unions
//     that round-trip through the library.
//  2. Doc-string analysis of the variant's first field, for marker-only
//     unions (BotCommandScope, InputMedia, etc.) where the IR has no
//     explicit discriminator metadata.
//
// Variants whose first field has no discriminator hint (Message,
// InaccessibleMessage, the InputMessageContent family) are omitted —
// the caller writes the dispatching fields directly and Telegram
// identifies them structurally.
func variantDiscriminators(api *spec.API) map[string]variantDiscriminator {
	out := make(map[string]variantDiscriminator, 128)

	// Pass 1: reverse-lookup from knownDiscriminators.
	for _, ds := range knownDiscriminators {
		if ds.Field == "" {
			continue
		}
		for value, variant := range ds.Variants {
			out[variant] = variantDiscriminator{
				JSONField: ds.Field,
				Value:     value,
			}
		}
	}

	// Build the set of every variant type referenced by any OneOf so we
	// can scan only those (avoids matching free-text "must be" prose in
	// non-variant types like Message).
	variantSet := make(map[string]bool, 128)
	for _, t := range api.Types {
		for _, v := range t.OneOf {
			variantSet[v] = true
		}
	}

	// Pass 2: doc-parse for variants without a known discriminator.
	for _, t := range api.Types {
		if !variantSet[t.Name] {
			continue
		}
		if _, ok := out[t.Name]; ok {
			// Pass-1 already provided the wire value; we still need
			// the Go field name (mirrors the JSON field but with
			// proper case). Resolve from t.Fields by JSONName match.
			disc := out[t.Name]
			for _, f := range t.Fields {
				if f.JSONName == disc.JSONField {
					disc.GoField = f.Name
					out[t.Name] = disc
					break
				}
			}
			continue
		}
		disc, ok := extractVariantDiscriminator(t)
		if !ok {
			continue
		}
		out[t.Name] = disc
	}

	// Drop entries we couldn't resolve a Go field for (defensive — every
	// pass-1 hit should have matched, but better to skip than emit
	// broken code referencing an unknown field name).
	for name, d := range out {
		if d.GoField == "" {
			delete(out, name)
		}
	}
	return out
}

// extractVariantDiscriminator inspects the first field of a variant
// struct and returns its discriminator if the field is a required
// string whose doc nails the value via "must be X" or "always “X”".
// Returns (zero, false) when no clear discriminator is present.
func extractVariantDiscriminator(t spec.TypeDecl) (variantDiscriminator, bool) {
	if len(t.Fields) == 0 {
		return variantDiscriminator{}, false
	}
	f := t.Fields[0]
	if !f.Required || f.Type.Kind != spec.KindPrimitive || f.Type.Name != "string" {
		return variantDiscriminator{}, false
	}
	value := parseDiscriminatorDoc(f.Doc)
	if value == "" {
		return variantDiscriminator{}, false
	}
	return variantDiscriminator{
		JSONField: f.JSONName,
		GoField:   f.Name,
		Value:     value,
	}, true
}

// parseDiscriminatorDoc extracts the wire-level discriminator value
// from a field doc string. Handles both Telegram phrasings:
//
//   - "Scope type, must be all_private_chats"           (bare token)
//   - "Type of the message origin, always “user”"      (curly-quoted)
//
// Returns "" when no discriminator is present.
func parseDiscriminatorDoc(doc string) string {
	// Curly-quoted form takes priority: "must be “X”" or "always “X”".
	if m := discCurlyRE.FindStringSubmatch(doc); len(m) == 2 {
		return m[1]
	}
	// Bare-token form: "must be <ident>" terminated by end-of-string,
	// punctuation, or whitespace.
	if m := discBareRE.FindStringSubmatch(doc); len(m) == 2 {
		return m[1]
	}
	return ""
}

// knownInterfaceTypes is the full set of sealed-interface union type names
// (both auto-decoded ones in knownDiscriminators and marker-only ones from
// types with OneOf). Populated at emitter construction. goType and
// unionTypeFor consult this so optional fields of any union type stay
// bare interface, never *Interface (which is meaningless in Go and trips
// users at every call site).
var knownInterfaceTypes = map[string]bool{}

// emitTypes renders types.gen.go.
func (e *emitter) emitTypes() error {
	t, err := template.New("types").Funcs(funcsWithDiscs(e.enums, e.variantDiscs)).Parse(typesTmpl)
	if err != nil {
		return fmt.Errorf("parse types.tmpl: %w", err)
	}
	filtered := *e.api
	filtered.Types = nil
	for _, typ := range e.api.Types {
		if !runtimeTypes[typ.Name] {
			filtered.Types = append(filtered.Types, typ)
		}
	}
	var buf bytes.Buffer
	if execErr := t.Execute(&buf, &filtered); execErr != nil {
		return fmt.Errorf("execute types.tmpl: %w", execErr)
	}
	src, err := format.Source(buf.Bytes())
	if err != nil {
		// Surface the unformatted output so debugging is possible.
		return fmt.Errorf("gofmt types.gen.go: %w\n--- unformatted ---\n%s", err, buf.String())
	}
	return os.WriteFile(filepath.Join(e.outDir, "types.gen.go"), src, 0o600)
}

// loadAPI reads and decodes the IR JSON.
func loadAPI(path string) (*spec.API, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var api spec.API
	if err := json.Unmarshal(data, &api); err != nil {
		return nil, err
	}
	return &api, nil
}

// funcsWithDiscs returns the shared FuncMap with the variant
// discriminator helpers bound to discs. types.tmpl uses
// variantDiscFor/variantHasDisc to emit per-variant MarshalJSON that
// hardcodes the wire discriminator value.
func funcsWithDiscs(plan *enumPlan, discs map[string]variantDiscriminator) template.FuncMap {
	fm := funcs(plan)
	fm["variantHasDisc"] = func(name string) bool {
		_, ok := discs[name]
		return ok
	}
	fm["variantDiscField"] = func(name string) string { return discs[name].JSONField }
	fm["variantDiscGoField"] = func(name string) string { return discs[name].GoField }
	fm["variantDiscValue"] = func(name string) string { return discs[name].Value }
	return fm
}

// funcs is the FuncMap shared across templates. plan is the resolved
// enum plan; pass nil only in unit tests that don't exercise enums.
func funcs(plan *enumPlan) template.FuncMap {
	return template.FuncMap{
		"goType": goType,
		"goField": func(parent string, f spec.Field) string {
			return goField(plan, parent, f)
		},
		"docComment":  docComment,
		"isOptional":  func(f spec.Field) bool { return !f.Required },
		"not":         func(b bool) bool { return !b },
		"title":       title,
		"isFileField": isFileField,
		"fileCheck":   fileCheck,
		"multipartFieldEntry": func(parent string, f spec.Field) string {
			return multipartFieldEntry(plan, parent, f)
		},
		"multipartFileEntry": multipartFileEntry,
		"returnGoType":       returnGoType,
		// enum helpers
		"enums": func() []enumDecl {
			if plan == nil {
				return nil
			}
			return plan.All()
		},
		"enumConstName": constName,
		// discriminator helpers for types.tmpl
		"hasDiscriminator": func(name string) bool { s, ok := knownDiscriminators[name]; return ok && len(s.Variants) > 0 },
		"isSealedUnionReturn": func(tr spec.TypeRef) bool {
			if tr.Kind != spec.KindNamed {
				return false
			}
			s, ok := knownDiscriminators[tr.Name]
			return ok && len(s.Variants) > 0
		},
		"isSealedUnionArrayReturn": func(tr spec.TypeRef) bool {
			if tr.Kind != spec.KindArray || tr.ElemType == nil || tr.ElemType.Kind != spec.KindNamed {
				return false
			}
			s, ok := knownDiscriminators[tr.ElemType.Name]
			return ok && len(s.Variants) > 0
		},
		"sealedUnionElemName": func(tr spec.TypeRef) string {
			if tr.Kind == spec.KindArray && tr.ElemType != nil {
				return tr.ElemType.Name
			}
			return ""
		},
		"isMaybeInaccessibleMessage": func(name string) bool { return name == "MaybeInaccessibleMessage" },
		"discriminatorField":         func(name string) string { return knownDiscriminators[name].Field },
		"discriminatorMap":           func(name string) map[string]string { return knownDiscriminators[name].Variants },
		// union-field helpers for per-struct UnmarshalJSON emission
		"unionFields":   unionFieldsOf,
		"isArrayUnion":  func(tr spec.TypeRef) bool { return hasUnionElem(tr) },
		"unionTypeName": func(tr spec.TypeRef) string { name, _ := unionTypeFor(tr); return name },
	}
}

// title upper-cases the first byte of s (ASCII only — all Telegram method names are ASCII).
func title(s string) string {
	if s == "" {
		return ""
	}
	r := s[0]
	if r >= 'a' && r <= 'z' {
		r = r - 'a' + 'A'
	}
	return string(r) + s[1:]
}

// isFileField reports whether the field carries an InputFile.
func isFileField(f spec.Field) bool {
	return mentionsInputFileTr(f.Type)
}

func mentionsInputFileTr(tr spec.TypeRef) bool {
	switch tr.Kind {
	case spec.KindNamed:
		return tr.Name == "InputFile"
	case spec.KindArray:
		if tr.ElemType != nil {
			return mentionsInputFileTr(*tr.ElemType)
		}
	case spec.KindOneOf:
		for _, v := range tr.Variants {
			if v == "InputFile" {
				return true
			}
		}
	}
	return false
}

// fileCheck returns the HasFile guard line for a file-carrying field.
// Both named InputFile and InputFile-or-String oneOf fields are now *InputFile,
// so no type assertion is needed in either case.
func fileCheck(f spec.Field) string {
	return fmt.Sprintf("\tif p.%s != nil && p.%s.IsLocalUpload() { return true }\n", f.Name, f.Name)
}

// multipartFileEntry returns the MultipartFiles append block for a file field.
// Both named InputFile and InputFile-or-String oneOf fields are now *InputFile,
// so the same code works for both cases.
func multipartFileEntry(f spec.Field) string {
	jsonName := f.JSONName
	return fmt.Sprintf(
		"\tif p.%s != nil && p.%s.IsLocalUpload() {\n\t\tname := p.%s.Filename\n\t\tif name == \"\" { name = %q }\n\t\tfiles = append(files, client.MultipartFile{FieldName: %q, Filename: name, Reader: p.%s.Reader})\n\t}\n",
		f.Name, f.Name, f.Name, jsonName, jsonName, f.Name)
}

// multipartFieldEntry generates the line that adds f to the multipart map.
// Required scalar fields go in unconditionally; optional ones go in only
// when non-zero/non-empty. Typed-string enum fields are cast to string
// before assignment because the multipart map is map[string]string.
func multipartFieldEntry(plan *enumPlan, parent string, f spec.Field) string {
	enumName := plan.FieldEnum(parent, f.Name)
	switch f.Type.Kind {
	case spec.KindPrimitive:
		switch f.Type.Name {
		case "int64":
			if f.Required {
				return fmt.Sprintf("\tout[%q] = strconv.FormatInt(p.%s, 10)\n", f.JSONName, f.Name)
			}
			return fmt.Sprintf("\tif p.%s != nil { out[%q] = strconv.FormatInt(*p.%s, 10) }\n", f.Name, f.JSONName, f.Name)
		case "string":
			if enumName != "" {
				if f.Required {
					return fmt.Sprintf("\tout[%q] = string(p.%s)\n", f.JSONName, f.Name)
				}
				return fmt.Sprintf("\tif p.%s != \"\" { out[%q] = string(p.%s) }\n", f.Name, f.JSONName, f.Name)
			}
			if f.Required {
				return fmt.Sprintf("\tout[%q] = p.%s\n", f.JSONName, f.Name)
			}
			return fmt.Sprintf("\tif p.%s != \"\" { out[%q] = p.%s }\n", f.Name, f.JSONName, f.Name)
		case "bool":
			if f.Required {
				return fmt.Sprintf("\tout[%q] = strconv.FormatBool(p.%s)\n", f.JSONName, f.Name)
			}
			return fmt.Sprintf("\tif p.%s != nil { out[%q] = strconv.FormatBool(*p.%s) }\n", f.Name, f.JSONName, f.Name)
		case "float64":
			if f.Required {
				return fmt.Sprintf("\tout[%q] = strconv.FormatFloat(p.%s, 'f', -1, 64)\n", f.JSONName, f.Name)
			}
			return fmt.Sprintf("\tif p.%s != nil { out[%q] = strconv.FormatFloat(*p.%s, 'f', -1, 64) }\n", f.Name, f.JSONName, f.Name)
		}
	case spec.KindOneOf:
		// Integer-or-String → ChatID: use .String() wire form.
		if matchesVariants(f.Type.Variants, "int64", "string") {
			if f.Required {
				return fmt.Sprintf("\tout[%q] = p.%s.String()\n", f.JSONName, f.Name)
			}
			return fmt.Sprintf("\tif !p.%s.IsZero() { out[%q] = p.%s.String() }\n", f.Name, f.JSONName, f.Name)
		}
		// InputFile-or-String → *InputFile: non-upload branch sends PathOrID.
		if matchesVariants(f.Type.Variants, "InputFile", "string") {
			return fmt.Sprintf("\tif p.%s != nil && !p.%s.IsLocalUpload() && p.%s.PathOrID != \"\" { out[%q] = p.%s.PathOrID }\n",
				f.Name, f.Name, f.Name, f.JSONName, f.Name)
		}
		// Sealed-interface unions — JSON-marshal.
		if f.Required {
			return fmt.Sprintf("\tif b, _ := json.Marshal(p.%s); len(b) > 0 && string(b) != \"null\" { out[%q] = string(b) }\n", f.Name, f.JSONName)
		}
		return fmt.Sprintf("\tif p.%s != nil { if b, _ := json.Marshal(p.%s); len(b) > 0 && string(b) != \"null\" { out[%q] = string(b) } }\n", f.Name, f.Name, f.JSONName)
	}
	// Named or array: fall back to JSON-marshal to JSON string.
	if f.Required {
		return fmt.Sprintf("\tif b, _ := json.Marshal(p.%s); len(b) > 0 { out[%q] = string(b) }\n", f.Name, f.JSONName)
	}
	return fmt.Sprintf("\tif p.%s != nil { if b, _ := json.Marshal(p.%s); len(b) > 0 { out[%q] = string(b) } }\n", f.Name, f.Name, f.JSONName)
}

func returnGoType(tr spec.TypeRef) string {
	switch tr.Kind {
	case spec.KindPrimitive:
		return tr.Name
	case spec.KindNamed:
		// Sealed-interface unions are returned by interface value, not pointer
		// (you can't take a pointer to an interface in any useful way; the
		// generated UnmarshalXxx returns the interface directly).
		if _, ok := knownDiscriminators[tr.Name]; ok {
			return tr.Name
		}
		// MessageOrBool is a hand-coded runtime wrapper — pointer return.
		return "*" + tr.Name
	case spec.KindArray:
		if tr.ElemType == nil {
			return "[]any"
		}
		return "[]" + returnGoElem(*tr.ElemType)
	case spec.KindOneOf:
		// Integer-or-String return (rare but possible).
		if matchesVariants(tr.Variants, "int64", "string") {
			return "ChatID"
		}
		return "any"
	}
	return "any"
}

func returnGoElem(tr spec.TypeRef) string {
	switch tr.Kind {
	case spec.KindPrimitive:
		return tr.Name
	case spec.KindNamed:
		return tr.Name
	case spec.KindArray:
		if tr.ElemType == nil {
			return "any"
		}
		return "[]" + returnGoElem(*tr.ElemType)
	}
	return "any"
}

// emitMethods renders methods.gen.go.
func (e *emitter) emitMethods() error {
	t, err := template.New("methods").Funcs(funcs(e.enums)).Parse(methodsTmpl)
	if err != nil {
		return fmt.Errorf("parse methods.tmpl: %w", err)
	}
	var buf bytes.Buffer
	if execErr := t.Execute(&buf, e.api); execErr != nil {
		return fmt.Errorf("execute methods.tmpl: %w", execErr)
	}
	src, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("gofmt methods.gen.go: %w\n--- unformatted ---\n%s", err, buf.String())
	}
	return os.WriteFile(filepath.Join(e.outDir, "methods.gen.go"), src, 0o600)
}

// emitEnums renders enums.gen.go.
func (e *emitter) emitEnums() error {
	t, err := template.New("enums").Funcs(funcs(e.enums)).Parse(enumsTmpl)
	if err != nil {
		return fmt.Errorf("parse enums.tmpl: %w", err)
	}
	var buf bytes.Buffer
	if execErr := t.Execute(&buf, e.api); execErr != nil {
		return fmt.Errorf("execute enums.tmpl: %w", execErr)
	}
	src, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("gofmt enums.gen.go: %w\n--- unformatted ---\n%s", err, buf.String())
	}
	return os.WriteFile(filepath.Join(e.outDir, "enums.gen.go"), src, 0o600)
}

// goType returns the Go type expression for a TypeRef.
// Optional fields use pointer types for primitives and named types,
// or rely on omitempty for slices and maps. parameter `optional` controls
// whether to wrap pointer-style.
func goType(tr spec.TypeRef, optional bool) string {
	switch tr.Kind {
	case spec.KindPrimitive:
		if optional && (tr.Name == "bool" || tr.Name == "int64" || tr.Name == "float64") {
			return "*" + tr.Name
		}
		return tr.Name
	case spec.KindNamed:
		// Named types are always pointer-optional when optional, except:
		// 1. Union (interface) types — they are naturally nil-able; pointer-to-interface is invalid.
		// 2. InputFile is always pointer-typed even when required: the
		//    multipart helpers (fileCheck, multipartFileEntry) call
		//    f.IsLocalUpload() and dereference Reader, both of which
		//    expect a pointer receiver.
		if knownInterfaceTypes[tr.Name] {
			// Interface type — never add *.
			return tr.Name
		}
		if optional || tr.Name == "InputFile" {
			return "*" + tr.Name
		}
		return tr.Name
	case spec.KindArray:
		if tr.ElemType == nil {
			return "[]any"
		}
		// Inside slices, the element shape is its own thing — never wrap
		// the element in a pointer just because the field is optional.
		return "[]" + goType(*tr.ElemType, false)
	case spec.KindOneOf:
		// Integer-or-String: typed ChatID wrapper.
		if matchesVariants(tr.Variants, "int64", "string") {
			if optional {
				return "*ChatID"
			}
			return "ChatID"
		}
		// InputFile-or-String: *InputFile runtime helper handles both.
		if matchesVariants(tr.Variants, "InputFile", "string") {
			return "*InputFile"
		}
		// All-named variants sealed interface: fall back to interface.
		return "any"
	}
	return "any"
}

// unionField pairs a struct field with the name of its union type.
type unionField struct {
	Field     spec.Field
	UnionName string // e.g. "ChatMember"
}

// unionFieldsOf returns the subset of t.Fields whose type is a known
// discriminated union (directly or as array element).
func unionFieldsOf(t spec.TypeDecl) []unionField {
	var out []unionField
	for _, f := range t.Fields {
		if u, ok := unionTypeFor(f.Type); ok {
			out = append(out, unionField{Field: f, UnionName: u})
		}
	}
	return out
}

// unionTypeFor inspects a TypeRef and reports whether it (or its array
// element) is a known discriminated union. Returns the union name and true.
func unionTypeFor(tr spec.TypeRef) (string, bool) {
	switch tr.Kind {
	case spec.KindNamed:
		if _, ok := knownDiscriminators[tr.Name]; ok {
			return tr.Name, true
		}
	case spec.KindArray:
		if tr.ElemType != nil {
			return unionTypeFor(*tr.ElemType)
		}
	case spec.KindOneOf:
		if u := unionNameByVariants(tr.Variants); u != "" {
			return u, true
		}
	}
	return "", false
}

// unionNameByVariants finds the parent union whose variant type names exactly
// match the given variant set (order-insensitive).
func unionNameByVariants(variants []string) string {
	for parentName, ds := range knownDiscriminators {
		wanted := make([]string, 0, len(ds.Variants))
		for _, vt := range ds.Variants {
			wanted = append(wanted, vt)
		}
		if matchesVariants(variants, wanted...) {
			return parentName
		}
	}
	return ""
}

// hasUnionElem reports whether tr is an array whose element type is a known union.
func hasUnionElem(tr spec.TypeRef) bool {
	if tr.Kind != spec.KindArray || tr.ElemType == nil {
		return false
	}
	_, ok := unionTypeFor(*tr.ElemType)
	return ok
}

// matchesVariants reports whether got equals want as a set (order-insensitive).
func matchesVariants(got []string, want ...string) bool {
	if len(got) != len(want) {
		return false
	}
	seen := make(map[string]int, len(got))
	for _, g := range got {
		seen[g]++
	}
	for _, w := range want {
		seen[w]--
	}
	for _, v := range seen {
		if v != 0 {
			return false
		}
	}
	return true
}

// goField returns the Go struct-field declaration for a Field.
// When the field carries scraper-detected enum values and the emitter
// has a planned enum name for (parent, field), the field's Go type is
// the enum identifier. Typed-string enums use the zero string ""
// behaviour for omitempty, so we do not pointer-wrap optional enum
// fields. Parent is "" for method parameters.
func goField(plan *enumPlan, parent string, f spec.Field) string {
	tag := fmt.Sprintf("`json:%q`", f.JSONName+omitempty(f))
	if name := plan.FieldEnum(parent, f.Name); name != "" {
		return fmt.Sprintf("%s %s %s", f.Name, name, tag)
	}
	// Pinned companion-enum retype: allowed_updates is an Array of String
	// in the upstream spec, but the Go API exposes a hand-curated
	// UpdateType (api/enums.go) since the values are not enumerated
	// inline by Telegram. Retype []string → []UpdateType wherever the
	// wire field is allowed_updates so callers can pass typed constants
	// (api.UpdateMessage, ...) without string casts. Wire format is
	// unchanged: UpdateType is a typed string, marshals identically.
	if f.JSONName == "allowed_updates" &&
		f.Type.Kind == spec.KindArray &&
		f.Type.ElemType != nil &&
		f.Type.ElemType.Kind == spec.KindPrimitive &&
		f.Type.ElemType.Name == "string" {
		return fmt.Sprintf("%s []UpdateType %s", f.Name, tag)
	}
	return fmt.Sprintf("%s %s %s", f.Name, goType(f.Type, !f.Required), tag)
}

func omitempty(f spec.Field) string {
	if f.Required {
		return ""
	}
	return ",omitempty"
}

// docComment converts a doc string into a Go-style block comment with
// a leading "// " on each line.
func docComment(s string) string {
	if s == "" {
		return ""
	}
	var buf bytes.Buffer
	for _, line := range splitLines(s) {
		buf.WriteString("// ")
		buf.WriteString(line)
		buf.WriteByte('\n')
	}
	return buf.String()
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

// hasVariants reports whether the variant list contains all of the named strings (order-insensitive).
func hasVariants(variants []string, names ...string) bool {
	return matchesVariants(variants, names...)
}

// buildUnionTypeSet returns the set of all type names that generate interface types
// (i.e., types with one_of). This includes knownDiscriminators and marker-interface
// unions not covered by the discriminator map.
func buildUnionTypeSet(api *spec.API) map[string]bool {
	s := make(map[string]bool, len(knownDiscriminators)+16)
	for name := range knownDiscriminators {
		s[name] = true
	}
	for _, t := range api.Types {
		if len(t.OneOf) > 0 {
			s[t.Name] = true
		}
	}
	return s
}

// makeSentinelValue returns a sentinelValue func that uses the given union type set.
// It returns a minimal valid Go expression for a spec.Field's type,
// used in generated test param literals. plan supplies typed-enum names
// so a method-param sentinel for a ParseMode field becomes a typed
// constant rather than a magic string.
func makeSentinelValue(unionTypes map[string]bool, plan *enumPlan) func(spec.Field) string {
	return func(f spec.Field) string {
		return sentinelForField(f, unionTypes, plan)
	}
}

func sentinelForField(f spec.Field, unionTypes map[string]bool, plan *enumPlan) string {
	if name := plan.FieldEnum("", f.Name); name != "" && len(f.EnumValues) > 0 {
		return constName(name, f.EnumValues[0])
	}
	tr := f.Type
	switch tr.Kind {
	case spec.KindPrimitive:
		switch tr.Name {
		case "int64":
			return "42"
		case "string":
			return `"test_value"`
		case "bool":
			return "true"
		case "float64":
			return "1.0"
		}
	case spec.KindNamed:
		switch tr.Name {
		case "ChatID":
			return "ChatIDFromInt(123)"
		case "InputFile":
			return `&InputFile{PathOrID: "file_id_test"}`
		}
		// Interface (union) types are nil-able.
		if unionTypes[tr.Name] {
			return "nil"
		}
		// Required named struct types are value types in the generated struct.
		if f.Required {
			return tr.Name + "{}"
		}
		return "&" + tr.Name + "{}"
	case spec.KindArray:
		return "nil"
	case spec.KindOneOf:
		if hasVariants(tr.Variants, "int64", "string") {
			return "ChatIDFromInt(123)"
		}
		if hasVariants(tr.Variants, "InputFile", "string") {
			return `&InputFile{PathOrID: "file_id_test"}`
		}
		// Sealed named-union interface: use nil (any).
		return "nil"
	}
	return "nil"
}

// successResp returns a backtick Go string literal containing a minimal
// {"ok":true,"result":...} JSON body for the method's return type.
func successResp(m spec.MethodDecl) string {
	body := successBody(m.Returns)
	return "`{\"ok\":true,\"result\":" + body + "}`"
}

func successBody(tr spec.TypeRef) string {
	switch tr.Kind {
	case spec.KindPrimitive:
		switch tr.Name {
		case "bool":
			return "true"
		case "int64", "float64":
			return "0"
		case "string":
			return `""`
		}
	case spec.KindNamed:
		if tr.Name == "MessageOrBool" {
			return "true"
		}
		// Sealed-interface unions need a discriminator field so UnmarshalXxx can dispatch.
		// Pick the lexicographically first variant value for determinism (map
		// iteration order in Go is randomized — using `range` directly produces
		// non-deterministic regen output).
		if disc, ok := knownDiscriminators[tr.Name]; ok && disc.Field != "" {
			values := make([]string, 0, len(disc.Variants))
			for v := range disc.Variants {
				values = append(values, v)
			}
			sort.Strings(values)
			if len(values) > 0 {
				return fmt.Sprintf(`{"%s":"%s"}`, disc.Field, values[0])
			}
		}
		// MaybeInaccessibleMessage uses date==0 → InaccessibleMessage variant.
		if tr.Name == "MaybeInaccessibleMessage" {
			return `{"date":0,"chat":{"id":1,"type":"private"},"message_id":1}`
		}
		return "{}"
	case spec.KindArray:
		return "[]"
	case spec.KindOneOf:
		return "null"
	}
	return "null"
}

// emitTests renders methods_gen_test.go.
func (e *emitter) emitTests() error {
	unionTypes := buildUnionTypeSet(e.api)

	// Add test-specific helpers to the shared func map.
	fm := funcs(e.enums)
	fm["sentinelValue"] = makeSentinelValue(unionTypes, e.enums)
	fm["successResp"] = successResp

	t, err := template.New("tests").Funcs(fm).Parse(testsTmpl)
	if err != nil {
		return fmt.Errorf("parse tests.tmpl: %w", err)
	}
	var buf bytes.Buffer
	if execErr := t.Execute(&buf, e.api); execErr != nil {
		return fmt.Errorf("execute tests.tmpl: %w", execErr)
	}
	src, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("gofmt methods_gen_test.go: %w\n--- unformatted ---\n%s", err, buf.String())
	}
	return os.WriteFile(filepath.Join(e.outDir, "methods_gen_test.go"), src, 0o600)
}
