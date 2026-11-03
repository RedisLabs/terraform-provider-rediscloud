// Package deprecation builds the customer-facing warning used when a
// Redis Cloud attribute is deprecated. Every resource should use Message
// so terraform plan and terraform apply print the same sentence.
package deprecation

import (
	"fmt"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Message is the warning detail for a deprecated attribute.
// replacement is the customer-facing text that follows "Use ", including
// backticks when it names an attribute, for example "`dataset_size_in_gb`".
func Message(attribute, replacement string) string {
	return fmt.Sprintf("`%s` is deprecated and will be removed in the next major version of the provider. Use %s instead.", attribute, replacement)
}

// Warning is a Terraform warning diagnostic with Message as its detail.
func Warning(attribute, replacement string) diag.Diagnostic {
	return diag.Diagnostic{
		Severity: diag.Warning,
		Summary:  "Deprecated attribute",
		Detail:   Message(attribute, replacement),
	}
}

// WhenNonEmpty warns on plan and apply only when the practitioner set a
// non-empty string. Use this instead of schema.Schema.Deprecated when the
// attribute has an empty Default: a Deprecated field on a defaulted
// attribute can warn on every plan, including configurations that never
// set the attribute.
func WhenNonEmpty(attribute, replacement string) schema.SchemaValidateDiagFunc {
	return func(v interface{}, _ cty.Path) diag.Diagnostics {
		s, ok := v.(string)
		if !ok || s == "" {
			return nil
		}
		return diag.Diagnostics{Warning(attribute, replacement)}
	}
}
