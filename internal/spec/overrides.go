package spec

import (
	"errors"
	"fmt"
	"github.com/goccy/go-json"
	"os"
)

// Overrides is the schema of internal/spec/overrides.json. It lets engineers
// pin specific method returns or field types, and approve methods that
// genuinely return bool but whose doc phrasing the scraper doesn't recognise.
type Overrides struct {
	// MethodReturns maps "<methodName>" → desired return TypeRef.
	// Applied AFTER the scraper extracts a return type, overriding it.
	MethodReturns map[string]TypeRef `json:"method_returns,omitempty"`

	// FieldTypes maps "<TypeName>.<FieldName>" → desired field TypeRef.
	// Applied AFTER the scraper builds the IR, overriding the field type.
	FieldTypes map[string]TypeRef `json:"field_types,omitempty"`

	// ApprovedBoolMethods lists methods whose returns are genuinely bool.
	// The audit tool ignores these.
	ApprovedBoolMethods []string `json:"approved_bool_methods,omitempty"`
}

// LoadOverrides reads and parses overrides.json. Returns an empty Overrides
// (not an error) if the file does not exist.
func LoadOverrides(path string) (*Overrides, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Overrides{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var o Overrides
	if err := json.Unmarshal(data, &o); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &o, nil
}

// Apply patches an API in place using the overrides.
func (o *Overrides) Apply(api *API) {
	if o == nil {
		return
	}
	for i, m := range api.Methods {
		if rt, ok := o.MethodReturns[m.Name]; ok {
			api.Methods[i].Returns = rt
		}
	}
	for i, t := range api.Types {
		for j, f := range t.Fields {
			key := t.Name + "." + f.Name
			if ft, ok := o.FieldTypes[key]; ok {
				api.Types[i].Fields[j].Type = ft
			}
		}
	}
}

// IsBoolApproved reports whether methodName is on the approved bool list.
func (o *Overrides) IsBoolApproved(methodName string) bool {
	if o == nil {
		return false
	}
	for _, n := range o.ApprovedBoolMethods {
		if n == methodName {
			return true
		}
	}
	return false
}
