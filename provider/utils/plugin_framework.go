package utils

import "github.com/hashicorp/terraform-plugin-framework/attr"

// IsConfigured returns true if the field was explicitly set by the user.
// Mimics SDK v2's GetOk returning true - checks that value is not null and not unknown.
func IsConfigured(field attr.Value) bool {
	return !field.IsNull() && !field.IsUnknown()
}
