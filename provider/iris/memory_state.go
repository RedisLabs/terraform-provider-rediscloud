package iris

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// GET responses are authoritative; omitted service settings are cleared.
func (m *AgentMemoryModel) setFromAPI(ctx context.Context, current *agentMemoryResponse) diag.Diagnostics {
	failure := current.Failure
	if failure == nil {
		failure = &memoryFailure{}
	}
	m.CreatedAt = types.StringPointerValue(current.CreatedAt)
	m.ExtractionStrategy = types.StringPointerValue(current.ExtractionStrategy)
	m.FailureMessage = types.StringPointerValue(failure.Message)
	m.FailureOccurredAt = types.StringPointerValue(failure.OccurredAt)
	m.State = types.StringPointerValue(current.State)
	m.SubscriptionID = types.StringPointerValue(current.SubscriptionID)
	m.Name = types.StringPointerValue(current.Name)
	if current.StoreID != nil {
		m.ID = types.StringPointerValue(current.StoreID)
	}
	if current.Database != nil {
		m.DatabaseID = types.StringValue(current.Database.DatabaseID)
	}
	// Inline custom types are a create-only snapshot omitted from GET responses.
	if m.CustomMemoryTypes.IsNull() || m.CustomMemoryTypes.IsUnknown() {
		m.CustomMemoryTypes = types.SetValueMust(customMemoryTypeElementType, nil)
	}
	m.setMemorySettingsFromAPI(current)
	m.setSummarizationFromAPI(current.Summarization)
	m.setModelSelectionsFromAPI(current.LongTermMemory)
	diagnostics := m.setExclusionsFromAPI(ctx, current.LongTermMemoryExclusions)
	diagnostics.Append(m.setEndpointsFromAPI(ctx, current.Endpoints)...)
	return diagnostics
}

func (m *AgentMemoryModel) setMemorySettingsFromAPI(current *agentMemoryResponse) {
	short := current.ShortMemory
	if short == nil {
		short = &memoryTTL{}
	}
	long := current.LongTermMemory
	if long == nil {
		long = &longTermMemoryConfig{}
	}
	cadence := current.ExtractionCadence
	if cadence == nil {
		cadence = &memoryCadence{}
	}
	ownership := current.Ownership
	if ownership == nil {
		ownership = &memoryOwnership{}
	}
	m.ShortMemoryTTLSeconds = types.Int64PointerValue(short.TTLSeconds)
	m.LongTermMemoryTTLSeconds = types.Int64PointerValue(long.TTLSeconds)
	m.SessionSummaryEnabled = types.BoolPointerValue(long.SessionSummaryEnabled)
	m.ExtractionIntervalSeconds = types.Int64PointerValue(cadence.ActiveIntervalSeconds)
	m.OwnershipMode = types.StringPointerValue(ownership.Mode)
	m.OwnershipEnabled = types.BoolPointerValue(ownership.Enabled)
	m.OwnershipSource = types.StringPointerValue(ownership.Source)
}

func (m *AgentMemoryModel) setSummarizationFromAPI(summarization *memorySummarization) {
	if summarization == nil {
		summarization = &memorySummarization{}
	}
	event := summarization.EventCount
	if event == nil {
		event = &memoryEventCount{}
	}
	m.SummarizationEnabled = types.BoolPointerValue(summarization.Enabled)
	m.SummarizationTriggerStrategy = types.StringPointerValue(summarization.TriggerStrategy)
	m.SummarizationEventThreshold = types.Int64PointerValue(event.Threshold)
	m.SummarizationRetainCount = types.Int64PointerValue(event.RetainCount)
}

func (m *AgentMemoryModel) setModelSelectionsFromAPI(long *longTermMemoryConfig) {
	if long == nil {
		long = &longTermMemoryConfig{}
	}
	llm := long.LLM
	if llm == nil {
		llm = &modelConfig{}
	}
	embedding := long.Embedding
	if embedding == nil {
		embedding = &modelConfig{}
	}
	classification := long.Classification
	if classification == nil {
		classification = &modelConfig{}
	}
	m.LLMProvider = types.StringPointerValue(llm.Provider)
	m.LLMModel = types.StringPointerValue(llm.Model)
	m.EmbeddingProvider = types.StringPointerValue(embedding.Provider)
	m.EmbeddingModel = types.StringPointerValue(embedding.Model)
	m.ClassificationProvider = types.StringPointerValue(classification.Provider)
	m.ClassificationModel = types.StringPointerValue(classification.Model)
}

func (m *AgentMemoryModel) setExclusionsFromAPI(ctx context.Context, exclusions *memoryExclusions) diag.Diagnostics {
	if exclusions == nil {
		exclusions = &memoryExclusions{}
	}
	semantic := exclusions.Semantic
	if semantic == nil {
		semantic = &memorySemanticExclusions{}
	}
	builtIn := exclusions.BuiltInDetectors
	if builtIn == nil {
		builtIn = &memoryBuiltInDetectors{}
	}
	custom := exclusions.CustomDetectors
	if custom == nil {
		custom = &memoryCustomDetectors{}
	}
	m.ExclusionsEnabled = types.BoolPointerValue(exclusions.Enabled)
	m.SemanticExclusionsEnabled = types.BoolPointerValue(semantic.Enabled)
	m.SemanticExclusionsPrompt = types.StringPointerValue(semantic.Prompt)
	m.BuiltInDetectorsEnabled = types.BoolPointerValue(builtIn.Enabled)
	m.CustomDetectorsEnabled = types.BoolPointerValue(custom.Enabled)
	var diagnostics diag.Diagnostics
	builtModels := make([]BuiltInDetectorModel, 0, len(builtIn.Detectors))
	for _, v := range builtIn.Detectors {
		builtModels = append(builtModels, BuiltInDetectorModel{ID: types.StringValue(v.ID), Enabled: types.BoolValue(v.Enabled), Action: types.StringPointerValue(v.Action)})
	}
	builtValue, d := types.SetValueFrom(ctx, builtInDetectorElementType, builtModels)
	diagnostics.Append(d...)
	m.BuiltInDetectors = builtValue
	customModels := make([]CustomDetectorModel, 0, len(custom.Detectors))
	for _, v := range custom.Detectors {
		customModels = append(customModels, CustomDetectorModel{Name: types.StringValue(v.Name), Enabled: types.BoolValue(v.Enabled), Action: types.StringPointerValue(v.Action), Pattern: types.StringValue(v.Matcher.Regex.Pattern)})
	}
	customValue, d := types.SetValueFrom(ctx, customDetectorElementType, customModels)
	diagnostics.Append(d...)
	m.CustomDetectors = customValue
	return diagnostics
}

func (m *AgentMemoryModel) setEndpointsFromAPI(ctx context.Context, endpoints []memoryEndpoint) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	models := make([]EndpointModel, 0, len(endpoints))
	for _, v := range endpoints {
		ips, d := types.SetValueFrom(ctx, types.StringType, v.EgressIPs)
		diagnostics.Append(d...)
		models = append(models, EndpointModel{URL: types.StringValue(v.URL), Provider: types.StringValue(v.Provider), Region: types.StringValue(v.Region), EgressIPs: ips, Accessible: types.BoolValue(v.Accessible)})
	}
	value, d := types.SetValueFrom(ctx, endpointElementType, models)
	diagnostics.Append(d...)
	m.Endpoints = value
	return diagnostics
}
