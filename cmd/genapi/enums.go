package main

import (
	"sort"
	"strings"

	"github.com/lukaszraczylo/go-telegram/internal/spec"
)

// enumDecl is one generated enum: a Go type alias of string plus a set
// of named constants. Values keep doc order; constant identifiers are
// derived from values via constName.
type enumDecl struct {
	Name   string
	Values []string
}

// enumPlan is the deduplicated, name-resolved set of enums emitted from
// an API IR. Lookup returns the enum name for a given field reference;
// All returns the deterministic-ordered list of declarations to emit.
type enumPlan struct {
	// fieldKey -> enum name. The fieldKey is a string built by enumKey.
	byField map[string]string
	// enum name -> declaration.
	decls map[string]enumDecl
}

// enumKey identifies a single Field occurrence so the emitter can look
// up the enum name later. Parent is "" for method params (the method
// doesn't share a Go type with the field).
func enumKey(parent, fieldName string) string { return parent + "::" + fieldName }

// planEnums walks the IR, decides on enum names, deduplicates, and
// returns an enumPlan. All scraper-marked enum fields are covered.
func planEnums(api *spec.API) *enumPlan {
	type ref struct {
		parent    string
		fieldName string
		jsonName  string
		values    []string
		valueKey  string // canonical key for value-set dedup
	}

	// Unification pass: for each sealed-interface union, fold per-variant
	// single-value enum fields that share a discriminator name into ONE
	// unified enum at union level. Claimed (parent,fieldName) tuples are
	// excluded from the per-field grouping below.
	unifiedDecls, unifiedByField := planUnifiedUnionEnums(api)
	claimed := func(parent, fieldName string) bool {
		_, ok := unifiedByField[enumKey(parent, fieldName)]
		return ok
	}

	var refs []ref
	collect := func(parent string, fields []spec.Field) {
		for _, f := range fields {
			if len(f.EnumValues) == 0 {
				continue
			}
			if claimed(parent, f.Name) {
				continue
			}
			refs = append(refs, ref{
				parent:    parent,
				fieldName: f.Name,
				jsonName:  f.JSONName,
				values:    f.EnumValues,
				valueKey:  valueKey(f.EnumValues),
			})
		}
	}
	for _, t := range api.Types {
		collect(t.Name, t.Fields)
	}
	for _, m := range api.Methods {
		// Method params have no shared Go parent type, so we pass "" as
		// the parent. The default-name heuristic still produces the
		// right answer for ParseMode-style enums.
		collect("", m.Params)
	}

	// candidate name per ref (before collision resolution)
	candidate := make([]string, len(refs))
	for i, r := range refs {
		candidate[i] = defaultEnumName(r.parent, r.jsonName, r.fieldName)
	}

	// Group by valueKey to coalesce identical value-sets across fields.
	// Choose canonical name: prefer the most common candidate; tie-break
	// by shortest name; final tie-break alphabetical.
	type groupInfo struct {
		values []string
		name   string
		first  int
	}
	groups := map[string]*groupInfo{}
	for i, r := range refs {
		g, ok := groups[r.valueKey]
		if !ok {
			groups[r.valueKey] = &groupInfo{values: r.values, first: i}
			g = groups[r.valueKey]
		}
		_ = g
	}
	// Rank candidate names per group.
	for vk := range groups {
		counts := map[string]int{}
		hasParent := map[string]bool{}
		var names []string
		for i, r := range refs {
			if r.valueKey != vk {
				continue
			}
			n := candidate[i]
			if _, ok := counts[n]; !ok {
				names = append(names, n)
			}
			counts[n]++
			if r.parent != "" {
				hasParent[n] = true
			}
		}
		// Pick the canonical name for this group:
		//   1. highest occurrence count wins;
		//   2. names that originated from a parent type win over plain
		//      method-param candidates (avoids "Format"-style
		//      monosyllables);
		//   3. shortest name wins;
		//   4. alphabetical for full determinism.
		sort.SliceStable(names, func(a, b int) bool {
			if counts[names[a]] != counts[names[b]] {
				return counts[names[a]] > counts[names[b]]
			}
			if hasParent[names[a]] != hasParent[names[b]] {
				return hasParent[names[a]]
			}
			if len(names[a]) != len(names[b]) {
				return len(names[a]) < len(names[b])
			}
			return names[a] < names[b]
		})
		groups[vk].name = names[0]
	}

	// Collision pass: two groups must not share the same enum name.
	// When that happens, suffix the loser(s) with their parent type
	// name so the result is unique. Iterate in deterministic order
	// (groups sorted by valueKey).
	used := map[string]string{} // name -> valueKey owner
	var keys []string
	for vk := range groups {
		keys = append(keys, vk)
	}
	sort.Strings(keys)
	for _, vk := range keys {
		g := groups[vk]
		if _, taken := used[g.name]; !taken {
			used[g.name] = vk
			continue
		}
		// Find a unique name by prepending a parent prefix from one of
		// the contributing refs (the lowest-index ref in this group).
		for i, r := range refs {
			if r.valueKey != vk {
				continue
			}
			if r.parent == "" {
				continue
			}
			cand := r.parent + goNamePart(r.jsonName)
			if _, taken := used[cand]; !taken {
				g.name = cand
				used[cand] = vk
				goto next
			}
			_ = i
		}
		// Fallback: append a numeric disambiguator. Should not happen
		// in practice for the Telegram docs but keeps the algorithm
		// total.
		for n := 2; ; n++ {
			cand := groups[vk].name + itoa(n)
			if _, taken := used[cand]; !taken {
				g.name = cand
				used[cand] = vk
				break
			}
		}
	next:
	}

	// Build the plan.
	plan := &enumPlan{
		byField: map[string]string{},
		decls:   map[string]enumDecl{},
	}
	for i, r := range refs {
		name := groups[r.valueKey].name
		plan.byField[enumKey(r.parent, r.fieldName)] = name
		_ = i
	}
	for vk, g := range groups {
		plan.decls[g.name] = enumDecl{Name: g.name, Values: g.values}
		_ = vk
	}
	// Merge unified union enums (already named with stutter handling and
	// keyed per-variant in unifiedByField).
	for k, name := range unifiedByField {
		plan.byField[k] = name
	}
	for name, d := range unifiedDecls {
		plan.decls[name] = d
	}
	return plan
}

