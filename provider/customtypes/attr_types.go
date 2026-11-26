package customtypes

import (
	"context"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/attr"
)

// Use AttrTypesOf to derive attribute types from tagged framework value fields,
// including promoted fields from embedded value structs.
func AttrTypesOf(model any) map[string]attr.Type {
	v := reflect.ValueOf(model)
	t := v.Type()
	fields := reflect.VisibleFields(t)
	m := make(map[string]attr.Type, len(fields))
	for _, field := range fields {
		tag := field.Tag.Get("tfsdk")
		if tag == "" || tag == "-" {
			continue
		}
		m[tag] = v.FieldByIndex(field.Index).Interface().(attr.Value).Type(context.Background())
	}
	return m
}
