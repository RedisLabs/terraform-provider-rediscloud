package iris

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func TestMemoryTypePlanDefersUnknownImmutableValues(t *testing.T) {
	ctx := context.Background()
	r := NewAgentMemoryTypeResource()
	fields, d := types.SetValueFrom(ctx, memoryFieldElementType, []MemoryFieldModel{{Name: types.StringValue("topic"), Type: types.StringValue("STRING"), Description: types.StringValue("Topic")}})
	require.False(t, d.HasError())
	previous := AgentMemoryTypeModel{Timeouts: nullTimeouts(), Name: types.StringValue("profile"), StoreID: types.StringValue("store-1"), Description: types.StringValue("Profile"), Fields: fields}
	for _, unknown := range []string{"description", "field", "nested field"} {
		t.Run(unknown, func(t *testing.T) {
			planned := previous
			switch unknown {
			case "description":
				planned.Description = types.StringUnknown()
			case "field":
				planned.Fields = types.SetUnknown(memoryFieldElementType)
			case "nested field":
				value, d := types.SetValueFrom(ctx, memoryFieldElementType, []MemoryFieldModel{{Name: types.StringValue("topic"), Type: types.StringUnknown(), Description: types.StringValue("Topic")}})
				require.False(t, d.HasError())
				planned.Fields = value
			}
			resp := resource.ModifyPlanResponse{}
			r.(resource.ResourceWithModifyPlan).ModifyPlan(ctx, resource.ModifyPlanRequest{State: modelState(t, r, &previous), Plan: tfsdk.Plan(modelState(t, r, &planned))}, &resp)
			require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
		})
	}
}
