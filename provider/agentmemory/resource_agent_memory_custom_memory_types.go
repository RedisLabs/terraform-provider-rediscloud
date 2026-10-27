package agentmemory

import (
	"context"
	"fmt"

	agentmemoryapi "github.com/RedisLabs/rediscloud-go-api/service/agentmemory"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type customMemoryTypePlanEntry struct {
	index int
	model AgentMemoryCustomMemoryTypeModel
}

func validateCustomMemoryTypesPlan(ctx context.Context, plan, state AgentMemoryResourceModel, resp *resource.ModifyPlanResponse) {
	if plan.CustomMemoryTypes.IsUnknown() {
		// Unknown definitions may remove or redefine an existing type. Reserve replacement
		// in the initial plan rather than discovering an immutable edit during apply.
		if len(state.CustomMemoryTypes.Elements()) > 0 {
			resp.RequiresReplace.Append(path.Root("custom_memory_types"))
		}
		return
	}
	plannedTypes := customMemoryTypeModels(ctx, plan.CustomMemoryTypes, &resp.Diagnostics)
	existingTypes := customMemoryTypeModels(ctx, state.CustomMemoryTypes, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	planByName := make(map[string]customMemoryTypePlanEntry, len(plannedTypes))
	for i, memoryType := range plannedTypes {
		name, ok := knownString(memoryType.Name)
		if !ok {
			continue
		}
		if previous, exists := planByName[name]; exists {
			resp.Diagnostics.AddAttributeError(
				path.Root("custom_memory_types").AtListIndex(i).AtName("name"),
				"Duplicate Agent Memory custom memory type",
				fmt.Sprintf("custom_memory_types names must be unique. %q is already configured at custom_memory_types.%d.", name, previous.index),
			)
			continue
		}
		planByName[name] = customMemoryTypePlanEntry{index: i, model: memoryType}
	}

	if resp.Diagnostics.HasError() {
		return
	}

	for _, existing := range existingTypes {
		name, ok := knownString(existing.Name)
		if !ok {
			continue
		}
		planned, exists := planByName[name]
		if !exists {
			// An unknown name can resolve to a rename; replacement must be visible
			// in the initial plan, before Terraform resolves it during apply.
			resp.RequiresReplace.Append(path.Root("custom_memory_types"))
			continue
		}
		if customMemoryTypeDefinitionHasUnknown(ctx, planned.model, &resp.Diagnostics) || !customMemoryTypeDefinitionEqual(ctx, planned.model, existing, &resp.Diagnostics) {
			resp.RequiresReplace.Append(path.Root("custom_memory_types").AtListIndex(planned.index))
			continue
		}
		if existing.ExtractionStrategy != nil && planned.model.ExtractionStrategy == nil {
			resp.RequiresReplace.Append(path.Root("custom_memory_types").AtListIndex(planned.index).AtName("extraction_strategy"))
		}
	}
}

func customMemoryTypeUpdatesFromPlan(ctx context.Context, plan, state CustomMemoryTypeListValue, diags *diag.Diagnostics) ([]agentmemoryapi.CustomMemoryType, []agentmemoryapi.MemoryTypeStrategyUpdate) {
	plannedTypes := customMemoryTypeModels(ctx, plan, diags)
	existingTypes := customMemoryTypeModels(ctx, state, diags)
	stateByName := make(map[string]AgentMemoryCustomMemoryTypeModel, len(existingTypes))
	for _, memoryType := range existingTypes {
		name, ok := knownString(memoryType.Name)
		if ok {
			stateByName[name] = memoryType
		}
	}

	var addTypes []agentmemoryapi.CustomMemoryType
	var updateStrategies []agentmemoryapi.MemoryTypeStrategyUpdate
	for _, planned := range plannedTypes {
		name, ok := knownString(planned.Name)
		if !ok {
			continue
		}
		existing, exists := stateByName[name]
		if !exists {
			addTypes = append(addTypes, customMemoryTypeFromModel(ctx, planned, diags))
			continue
		}
		if customExtractionStrategyEqual(planned.ExtractionStrategy, existing.ExtractionStrategy) {
			continue
		}
		if planned.ExtractionStrategy == nil {
			continue
		}

		update := agentmemoryapi.MemoryTypeStrategyUpdate{TypeName: planned.Name.ValueStringPointer()}
		update.Prompt = stringPointerFromValue(planned.ExtractionStrategy.Prompt)
		if !planned.ExtractionStrategy.Enabled.IsNull() && !planned.ExtractionStrategy.Enabled.IsUnknown() {
			enabled := planned.ExtractionStrategy.Enabled.ValueBool()
			update.Enabled = &enabled
		}
		updateStrategies = append(updateStrategies, update)
	}

	return addTypes, updateStrategies
}

func customMemoryTypeFromModel(ctx context.Context, model AgentMemoryCustomMemoryTypeModel, diags *diag.Diagnostics) agentmemoryapi.CustomMemoryType {
	return agentmemoryapi.CustomMemoryType{
		Name:               model.Name.ValueStringPointer(),
		Description:        model.Description.ValueStringPointer(),
		Fields:             customFieldsFromModel(ctx, model.Fields, diags),
		ExtractionStrategy: customExtractionStrategyFromModel(model.ExtractionStrategy),
	}
}

func customMemoryTypeDefinitionEqual(ctx context.Context, left, right AgentMemoryCustomMemoryTypeModel, diags *diag.Diagnostics) bool {
	if left.Fields.IsUnknown() || right.Fields.IsUnknown() {
		return false
	}
	leftFields := customFieldModels(ctx, left.Fields, diags)
	rightFields := customFieldModels(ctx, right.Fields, diags)
	if diags.HasError() {
		return false
	}
	if !left.Description.Equal(right.Description) {
		return false
	}
	if len(leftFields) != len(rightFields) {
		return false
	}

	leftFieldsByName := make(map[string]AgentMemoryCustomFieldModel, len(leftFields))
	for _, field := range leftFields {
		name, ok := knownString(field.Name)
		if !ok {
			return false
		}
		if _, exists := leftFieldsByName[name]; exists {
			return false
		}
		leftFieldsByName[name] = field
	}
	for _, rightField := range rightFields {
		name, ok := knownString(rightField.Name)
		if !ok {
			return false
		}
		leftField, exists := leftFieldsByName[name]
		if !exists || !customMemoryFieldEqual(leftField, rightField) {
			return false
		}
	}
	return true
}

func customMemoryTypeDefinitionHasUnknown(ctx context.Context, memoryType AgentMemoryCustomMemoryTypeModel, diags *diag.Diagnostics) bool {
	if memoryType.Description.IsUnknown() || memoryType.Fields.IsUnknown() {
		return true
	}
	for _, field := range customFieldModels(ctx, memoryType.Fields, diags) {
		if field.Name.IsUnknown() || field.Description.IsUnknown() || field.Type.IsUnknown() {
			return true
		}
	}
	return false
}

func customMemoryFieldEqual(left, right AgentMemoryCustomFieldModel) bool {
	return left.Name.Equal(right.Name) &&
		left.Description.Equal(right.Description) &&
		left.Type.Equal(right.Type)
}

func customExtractionStrategyEqual(left, right *AgentMemoryCustomExtractionStrategy) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Prompt.Equal(right.Prompt) && left.Enabled.Equal(right.Enabled)
}

func knownString(value types.String) (string, bool) {
	if value.IsNull() || value.IsUnknown() {
		return "", false
	}
	return value.ValueString(), true
}
