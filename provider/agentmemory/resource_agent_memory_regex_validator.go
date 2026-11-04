package agentmemory

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.String = nonEmptyRegexMatchValidator{}

type nonEmptyRegexMatchValidator struct{}

func (v nonEmptyRegexMatchValidator) Description(_ context.Context) string {
	return "pattern must not match empty text"
}

func (v nonEmptyRegexMatchValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v nonEmptyRegexMatchValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	if value == "" {
		return
	}
	pattern, err := regexp.Compile(value)
	if err != nil {
		// RE2 supports syntax Go does not (e.g. \C); leave syntax validation to the API.
		return
	}
	if pattern.MatchString("") {
		resp.Diagnostics.AddAttributeError(req.Path, "Regex matches empty text", "Custom detector patterns must not match empty text. Require at least one character in every matching alternative.")
	}
}