// planUnifiedUnionEnums detects sealed-interface unions whose variants
// share a single discriminator field with one enum value each, and emits
// ONE unified enum per union covering all variant values. Returns the
// declarations to emit and the per-(variant,fieldName) map to point each
// variant's field at the unified enum.
//
// A union qualifies when EVERY variant in t.OneOf:
//  1. defines a field with the same Go-name (e.g. "Status", "Type", "Source");
//  2. that field is a required string with len(EnumValues)==1.
//
// The picked Go-name is the first one tried in this priority order:
//   - knownDiscriminators[union].Field's Go-name (resolved via JSONName match);
//   - "Type", "Status", "Source" (the three discriminators Telegram uses).
//
// First match wins; if none qualify, the union is skipped (variants keep
// their existing per-field treatment, which still single-emits via the
// regular grouping pass).
func planUnifiedUnionEnums(api *spec.API) (map[string]enumDecl, map[string]string) {
	decls := map[string]enumDecl{}
	byField := map[string]string{}

	typeByName := make(map[string]*spec.TypeDecl, len(api.Types))
	for i := range api.Types {
		typeByName[api.Types[i].Name] = &api.Types[i]
	}

	// Iterate unions in deterministic (declaration) order.
	for ui := range api.Types {
		u := &api.Types[ui]
		if len(u.OneOf) == 0 {
			continue
		}

		// Resolve the variants. Skip unions where any variant is missing
		// (defensive — shouldn't happen in a well-formed IR).
		variants := make([]*spec.TypeDecl, 0, len(u.OneOf))
		for _, vName := range u.OneOf {
			v, ok := typeByName[vName]
			if !ok {
				variants = nil
				break
			}
			variants = append(variants, v)
		}
		if len(variants) == 0 {
			continue
		}

		// Build the candidate Go-name list. Priority order:
		//  1. discriminator GoField from knownDiscriminators (resolved via JSONName);
		//  2. "Type", "Status", "Source".
		var candidateNames []string
		seen := map[string]bool{}
		add := func(name string) {
			if name == "" || seen[name] {
				return
			}
			seen[name] = true
			candidateNames = append(candidateNames, name)
		}
		if ds, ok := knownDiscriminators[u.Name]; ok && ds.Field != "" {
			// Resolve Go-name from the first variant whose field matches the JSON name.
			for _, v := range variants {
				for _, f := range v.Fields {
					if f.JSONName == ds.Field {
						add(f.Name)
						break
					}
				}
			}
		}
		for _, n := range []string{"Type", "Status", "Source"} {
			add(n)
		}

		// Find the first candidate Go-name where every variant has a
		// matching single-value string-enum field.
		var (
			pickedName string
			pickedDocs map[string]spec.Field // variant name -> field
		)
		for _, name := range candidateNames {
			matches := map[string]spec.Field{}
			ok := true
			for _, v := range variants {
				var hit *spec.Field
				for fi := range v.Fields {
					if v.Fields[fi].Name == name {
						hit = &v.Fields[fi]
						break
					}
				}
				if hit == nil ||
					hit.Type.Kind != spec.KindPrimitive ||
					hit.Type.Name != "string" ||
					len(hit.EnumValues) != 1 {
					ok = false
					break
				}
				matches[v.Name] = *hit
			}
			if ok {
				pickedName = name
				pickedDocs = matches
				break
			}
		}
		if pickedName == "" {
			continue
		}

		// Build the unified enum name with stutter handling.
		enumName := unifiedEnumName(u.Name, pickedName)

		// Collect values across variants in deterministic order, deduping.
		valueOrder := make([]string, 0, len(variants))
		valueSeen := map[string]bool{}
		for _, v := range u.OneOf {
			f := pickedDocs[v]
			val := f.EnumValues[0]
			if valueSeen[val] {
				continue
			}
			valueSeen[val] = true
			valueOrder = append(valueOrder, val)
		}

		decls[enumName] = enumDecl{Name: enumName, Values: valueOrder}
		for _, v := range variants {
			byField[enumKey(v.Name, pickedName)] = enumName
		}
	}

	return decls, byField
}

