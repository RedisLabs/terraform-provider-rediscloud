package customtypes

import (
	"context"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/attr"
)

// attrTypesOf builds an attr-type map from a flat, tfsdk-tagged model struct: each
// field's `tfsdk` tag becomes the key and the field's own attr.Type the value. A custom
// type can then derive its attribute types from its model instead of hand-maintaining a
// parallel map.
// Generic across custom types; flat models only (every tagged field must implement attr.Value).
func attrTypesOf(model any) map[string]attr.Type {
	v := reflect.ValueOf(model)
	t := v.Type()
	m := make(map[string]attr.Type, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("tfsdk")
		if tag == "" || tag == "-" {
			continue
		}
		m[tag] = v.Field(i).Interface().(attr.Value).Type(context.Background())
	}
	return m
}
