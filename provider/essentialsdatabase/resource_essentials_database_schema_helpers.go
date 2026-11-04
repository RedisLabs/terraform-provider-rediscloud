package essentialsdatabase

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type cidrValidator struct{}

func (v cidrValidator) Description(_ context.Context) string {
	return "value must be a valid CIDR"
}

func (v cidrValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v cidrValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	if _, _, err := net.ParseCIDR(value); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid CIDR",
			fmt.Sprintf("expected %q to be a valid CIDR Value, got %v: %v", req.Path.String(), value, err))
	}
}

// lowerCaseTagsValidator keeps the SDKv2 pro.ValidateTagsfunc rule and message.
type lowerCaseTagsValidator struct{}

func (v lowerCaseTagsValidator) Description(_ context.Context) string {
	return "tag keys and values must be lower case"
}

func (v lowerCaseTagsValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v lowerCaseTagsValidator) ValidateMap(_ context.Context, req validator.MapRequest, resp *validator.MapResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	keys := make([]string, 0, len(req.ConfigValue.Elements()))
	for k := range req.ConfigValue.Elements() {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	invalid := make([]string, 0)
	for _, k := range keys {
		if k != strings.ToLower(k) {
			invalid = append(invalid, k)
		}
		value, ok := req.ConfigValue.Elements()[k].(types.String)
		if !ok || value.IsUnknown() || value.IsNull() {
			continue
		}
		if s := value.ValueString(); s != strings.ToLower(s) {
			invalid = append(invalid, s)
		}
	}
	if len(invalid) > 0 {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid Tags",
			fmt.Sprintf("tag keys and values must be lower case, invalid entries: %s", strings.Join(invalid, ", ")))
	}
}

// SDKv2 stored unset collections as empty values after a refresh and treated empty and null as equal.
// These modifiers plan null for an unset collection, except when the state already holds an empty one,
// so that state written by SDKv2 plans without changes.

func emptyListUnlessConfigured() planmodifier.List {
	return emptyListUnlessConfiguredModifier{}
}

type emptyListUnlessConfiguredModifier struct{}

func (m emptyListUnlessConfiguredModifier) Description(_ context.Context) string {
	return "Plans null when unset, keeping an empty prior value."
}

func (m emptyListUnlessConfiguredModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m emptyListUnlessConfiguredModifier) PlanModifyList(_ context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	if !req.ConfigValue.IsNull() {
		return
	}
	if !req.StateValue.IsNull() && !req.StateValue.IsUnknown() && len(req.StateValue.Elements()) == 0 {
		resp.PlanValue = req.StateValue
		return
	}
	resp.PlanValue = types.ListNull(req.PlanValue.ElementType(context.Background()))
}

func emptyMapUnlessConfigured() planmodifier.Map {
	return emptyMapUnlessConfiguredModifier{}
}

type emptyMapUnlessConfiguredModifier struct{}

func (m emptyMapUnlessConfiguredModifier) Description(_ context.Context) string {
	return "Plans null when unset, keeping an empty prior value."
}

func (m emptyMapUnlessConfiguredModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m emptyMapUnlessConfiguredModifier) PlanModifyMap(_ context.Context, req planmodifier.MapRequest, resp *planmodifier.MapResponse) {
	if !req.ConfigValue.IsNull() {
		return
	}
	if !req.StateValue.IsNull() && !req.StateValue.IsUnknown() && len(req.StateValue.Elements()) == 0 {
		resp.PlanValue = req.StateValue
		return
	}
	resp.PlanValue = types.MapNull(req.PlanValue.ElementType(context.Background()))
}
