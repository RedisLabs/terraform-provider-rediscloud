package iris

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func decodeJSON(s string) (any, error) {
	d := json.NewDecoder(bytes.NewBufferString(s))
	d.UseNumber()
	var value any
	if err := d.Decode(&value); err != nil {
		return nil, fmt.Errorf("must contain valid JSON")
	}
	if d.Decode(new(any)) != io.EOF {
		return nil, fmt.Errorf("must contain exactly one JSON value")
	}
	return value, nil
}
func equalJSON(a, b string) bool {
	x, e := decodeJSON(a)
	if e != nil {
		return false
	}
	y, e := decodeJSON(b)
	return e == nil && reflect.DeepEqual(x, y)
}
func toWire(v attr.Value, isJSON bool) (any, error) {
	if v.IsNull() {
		return nil, nil
	}
	if v.IsUnknown() {
		return nil, fmt.Errorf("contains an unknown value")
	}
	switch v := v.(type) {
	case types.String:
		if isJSON {
			return decodeJSON(v.ValueString())
		}
		return v.ValueString(), nil
	case types.Int64:
		return v.ValueInt64(), nil
	case types.Float64:
		return v.ValueFloat64(), nil
	case types.Bool:
		return v.ValueBool(), nil
	case types.List:
		out := make([]any, 0, len(v.Elements()))
		for _, e := range v.Elements() {
			x, err := toWire(e, false)
			if err != nil {
				return nil, err
			}
			out = append(out, x)
		}
		return out, nil
	case types.Set:
		out := make([]any, 0, len(v.Elements()))
		for _, e := range v.Elements() {
			x, err := toWire(e, false)
			if err != nil {
				return nil, err
			}
			out = append(out, x)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported generated value type %T", v)
	}
}
func fromWire(ctx context.Context, raw any, typ attr.Type, isJSON bool) (attr.Value, error) {
	if raw == nil {
		return nullValue(ctx, typ), nil
	}
	if isJSON {
		b, err := json.Marshal(raw)
		if err != nil {
			return nil, fmt.Errorf("cannot encode service value")
		}
		return types.StringValue(string(b)), nil
	}
	switch {
	case typ.Equal(types.StringType):
		if value, ok := raw.(string); ok {
			return types.StringValue(value), nil
		}
	case typ.Equal(types.BoolType):
		if value, ok := raw.(bool); ok {
			return types.BoolValue(value), nil
		}
	case typ.Equal(types.Int64Type):
		if value, ok := raw.(json.Number); ok {
			number, err := value.Int64()
			if err == nil {
				return types.Int64Value(number), nil
			}
		}
	case typ.Equal(types.Float64Type):
		if value, ok := raw.(json.Number); ok {
			number, err := value.Float64()
			if err == nil {
				return types.Float64Value(number), nil
			}
		}
	default:
		values, ok := raw.([]any)
		if !ok {
			break
		}
		collection, ok := typ.(attr.TypeWithElementType)
		if !ok {
			break
		}
		elements := make([]attr.Value, 0, len(values))
		for _, rawElement := range values {
			element, err := fromWire(ctx, rawElement, collection.ElementType(), false)
			if err != nil {
				return nil, err
			}
			elements = append(elements, element)
		}
		switch typ.(type) {
		case basetypes.ListType:
			value, diags := types.ListValue(collection.ElementType(), elements)
			if !diags.HasError() {
				return value, nil
			}
		case basetypes.SetType:
			value, diags := types.SetValue(collection.ElementType(), elements)
			if !diags.HasError() {
				return value, nil
			}
		}
	}
	return nil, fmt.Errorf("service value has the wrong type")
}

func nullValue(ctx context.Context, typ attr.Type) attr.Value {
	v, err := typ.ValueFromTerraform(ctx, tftypes.NewValue(typ.TerraformType(ctx), nil))
	if err != nil {
		panic(err)
	}
	return v
}
func (r *serviceResource) save(ctx context.Context, values map[string]attr.Value, typ types.ObjectType, state *tfsdk.State, diags *diag.Diagnostics) {
	for name, t := range typ.AttrTypes {
		v := values[name]
		if v == nil || v.IsUnknown() {
			values[name] = nullValue(ctx, t)
		}
	}
	object, d := types.ObjectValue(typ.AttrTypes, values)
	diags.Append(d...)
	if !d.HasError() {
		diags.Append(state.Set(ctx, object)...)
	}
}

// Keep the caller's JSON shape while reading current values. Response-only defaults
// are omitted; request-only fields and secrets that cannot be read are retained.
func projectJSON(template, current any, readShape map[string]any) any {
	if requested, ok := template.(map[string]any); ok {
		actual, _ := current.(map[string]any)
		if actual == nil {
			return current
		}
		out := map[string]any{}
		properties, _ := readShape["properties"].(map[string]any)
		for key, value := range requested {
			shape, _ := properties[key].(map[string]any)
			if response, ok := actual[key]; ok {
				out[key] = projectJSON(value, response, shape)
			} else if shape == nil || shape["format"] == "password" {
				out[key] = value
			} else {
				out[key] = nil
			}
		}
		return out
	}
	if requested, ok := template.([]any); ok {
		actual, ok := current.([]any)
		if !ok {
			return current
		}
		if len(requested) != len(actual) {
			return current
		}
		out := make([]any, len(actual))
		shape, _ := readShape["items"].(map[string]any)
		used := map[int]bool{}
		for i, value := range requested {
			index := i
			if key := objectIdentity(value); key != "" {
				for j, candidate := range actual {
					if !used[j] && objectIdentity(candidate) == key {
						index = j
						break
					}
				}
			}
			used[index] = true
			out[i] = projectJSON(value, actual[index], shape)
		}
		return out
	}
	return current
}
func objectIdentity(v any) string {
	m, _ := v.(map[string]any)
	if m == nil {
		return ""
	}
	for _, key := range []string{"id", "name", "storeId", "extractorId"} {
		if s, ok := m[key].(string); ok {
			return key + ":" + s
		}
	}
	if m["product"] != nil {
		return fmt.Sprint(m["product"], "/", m["resourceType"], "/", m["resourceId"])
	}
	return ""
}
