package utils

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// IsConfigured returns true if the field was explicitly set by the user.
// Mimics SDK v2's GetOk returning true - checks that value is not null and not unknown.
func IsConfigured(field attr.Value) bool {
	return !field.IsNull() && !field.IsUnknown()
}

// SetStringFromAPI sets a types.String field from an API response pointer.
// Use for Optional+Computed fields that should always reflect the API value.
func SetStringFromAPI(field *types.String, apiValue *string) {
	if apiValue != nil {
		*field = types.StringValue(*apiValue)
	} else {
		*field = types.StringNull()
	}
}

// SetBoolFromAPI sets a types.Bool field from an API response pointer.
// Use for Optional+Computed fields that should always reflect the API value.
func SetBoolFromAPI(field *types.Bool, apiValue *bool, defaultValue bool) {
	if apiValue != nil {
		*field = types.BoolValue(*apiValue)
	} else {
		*field = types.BoolValue(defaultValue)
	}
}

// SetStringPreserveConfig sets a types.String field, preserving user config if set.
// Use for Optional-only fields where user's config takes precedence over API.
func SetStringPreserveConfig(field *types.String, apiValue *string) {
	if IsConfigured(*field) {
		return
	}
	SetStringFromAPI(field, apiValue)
}