// unifiedEnumName builds the union-level enum name. Falls back to a
// "Kind" suffix when the naive concatenation reads as a stutter:
//
//   - union name ends in the field name verbatim (e.g. BackgroundType+Type);
//   - union name ends in any "concept noun" — Type/Status/Source/State —
//     so appending another such noun would duplicate the suffix
//     (e.g. ChatBoostSource+Source, RevenueWithdrawalState+Type).
//
// Otherwise the natural concatenation wins (ChatMember+Status →
// ChatMemberStatus, MessageOrigin+Type → MessageOriginType).
func unifiedEnumName(unionName, fieldName string) string {
	for _, suf := range []string{"Type", "Status", "Source", "State"} {
		if strings.HasSuffix(unionName, suf) {
			return unionName + "Kind"
		}
	}
	return unionName + fieldName
}

// All returns the enum declarations sorted by name for deterministic emit.
func (p *enumPlan) All() []enumDecl {
	out := make([]enumDecl, 0, len(p.decls))
	for _, d := range p.decls {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// FieldEnum returns the enum name for a field on a given parent type
// (use parent="" for method parameters), or "" if the field is not an
// enum.
func (p *enumPlan) FieldEnum(parent, fieldName string) string {
	if p == nil {
		return ""
	}
	return p.byField[enumKey(parent, fieldName)]
}

// defaultEnumName picks an initial Go enum name for a field. parse_mode
// fields collapse to the canonical "ParseMode"; otherwise the name is
// parent + PascalCase(jsonName).
func defaultEnumName(parent, jsonName, fieldName string) string {
	if strings.HasSuffix(jsonName, "parse_mode") {
		return "ParseMode"
	}
	return parent + goNamePart(jsonName)
}

// constName builds a Go constant identifier "<EnumName><PascalValue>"
// from a wire value. Slashes (mime types) become "Of" so
// "image/jpeg" → "ImageOfJpeg".
func constName(enumName, value string) string {
	return enumName + valuePascal(value)
}

func valuePascal(v string) string {
	// "image/jpeg" → "ImageOfJpeg"
	parts := strings.Split(v, "/")
	for i, p := range parts {
		parts[i] = goNamePart(p)
	}
	return strings.Join(parts, "Of")
}

// goNamePart converts a snake_case (or already-PascalCase) token to
// PascalCase, mirroring scrape.goName behaviour without the acronym
// special-cases (which apply to wire identifiers, not enum values).
func goNamePart(s string) string {
	if s == "" {
		return ""
	}
	parts := strings.Split(s, "_")
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		// Acronyms used in Telegram wire names. Keeping in sync with
		// scrape/table.go avoids divergent capitalisation between
		// fieldName and constName.
		switch p {
		case "id":
			b.WriteString("ID")
			continue
		case "url":
			b.WriteString("URL")
			continue
		case "ip":
			b.WriteString("IP")
			continue
		case "https":
			b.WriteString("HTTPS")
			continue
		case "json":
			b.WriteString("JSON")
			continue
		case "html":
			b.WriteString("HTML")
			continue
		}
		if c := p[0]; c >= 'a' && c <= 'z' {
			b.WriteByte(c - 'a' + 'A')
			b.WriteString(p[1:])
		} else {
			b.WriteString(p)
		}
	}
	return b.String()
}

func valueKey(values []string) string {
	cp := make([]string, len(values))
	copy(cp, values)
	sort.Strings(cp)
	return strings.Join(cp, "\x00")
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
