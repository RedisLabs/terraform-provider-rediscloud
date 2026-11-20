package iris

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type jsonShapeValidator struct{ shape map[string]any }

func (jsonShapeValidator) Description(context.Context) string {
	return "JSON matching the Iris API request schema"
}
func (v jsonShapeValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}
func (v jsonShapeValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value, err := decodeJSON(req.ConfigValue.ValueString())
	if err == nil {
		err = validateShape(value, v.shape, "value")
	}
	if err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid Iris JSON configuration", err.Error())
	}
}
func validateShape(value any, s map[string]any, location string) error {
	fail := func(detail string) error { return fmt.Errorf("%s %s", location, detail) }
	if value == nil {
		return fail("must not be null; omit optional properties instead")
	}
	switch s["type"] {
	case "object":
		m, ok := value.(map[string]any)
		if !ok {
			return fail("must be an object")
		}
		properties, _ := s["properties"].(map[string]any)
		for _, raw := range anySlice(s["required"]) {
			if _, ok := m[fmt.Sprint(raw)]; !ok {
				return fmt.Errorf("%s.%s is required", location, raw)
			}
		}
		for key, v := range m {
			child, known := properties[key].(map[string]any)
			if !known {
				child, _ = s["additionalProperties"].(map[string]any)
				if child == nil && len(properties) > 0 {
					return fmt.Errorf("%s contains an unsupported property", location)
				}
			}
			if err := validateShape(v, child, location+"."+key); err != nil {
				return err
			}
		}
	case "array":
		values, ok := value.([]any)
		if !ok {
			return fail("must be an array")
		}
		if min, ok := s["minItems"].(float64); ok && len(values) < int(min) {
			return fail(fmt.Sprintf("must have at least %d items", int(min)))
		}
		if max, ok := s["maxItems"].(float64); ok && len(values) > int(max) {
			return fail(fmt.Sprintf("must have at most %d items", int(max)))
		}
		child, _ := s["items"].(map[string]any)
		seen := map[string]bool{}
		for i, v := range values {
			if s["uniqueItems"] == true {
				b, _ := json.Marshal(v)
				if seen[string(b)] {
					return fail("must not contain duplicate items")
				}
				seen[string(b)] = true
			}
			if err := validateShape(v, child, fmt.Sprintf("%s[%d]", location, i)); err != nil {
				return err
			}
		}
	case "string":
		text, ok := value.(string)
		if !ok {
			return fail("must be a string")
		}
		if min, ok := s["minLength"].(float64); ok && utf8.RuneCountInString(text) < int(min) {
			return fail(fmt.Sprintf("must contain at least %d characters", int(min)))
		}
		if max, ok := s["maxLength"].(float64); ok && utf8.RuneCountInString(text) > int(max) {
			return fail(fmt.Sprintf("must contain at most %d characters", int(max)))
		}
		if pattern, ok := s["pattern"].(string); ok {
			pattern = strings.ReplaceAll(pattern, `\\`, `\`)
			pattern = strings.ReplaceAll(pattern, `(?!\.{1,2}$)`, "")
			pattern = unicodeEscape.ReplaceAllString(pattern, `\x{$1}`)
			re, err := regexp.Compile(pattern)
			if err != nil {
				return fail("uses an unsupported API regex")
			}
			if !re.MatchString(text) {
				return fail("does not match the API format")
			}
		}
	case "integer", "number":
		n, ok := value.(json.Number)
		if !ok {
			return fail("must be a number")
		}
		number, ok := new(big.Rat).SetString(string(n))
		if !ok {
			return fail("must be a finite number")
		}
		if s["type"] == "integer" && !number.IsInt() {
			return fail("must be an integer")
		}
		for _, constraint := range []string{"minimum", "maximum"} {
			if limit, ok := s[constraint].(float64); ok {
				bound := new(big.Rat).SetFloat64(limit)
				cmp := number.Cmp(bound)
				if constraint == "minimum" && cmp < 0 || constraint == "maximum" && cmp > 0 {
					return fail("is outside the API limits")
				}
			}
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fail("must be a boolean")
		}
	}
	if options := anySlice(s["enum"]); len(options) > 0 {
		found := false
		for _, option := range options {
			if fmt.Sprint(option) == fmt.Sprint(value) {
				found = true
			}
		}
		if !found {
			return fail("is not a supported API value")
		}
	}
	return nil
}

var unicodeEscape = regexp.MustCompile(`\\u([0-9a-fA-F]{4})`)

func anySlice(v any) []any { a, _ := v.([]any); return a }

type pathComponentValidator struct{ tenant bool }

func (pathComponentValidator) Description(context.Context) string {
	return "a nonempty path component without slashes, dot segments or control characters"
}
func (v pathComponentValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}
func (v pathComponentValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if err := validatePathComponent(req.ConfigValue.ValueString(), v.tenant); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid Iris path component", err.Error())
	}
}
func validatePathComponent(s string, tenant bool) error {
	if s == "" || s == "." || s == ".." || strings.ContainsAny(s, "/\\") || strings.TrimSpace(s) != s {
		return fmt.Errorf("path identifiers must be nonempty, have no surrounding whitespace, and contain no slash or dot segment")
	}
	for _, c := range s {
		if unicode.IsControl(c) || tenant && c == '|' {
			return fmt.Errorf("path identifier contains a disallowed character")
		}
	}
	return nil
}

type namespaceValidator struct{}

func (namespaceValidator) Description(context.Context) string {
	return "a namespace ID, or an empty string to clear the destination"
}
func (v namespaceValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}
func (namespaceValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() || req.ConfigValue.ValueString() == "" {
		return
	}
	if !namespacePattern.MatchString(req.ConfigValue.ValueString()) {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid namespace identifier", "Use ns_ followed by 32 lowercase hexadecimal characters, or an empty string.")
	}
}

var namespacePattern = regexp.MustCompile(`^ns_[0-9a-f]{32}$`)
