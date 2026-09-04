package customtypes

import (
	"context"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/attr"
)

// AttrTypesOf returns attribute types for tagged framework value fields.
// AttrTypesOf works on models that embed value structs as well.
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
