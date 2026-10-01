package agentmemory

import (
	"fmt"

	agentmemoryapi "github.com/RedisLabs/rediscloud-go-api/service/agentmemory"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const unsupportedCustomMemoryTypeChangeMessage = "The Agent Memory API can add new custom memory types and update extraction strategy prompt/enabled for existing custom memory types, but it cannot remove or redefine an existing custom memory type. Terraform blocks this change instead of replacing the Agent Memory service automatically."

type customMemoryTypePlanEntry struct {
	index int
	model AgentMemoryCustomMemoryTypeModel
}

func validateCustomMemoryTypesPlan(plan, state AgentMemoryResourceModel, resp *resource.ModifyPlanResponse) {
	planByName := make(map[string]customMemoryTypePlanEntry, len(plan.CustomMemoryTypes))
	for i, memoryType := range plan.CustomMemoryTypes {
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

	for _, existing := range state.CustomMemoryTypes {
		name, ok := knownString(existing.Name)
		if !ok {
			continue
		}
		planned, exists := planByName[name]
		if !exists {
			resp.Diagnostics.AddAttributeError(
				path.Root("custom_memory_types"),
				"Unsupported Agent Memory custom memory type removal",
				fmt.Sprintf("custom_memory_types contains an existing type named %q that is not present in the new configuration. %s", name, unsupportedCustomMemoryTypeChangeMessage),
			)
			continue
		}
		if customMemoryTypeDefinitionHasUnknown(planned.model) {
			continue
		}
		if !customMemoryTypeDefinitionEqual(planned.model, existing) {
			resp.Diagnostics.AddAttributeError(
				path.Root("custom_memory_types").AtListIndex(planned.index),
				"Unsupported Agent Memory custom memory type redefinition",
				fmt.Sprintf("custom_memory_types.%d changes the description or fields for existing type %q. %s", planned.index, name, unsupportedCustomMemoryTypeChangeMessage),
			)
			continue
		}
		if existing.ExtractionStrategy != nil && planned.model.ExtractionStrategy == nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("custom_memory_types").AtListIndex(planned.index).AtName("extraction_strategy"),
				"Unsupported Agent Memory custom memory type strategy removal",
				fmt.Sprintf("custom_memory_types.%d removes the extraction_strategy for existing type %q. %s", planned.index, name, unsupportedCustomMemoryTypeChangeMessage),
			)
		}
	}
}

func customMemoryTypeUpdatesFromPlan(plan, state []AgentMemoryCustomMemoryTypeModel) ([]agentmemoryapi.CustomMemoryType, []agentmemoryapi.MemoryTypeStrategyUpdate) {
	stateByName := make(map[string]AgentMemoryCustomMemoryTypeModel, len(state))
	for _, memoryType := range state {
		name, ok := knownString(memoryType.Name)
		if ok {
			stateByName[name] = memoryType
		}
	}

	var addTypes []agentmemoryapi.CustomMemoryType
	var updateStrategies []agentmemoryapi.MemoryTypeStrategyUpdate
	for _, planned := range plan {
		name, ok := knownString(planned.Name)
		if !ok {
			continue
		}
		existing, exists := stateByName[name]
		if !exists {
			addTypes = append(addTypes, customMemoryTypeFromModel(planned))
			continue
		}
		if customExtractionStrategyEqual(planned.ExtractionStrategy, existing.ExtractionStrategy) {
			continue
		}
		if planned.ExtractionStrategy == nil {
			continue
		}

		update := agentmemoryapi.MemoryTypeStrategyUpdate{TypeName: name}
		if !planned.ExtractionStrategy.Prompt.IsNull() && !planned.ExtractionStrategy.Prompt.IsUnknown() {
			update.Prompt = planned.ExtractionStrategy.Prompt.ValueString()
		}
		if !planned.ExtractionStrategy.Enabled.IsNull() && !planned.ExtractionStrategy.Enabled.IsUnknown() {
			enabled := planned.ExtractionStrategy.Enabled.ValueBool()
			update.Enabled = &enabled
		}
		updateStrategies = append(updateStrategies, update)
	}

	return addTypes, updateStrategies
}

func customMemoryTypeFromModel(model AgentMemoryCustomMemoryTypeModel) agentmemoryapi.CustomMemoryType {
	return agentmemoryapi.CustomMemoryType{
		Name:               stringFromValue(model.Name),
		Description:        stringFromValue(model.Description),
		Fields:             customFieldsFromModel(model.Fields),
		ExtractionStrategy: customExtractionStrategyFromModel(model.ExtractionStrategy),
	}
}

func customMemoryTypeDefinitionEqual(left, right AgentMemoryCustomMemoryTypeModel) bool {
	if !left.Description.Equal(right.Description) {
		return false
	}
	if len(left.Fields) != len(right.Fields) {
		return false
	}
	for i := range left.Fields {
		if !customMemoryFieldEqual(left.Fields[i], right.Fields[i]) {
			return false
		}
	}
	return true
}

func customMemoryTypeDefinitionHasUnknown(memoryType AgentMemoryCustomMemoryTypeModel) bool {
	if memoryType.Description.IsUnknown() {
		return true
	}
	for _, field := range memoryType.Fields {
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
