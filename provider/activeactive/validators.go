package activeactive

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// timeValidator validates that a string is a valid time in HH:MM format.
type timeValidator struct{}

func (v timeValidator) Description(_ context.Context) string {
	return "value must be a valid time in HH:MM format (24-hour)"
}

func (v timeValidator) MarkdownDescription(_ context.Context) string {
	return "value must be a valid time in HH:MM format (24-hour)"
}

func (v timeValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()
	if value == "" {
		return
	}

	if _, err := time.Parse("15:04", value); err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid time format",
			fmt.Sprintf("Value must be a valid time in HH:MM format (24-hour), got: %q", value),
		)
	}
}

// TimeValidator returns a validator that checks for valid HH:MM time format.
func TimeValidator() validator.String {
	return timeValidator{}
}

// respVersionValidator validates that the RESP version is either "resp2" or "resp3".
type respVersionValidator struct{}

func (v respVersionValidator) Description(_ context.Context) string {
	return "value must be 'resp2' or 'resp3'"
}

func (v respVersionValidator) MarkdownDescription(_ context.Context) string {
	return "value must be `resp2` or `resp3`"
}

func (v respVersionValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()
	if value == "" {
		return
	}

	matched, _ := regexp.MatchString("^(resp2|resp3)$", value)
	if !matched {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid RESP version",
			fmt.Sprintf("Value must be 'resp2' or 'resp3', got: %q", value),
		)
	}
}

// RespVersionValidator returns a validator that checks for valid RESP version.
func RespVersionValidator() validator.String {
	return respVersionValidator{}
}

// nameLengthValidator validates string length is within bounds.
type nameLengthValidator struct {
	min int
	max int
}

func (v nameLengthValidator) Description(_ context.Context) string {
	return fmt.Sprintf("value must be between %d and %d characters", v.min, v.max)
}

func (v nameLengthValidator) MarkdownDescription(_ context.Context) string {
	return fmt.Sprintf("value must be between %d and %d characters", v.min, v.max)
}

func (v nameLengthValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()
	length := len(value)

	if length < v.min || length > v.max {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid string length",
			fmt.Sprintf("String length must be between %d and %d characters, got: %d", v.min, v.max, length),
		)
	}
}

// StringLengthBetween returns a validator that checks string length is within bounds.
func StringLengthBetween(min, max int) validator.String {
	return nameLengthValidator{min: min, max: max}
}

// portRangeValidator validates that an integer is within the valid port range.
type portRangeValidator struct {
	min int64
	max int64
}

func (v portRangeValidator) Description(_ context.Context) string {
	return fmt.Sprintf("value must be between %d and %d", v.min, v.max)
}

func (v portRangeValidator) MarkdownDescription(_ context.Context) string {
	return fmt.Sprintf("value must be between %d and %d", v.min, v.max)
}

func (v portRangeValidator) ValidateInt64(ctx context.Context, req validator.Int64Request, resp *validator.Int64Response) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueInt64()

	if value < v.min || value > v.max {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid port number",
			fmt.Sprintf("Port must be between %d and %d, got: %d", v.min, v.max, value),
		)
	}
}

// PortRangeValidator returns a validator that checks port is within valid range.
func PortRangeValidator() validator.Int64 {
	return portRangeValidator{min: 10000, max: 19999}
}
