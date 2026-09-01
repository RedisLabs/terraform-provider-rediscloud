package customtypes

import (
	"context"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/attr"
)

// AttrTypesOf derives attribute types from a flat model whose tagged fields implement attr.Value.
func AttrTypesOf(model any) map[string]attr.Type {
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
