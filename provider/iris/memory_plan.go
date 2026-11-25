package iris

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var _ resource.ResourceWithModifyPlan = (*agentMemoryResource)(nil)

func (*agentMemoryResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var config, previous AgentMemoryModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &previous)...)
	}
	if resp.Diagnostics.HasError() {
		return
	}
	config.validateGroups(previous, &resp.Diagnostics)
	config.validateModelSelections(previous, &resp.Diagnostics)
}

func (m AgentMemoryModel) validateModelSelections(previous AgentMemoryModel, diagnostics *diag.Diagnostics) {
	// Selecting either model requires both selections. Credentials depend on the
	// selected provider and are validated by Iris, including keyless providers.
	selected := optionalString(m.LLMProvider) != nil || optionalString(m.LLMModel) != nil || optionalString(m.LLMAPIKey) != nil ||
		optionalString(m.EmbeddingProvider) != nil || optionalString(m.EmbeddingModel) != nil || optionalString(m.EmbeddingAPIKey) != nil
	requireStoreSetting("llm_provider", m.LLMProvider, previous.LLMProvider, selected, diagnostics)
	requireStoreSetting("llm_model", m.LLMModel, previous.LLMModel, selected, diagnostics)
	requireStoreSetting("embedding_provider", m.EmbeddingProvider, previous.EmbeddingProvider, selected, diagnostics)
	requireStoreSetting("embedding_model", m.EmbeddingModel, previous.EmbeddingModel, selected, diagnostics)
	classification := optionalString(m.ClassificationProvider) != nil || optionalString(m.ClassificationModel) != nil || optionalString(m.ClassificationAPIKey) != nil
	requireStoreSetting("classification_provider", m.ClassificationProvider, previous.ClassificationProvider, classification, diagnostics)
	requireStoreSetting("classification_model", m.ClassificationModel, previous.ClassificationModel, classification, diagnostics)
}

func (m AgentMemoryModel) validateGroups(previous AgentMemoryModel, diagnostics *diag.Diagnostics) {
	counts := optionalInt64(m.SummarizationEventThreshold) != nil || optionalInt64(m.SummarizationRetainCount) != nil
	requireStoreSetting("summarization_enabled", m.SummarizationEnabled, previous.SummarizationEnabled, counts || optionalString(m.SummarizationTriggerStrategy) != nil, diagnostics)
	enabled := m.SummarizationEnabled.ValueBool() || (m.SummarizationEnabled.IsNull() && previous.SummarizationEnabled.ValueBool())
	requireStoreSetting("summarization_event_threshold", m.SummarizationEventThreshold, previous.SummarizationEventThreshold, enabled, diagnostics)
	semantic := optionalBool(m.SemanticExclusionsEnabled) != nil || optionalString(m.SemanticExclusionsPrompt) != nil
	builtIn := optionalBool(m.BuiltInDetectorsEnabled) != nil || len(m.BuiltInDetectors.Elements()) > 0
	custom := optionalBool(m.CustomDetectorsEnabled) != nil || len(m.CustomDetectors.Elements()) > 0
	requireStoreSetting("exclusions_enabled", m.ExclusionsEnabled, previous.ExclusionsEnabled, semantic || builtIn || custom, diagnostics)
	requireStoreSetting("semantic_exclusions_enabled", m.SemanticExclusionsEnabled, previous.SemanticExclusionsEnabled, optionalString(m.SemanticExclusionsPrompt) != nil, diagnostics)
	requireStoreSetting("built_in_detectors_enabled", m.BuiltInDetectorsEnabled, previous.BuiltInDetectorsEnabled, len(m.BuiltInDetectors.Elements()) > 0, diagnostics)
	requireStoreSetting("custom_detectors_enabled", m.CustomDetectorsEnabled, previous.CustomDetectorsEnabled, len(m.CustomDetectors.Elements()) > 0, diagnostics)
}

// Omitted settings can be inherited on update. Unknown configured values defer
// validation; missing creation settings must not silently disable a group.
func requireStoreSetting(name string, configured, previous attr.Value, needed bool, diagnostics *diag.Diagnostics) {
	if needed && configured.IsNull() && previous.IsNull() {
		diagnostics.AddAttributeError(path.Root(name), "Missing store setting", fmt.Sprintf("Configure %s explicitly when supplying its related settings. Existing stores may omit it when its value is already known in state.", name))
	}
}
