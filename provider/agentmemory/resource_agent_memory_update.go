package agentmemory

import (
	"context"
	"reflect"
	"slices"
	"strings"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	agentmemoryapi "github.com/RedisLabs/rediscloud-go-api/service/agentmemory"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func hasAgentMemoryUpdates(update agentmemoryapi.UpdateStore) bool {
	return update.Name != nil || update.ShortMemory != nil || update.LongTermMemory != nil ||
		update.LLM != nil || update.Summarization != nil || update.LongTermMemoryExclusions != nil ||
		update.ExtractionCadence != nil || len(update.AddCustomMemoryTypes) > 0 ||
		len(update.UpdateCustomMemoryTypeStrategies) > 0
}

func exclusionsConfigEqual(ctx context.Context, plan, state ExclusionsListValue, diags *diag.Diagnostics) bool {
	plannedModel := exclusionsBlock(ctx, plan, diags)
	existingModel := exclusionsBlock(ctx, state, diags)
	planned := exclusionsFromModel(ctx, plannedModel, diags)
	existing := exclusionsFromModel(ctx, existingModel, diags)
	if planned == nil || existing == nil {
		return planned == nil && existing == nil
	}
	if plannedModel.Enabled.IsUnknown() {
		planned.Enabled = existing.Enabled
	}
	if planned.Semantic != nil && existing.Semantic != nil {
		semantic := semanticExclusionBlock(ctx, plannedModel.Semantic, diags)
		if semantic.Enabled.IsUnknown() {
			planned.Semantic.Enabled = existing.Semantic.Enabled
		}
		if semantic.Prompt.IsUnknown() {
			planned.Semantic.Prompt = existing.Semantic.Prompt
		}
	}
	if planned.BuiltInDetectors != nil && existing.BuiltInDetectors != nil {
		group := builtInDetectorsBlock(ctx, plannedModel.BuiltInDetectors, diags)
		previous := builtInDetectorsBlock(ctx, existingModel.BuiltInDetectors, diags)
		if group.Enabled.IsUnknown() {
			planned.BuiltInDetectors.Enabled = existing.BuiltInDetectors.Enabled
		}
		planned.BuiltInDetectors.Detectors = detectorSelectionsForComparison(ctx, group.Detectors, previous.Detectors, diags)
	}
	if planned.CustomDetectors != nil && existing.CustomDetectors != nil {
		group := customDetectorsBlock(ctx, plannedModel.CustomDetectors, diags)
		previous := customDetectorsBlock(ctx, existingModel.CustomDetectors, diags)
		if group.Enabled.IsUnknown() {
			planned.CustomDetectors.Enabled = existing.CustomDetectors.Enabled
		}
		planned.CustomDetectors.Detectors = customDetectorsForComparison(ctx, group.Detectors, previous.Detectors, diags)
	}
	// Detector order belongs to Terraform state, not to the remote configuration.
	for _, config := range []*agentmemoryapi.LongTermMemoryExclusions{planned, existing} {
		if config == nil {
			continue
		}
		if config.BuiltInDetectors != nil {
			slices.SortStableFunc(config.BuiltInDetectors.Detectors, func(a, b agentmemoryapi.DetectorSelection) int {
				return strings.Compare(redis.StringValue(a.ID), redis.StringValue(b.ID))
			})
		}
		if config.CustomDetectors != nil {
			slices.SortStableFunc(config.CustomDetectors.Detectors, func(a, b agentmemoryapi.CustomDetector) int {
				return strings.Compare(redis.StringValue(a.Name), redis.StringValue(b.Name))
			})
		}
	}
	return reflect.DeepEqual(planned, existing)
}

func summarizationConfigEqual(ctx context.Context, plan, state SummarizationListValue, diags *diag.Diagnostics) bool {
	planned := summarizationFromModel(ctx, summarizationBlock(ctx, plan, diags), diags)
	existing := summarizationFromModel(ctx, summarizationBlock(ctx, state, diags), diags)
	if planned == nil || existing == nil {
		return planned == nil && existing == nil
	}
	// Omitted PATCH fields leave the current remote value unchanged.
	if planned.TriggerStrategy == nil {
		planned.TriggerStrategy = existing.TriggerStrategy
	}
	if planned.EventCount == nil {
		planned.EventCount = existing.EventCount
	}
	return reflect.DeepEqual(planned, existing)
}

func applyLocalAgentMemoryPlan(ctx context.Context, plan AgentMemoryResourceModel, state *AgentMemoryResourceModel, diags *diag.Diagnostics) {
	state.Timeouts = plan.Timeouts
	// Read mappings keep computed values known while retaining configured block presence and order.
	if len(plan.CustomMemoryTypes.Elements()) == 0 {
		state.CustomMemoryTypes = plan.CustomMemoryTypes
	} else {
		current := customMemoryTypesFromModel(ctx, state.CustomMemoryTypes, diags)
		planned := customMemoryTypeModels(ctx, plan.CustomMemoryTypes, diags)
		models := customMemoryTypesFromAPI(ctx, current, planned, diags)
		list, listDiags := NewCustomMemoryTypeList(ctx, models)
		diags.Append(listDiags...)
		state.CustomMemoryTypes = list
	}
	if len(plan.Summarization.Elements()) == 0 {
		state.Summarization = plan.Summarization
	} else {
		current := summarizationFromModel(ctx, summarizationBlock(ctx, state.Summarization, diags), diags)
		planned := summarizationBlock(ctx, plan.Summarization, diags)
		model := summarizationFromAPI(ctx, current, planned, false, diags)
		state.Summarization = summarizationList(ctx, model, diags)
	}
	if len(plan.LongTermMemoryExclusions.Elements()) == 0 {
		state.LongTermMemoryExclusions = plan.LongTermMemoryExclusions
	} else {
		current := exclusionsFromModel(ctx, exclusionsBlock(ctx, state.LongTermMemoryExclusions, diags), diags)
		planned := exclusionsBlock(ctx, plan.LongTermMemoryExclusions, diags)
		model := exclusionsFromAPI(ctx, current, planned, false, diags)
		state.LongTermMemoryExclusions = exclusionsList(ctx, model, diags)
	}
}

// Computed defaults may become unknown during a local edit. Match identities before reusing them.
func detectorSelectionsForComparison(ctx context.Context, plan, state DetectorSelectionListValue, diags *diag.Diagnostics) []agentmemoryapi.DetectorSelection {
	models := detectorSelectionModels(ctx, plan, diags)
	previous := make(map[string]AgentMemoryDetectorSelectionModel)
	for _, model := range detectorSelectionModels(ctx, state, diags) {
		previous[model.ID.ValueString()] = model
	}
	for i, model := range models {
		existing, found := previous[model.ID.ValueString()]
		if !found {
			continue
		}
		if model.Enabled.IsUnknown() {
			models[i].Enabled = existing.Enabled
		}
		if model.Action.IsUnknown() {
			models[i].Action = existing.Action
		}
	}
	return detectorSelectionsFromModel(models)
}

func customDetectorsForComparison(ctx context.Context, plan, state CustomDetectorListValue, diags *diag.Diagnostics) []agentmemoryapi.CustomDetector {
	models := customDetectorModels(ctx, plan, diags)
	previous := make(map[string]AgentMemoryCustomDetectorModel)
	for _, model := range customDetectorModels(ctx, state, diags) {
		previous[model.Name.ValueString()] = model
	}
	for i, model := range models {
		existing, found := previous[model.Name.ValueString()]
		if !found {
			continue
		}
		if model.Enabled.IsUnknown() {
			models[i].Enabled = existing.Enabled
		}
		if model.Action.IsUnknown() {
			models[i].Action = existing.Action
		}
	}
	return customDetectorsFromModel(ctx, models, diags)
}
