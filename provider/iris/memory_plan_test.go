package iris

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"
)

func TestStoreGroupsRequireExplicitCreationFlags(t *testing.T) {
	ctx := context.Background()
	r := NewAgentMemoryResource().(*agentMemoryResource)
	builtIn, d := types.SetValueFrom(ctx, builtInDetectorElementType, []BuiltInDetectorModel{{ID: types.StringValue("email"), Enabled: types.BoolValue(true)}})
	require.False(t, d.HasError())
	custom, d := types.SetValueFrom(ctx, customDetectorElementType, []CustomDetectorModel{{Name: types.StringValue("private"), Enabled: types.BoolValue(true), Pattern: types.StringValue("secret")}})
	require.False(t, d.HasError())
	for _, tc := range []struct {
		name   string
		values map[string]attr.Value
	}{
		{"summary_threshold", map[string]attr.Value{"summarization_event_threshold": types.Int64Value(100)}},
		{"summary_trigger", map[string]attr.Value{"summarization_trigger_strategy": types.StringValue("EVENT_COUNT")}},
		{"summary_incomplete", map[string]attr.Value{"summarization_enabled": types.BoolValue(true)}},
		{"semantic_prompt", map[string]attr.Value{"semantic_exclusions_prompt": types.StringValue("private data")}},
		{"built_in_detectors", map[string]attr.Value{"built_in_detector": builtIn}},
		{"custom_detectors", map[string]attr.Value{"custom_detector": custom}},
		{"child_group", map[string]attr.Value{"custom_detectors_enabled": types.BoolValue(false)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config, plan := objects(t, r, tc.values)
			state := tfsdk.State{Schema: config.Schema, Raw: tftypes.NewValue(config.Schema.Type().TerraformType(ctx), nil)}
			resp := resource.ModifyPlanResponse{Plan: plan}
			r.ModifyPlan(ctx, resource.ModifyPlanRequest{Config: config, Plan: plan, State: state}, &resp)
			require.True(t, resp.Diagnostics.HasError(), "missing creation flags must fail during planning")
			for _, enabled := range []bool{true, false} {
				previous, _ := objects(t, r, map[string]attr.Value{
					"summarization_enabled": types.BoolValue(enabled), "summarization_trigger_strategy": types.StringValue("EVENT_COUNT"), "summarization_event_threshold": types.Int64Value(100),
					"exclusions_enabled": types.BoolValue(enabled), "semantic_exclusions_enabled": types.BoolValue(enabled), "built_in_detectors_enabled": types.BoolValue(enabled), "custom_detectors_enabled": types.BoolValue(enabled),
				})
				resp = resource.ModifyPlanResponse{Plan: plan}
				r.ModifyPlan(ctx, resource.ModifyPlanRequest{Config: config, Plan: plan, State: tfsdk.State(previous)}, &resp)
				require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
			}
		})
	}
	// Iris defaults the trigger strategy to EVENT_COUNT, so enabling summarization
	// with a threshold must not also require configuring the strategy.
	config, plan := objects(t, r, map[string]attr.Value{"summarization_enabled": types.BoolValue(true), "summarization_event_threshold": types.Int64Value(20)})
	resp := resource.ModifyPlanResponse{Plan: plan}
	r.ModifyPlan(ctx, resource.ModifyPlanRequest{Config: config, Plan: plan, State: tfsdk.State{Raw: tftypes.NewValue(config.Schema.Type().TerraformType(ctx), nil), Schema: config.Schema}}, &resp)
	require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
	// Explicitly configuring inactive groups is legitimate, as are unknown flags.
	for _, flag := range []types.Bool{types.BoolValue(false), types.BoolUnknown()} {
		config, plan := objects(t, r, map[string]attr.Value{"built_in_detector": builtIn, "built_in_detectors_enabled": flag, "exclusions_enabled": flag})
		resp := resource.ModifyPlanResponse{Plan: plan}
		r.ModifyPlan(ctx, resource.ModifyPlanRequest{Config: config, Plan: plan, State: tfsdk.State{Raw: tftypes.NewValue(config.Schema.Type().TerraformType(ctx), nil), Schema: config.Schema}}, &resp)
		require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
	}
}
