package iris

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

const (
	defaultOperationTimeout = 30 * time.Minute
	defaultReadTimeout      = 5 * time.Minute
)

// State must be fully known, even after a failed follow-up read. Use Terraform's
// value traversal so new computed fields and nested values need no field list.
func saveState(ctx context.Context, state *tfsdk.State, data any) diag.Diagnostics {
	diagnostics := state.Set(ctx, data)
	if diagnostics.HasError() {
		return diagnostics
	}
	known, err := tftypes.Transform(state.Raw, func(_ *tftypes.AttributePath, value tftypes.Value) (tftypes.Value, error) {
		if !value.IsKnown() {
			return tftypes.NewValue(value.Type(), nil), nil
		}
		return value, nil
	})
	if err != nil {
		diagnostics.AddError("Unable to save Iris state", err.Error())
	} else {
		state.Raw = known
	}
	return diagnostics
}

// readAfterWrite refreshes a model from the service after a successful write and
// saves it. State is saved even when the read fails, so the resource, and any
// create-only secret, stays tracked. A failed read is an error, which taints a
// resource created in the same apply, unless warnOnly is set for resources whose
// replacement must never be forced.
func readAfterWrite[R any](ctx context.Context, state *tfsdk.State, data any, action string, warnOnly bool, read func() (*R, error), apply func(*R) diag.Diagnostics) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	current, err := read()
	if current != nil {
		diagnostics.Append(apply(current)...)
	}
	switch {
	case err == nil:
	case warnOnly:
		diagnostics.AddWarning("Failed to read "+action+" Iris resource", "The resource remains managed using its write response; Terraform will retry the read on the next refresh. "+err.Error())
	default:
		diagnostics.AddError("Failed to read "+action+" Iris resource", err.Error())
	}
	diagnostics.Append(saveState(ctx, state, data)...)
	return diagnostics
}

func operationContext(ctx context.Context, timeout func(context.Context, time.Duration) (time.Duration, diag.Diagnostics), fallback time.Duration, diagnostics *diag.Diagnostics) (context.Context, context.CancelFunc) {
	if diagnostics.HasError() {
		return ctx, func() {}
	}
	duration, d := timeout(ctx, fallback)
	diagnostics.Append(d...)
	return context.WithTimeout(ctx, duration)
}

func Resources() []func() resource.Resource {
	return []func() resource.Resource{
		NewAgentMemoryResource,
		NewAgentMemoryTypeResource,
		NewIdentityAPIKeyResource,
	}
}
func configuredClient(req resource.ConfigureRequest, service string, diagnostics *diag.Diagnostics) *client.IrisServiceClient {
	if req.ProviderData == nil {
		return nil
	}
	c, ok := req.ProviderData.(*client.ApiClient)
	if !ok {
		diagnostics.AddError("Unexpected provider client", fmt.Sprintf("Expected *client.ApiClient, got %T", req.ProviderData))
		return nil
	}
	connector := c.Iris[service]
	if connector == nil {
		diagnostics.AddError("Missing Iris service configuration", "Configure the Iris "+service+" endpoint and credentials.")
	}
	return connector
}
func optionalString(value types.String) *string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	v := value.ValueString()
	return &v
}
func optionalInt64(value types.Int64) *int64 {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	v := value.ValueInt64()
	return &v
}
func changedString(value, previous types.String) *string {
	if value.Equal(previous) {
		return nil
	}
	return optionalString(value)
}
func changedInt64(value, previous types.Int64) *int64 {
	if value.Equal(previous) {
		return nil
	}
	return optionalInt64(value)
}
func importParts(id string, count int) ([]string, error) {
	parts := strings.Split(id, "/")
	if len(parts) != count {
		return nil, fmt.Errorf("expected %d slash-separated identifiers", count)
	}
	for _, part := range parts {
		if err := validatePathComponent(part); err != nil {
			return nil, err
		}
	}
	return parts, nil
}
func escaped(value types.String) string { return url.PathEscape(value.ValueString()) }
func verifyID(expected types.String, actual *string) error {
	if actual == nil || *actual == "" || *actual != expected.ValueString() {
		return fmt.Errorf("iris response did not match the managed identifier")
	}
	return nil
}

func optionalBool(v types.Bool) *bool {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	value := v.ValueBool()
	return &value
}
func changedBool(value, previous types.Bool) *bool {
	if value.Equal(previous) {
		return nil
	}
	return optionalBool(value)
}
