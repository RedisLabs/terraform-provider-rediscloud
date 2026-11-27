package iris

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type pathComponentValidator struct{}

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
	if err := validatePathComponent(req.ConfigValue.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid Iris path component", err.Error())
	}
}
func validatePathComponent(s string) error {
	if s == "" || s == "." || s == ".." || strings.ContainsAny(s, "/\\") || strings.TrimSpace(s) != s {
		return fmt.Errorf("path identifiers must be nonempty, have no surrounding whitespace, and contain no slash or dot segment")
	}
	for _, c := range s {
		if unicode.IsControl(c) {
			return fmt.Errorf("path identifier contains a disallowed character")
		}
	}
	return nil
}

type canonicalGrantNameValidator struct{}

func (canonicalGrantNameValidator) Description(context.Context) string {
	return "a lowercase grant name without surrounding whitespace"
}
func (v canonicalGrantNameValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}
func (canonicalGrantNameValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	if value != strings.ToLower(strings.TrimSpace(value)) {
		resp.Diagnostics.AddAttributeError(req.Path, "Non-canonical grant name", "Use lowercase without surrounding whitespace. Iris stores a canonical value; Terraform configuration must match it.")
	}
}

type tenantValidator struct{}

func (tenantValidator) Description(context.Context) string {
	return "a tenant without surrounding whitespace"
}
func (v tenantValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}
func (tenantValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	if value != strings.TrimSpace(value) {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid tenant", "Use a tenant without surrounding whitespace. Iris trims this value; Terraform configuration must match the stored tenant. Tenant case is preserved.")
	}
}
