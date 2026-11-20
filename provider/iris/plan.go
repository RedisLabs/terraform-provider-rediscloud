package iris

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.ResourceWithModifyPlan = (*serviceResource)(nil)

func (r *serviceResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if r.definition.Name != "iris_agent_memory" || req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	var planned, previous types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("long_term_memory_json"), &planned)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("long_term_memory_json"), &previous)...)
	if resp.Diagnostics.HasError() || planned.IsNull() || planned.IsUnknown() {
		return
	}
	next, err := decodeJSON(planned.ValueString())
	if err != nil {
		return
	}
	old, _ := decodeJSON(previous.ValueString())
	if modelsRequireReplacement(old, next) {
		resp.RequiresReplace.Append(path.Root("long_term_memory_json"))
	}
}
func modelsRequireReplacement(previous, planned any) bool {
	old, _ := previous.(map[string]any)
	next, _ := planned.(map[string]any)
	for _, name := range []string{"embedding", "llm", "classification"} {
		requested, ok := next[name].(map[string]any)
		if !ok {
			continue
		}
		current, _ := old[name].(map[string]any)
		if current == nil {
			if name != "classification" {
				return true
			}
			continue
		}
		for _, key := range []string{"provider", "model"} {
			if key == "model" && name != "embedding" {
				continue
			}
			a, _ := current[key].(string)
			b, _ := requested[key].(string)
			if a != b {
				return true
			}
		}
	}
	return false
}
