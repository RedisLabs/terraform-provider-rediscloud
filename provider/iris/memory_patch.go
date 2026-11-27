package iris

import (
	"cmp"
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// apiPatch compares Terraform's plan with state; create uses empty prior state. Omitted or unknown scalar values
// leave the service setting alone; repeated detector blocks are authoritative.
// Write-only model keys come from configuration and require a version change.
func (m *AgentMemoryModel) apiPatch(ctx context.Context, config, previous AgentMemoryModel) (agentMemorySettings, diag.Diagnostics) {
	result := agentMemorySettings{Name: changedString(m.Name, previous.Name)}
	if ttl := changedInt64(m.ShortMemoryTTLSeconds, previous.ShortMemoryTTLSeconds); ttl != nil {
		result.ShortMemory = &memoryTTL{TTLSeconds: ttl}
	}
	if interval := changedInt64(m.ExtractionIntervalSeconds, previous.ExtractionIntervalSeconds); interval != nil {
		result.ExtractionCadence = &memoryCadence{ActiveIntervalSeconds: interval}
	}
	if mode := changedString(m.OwnershipMode, previous.OwnershipMode); mode != nil {
		result.Ownership = &memoryOwnership{Mode: mode}
	}
	result.LongTermMemory = m.longTermMemoryPatch(config, previous)
	result.Summarization = m.summarizationPatch(previous)
	exclusions, diagnostics := m.exclusionsPatch(ctx, previous)
	result.LongTermMemoryExclusions = exclusions
	return result, diagnostics
}

func (m *AgentMemoryModel) longTermMemoryPatch(config, previous AgentMemoryModel) *longTermMemoryConfig {
	result := longTermMemoryConfig{
		TTLSeconds:            changedInt64(m.LongTermMemoryTTLSeconds, previous.LongTermMemoryTTLSeconds),
		SessionSummaryEnabled: changedBool(m.SessionSummaryEnabled, previous.SessionSummaryEnabled),
		LLM:                   modelPatch(m.LLMProvider, m.LLMModel, config.LLMAPIKey, previous.LLMProvider, previous.LLMModel, changedInt64(m.LLMAPIKeyVersion, previous.LLMAPIKeyVersion) != nil),
		Embedding:             modelPatch(m.EmbeddingProvider, m.EmbeddingModel, config.EmbeddingAPIKey, previous.EmbeddingProvider, previous.EmbeddingModel, changedInt64(m.EmbeddingAPIKeyVersion, previous.EmbeddingAPIKeyVersion) != nil),
		Classification:        modelPatch(m.ClassificationProvider, m.ClassificationModel, config.ClassificationAPIKey, previous.ClassificationProvider, previous.ClassificationModel, changedInt64(m.ClassificationAPIKeyVersion, previous.ClassificationAPIKeyVersion) != nil),
	}
	if result == (longTermMemoryConfig{}) {
		return nil
	}
	return &result
}

func modelPatch(provider, model, key, previousProvider, previousModel types.String, rotate bool) *modelConfig {
	if changedString(provider, previousProvider) == nil && changedString(model, previousModel) == nil && !rotate {
		return nil
	}
	result := &modelConfig{
		Provider: cmp.Or(optionalString(provider), optionalString(previousProvider)),
		Model:    cmp.Or(optionalString(model), optionalString(previousModel)),
	}
	if rotate && optionalString(key) != nil {
		result.Credentials = &modelCredentials{Type: "API_KEY", APIKey: key.ValueString()}
	}
	return result
}

func (m *AgentMemoryModel) summarizationPatch(previous AgentMemoryModel) *memorySummarization {
	if changedBool(m.SummarizationEnabled, previous.SummarizationEnabled) == nil &&
		changedString(m.SummarizationTriggerStrategy, previous.SummarizationTriggerStrategy) == nil &&
		changedInt64(m.SummarizationEventThreshold, previous.SummarizationEventThreshold) == nil &&
		changedInt64(m.SummarizationRetainCount, previous.SummarizationRetainCount) == nil {
		return nil
	}
	// Iris replaces this group. Preserve unconfigured settings, including both
	// counts when changing just one of them.
	result := &memorySummarization{
		Enabled:         cmp.Or(optionalBool(m.SummarizationEnabled), optionalBool(previous.SummarizationEnabled)),
		TriggerStrategy: cmp.Or(optionalString(m.SummarizationTriggerStrategy), optionalString(previous.SummarizationTriggerStrategy)),
	}
	counts := memoryEventCount{
		Threshold:   cmp.Or(optionalInt64(m.SummarizationEventThreshold), optionalInt64(previous.SummarizationEventThreshold)),
		RetainCount: cmp.Or(optionalInt64(m.SummarizationRetainCount), optionalInt64(previous.SummarizationRetainCount)),
	}
	if counts != (memoryEventCount{}) {
		result.EventCount = &counts
	}
	return result
}

func (m *AgentMemoryModel) exclusionsPatch(ctx context.Context, previous AgentMemoryModel) (*memoryExclusions, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	result := memoryExclusions{Enabled: changedBool(m.ExclusionsEnabled, previous.ExclusionsEnabled)}
	if changedBool(m.SemanticExclusionsEnabled, previous.SemanticExclusionsEnabled) != nil || changedString(m.SemanticExclusionsPrompt, previous.SemanticExclusionsPrompt) != nil {
		result.Semantic = &memorySemanticExclusions{
			Enabled: cmp.Or(optionalBool(m.SemanticExclusionsEnabled), optionalBool(previous.SemanticExclusionsEnabled)),
			Prompt:  cmp.Or(optionalString(m.SemanticExclusionsPrompt), optionalString(previous.SemanticExclusionsPrompt)),
		}
	}
	if changedBool(m.BuiltInDetectorsEnabled, previous.BuiltInDetectorsEnabled) != nil || detectorSelectionsChanged(m.BuiltInDetectors, previous.BuiltInDetectors) {
		detectors, d := builtInDetectorsToAPI(ctx, m.BuiltInDetectors)
		diagnostics.Append(d...)
		result.BuiltInDetectors = &memoryBuiltInDetectors{
			Enabled:   cmp.Or(optionalBool(m.BuiltInDetectorsEnabled), optionalBool(previous.BuiltInDetectorsEnabled)),
			Detectors: detectors,
		}
	}
	if changedBool(m.CustomDetectorsEnabled, previous.CustomDetectorsEnabled) != nil || detectorSelectionsChanged(m.CustomDetectors, previous.CustomDetectors) {
		detectors, d := customDetectorsToAPI(ctx, m.CustomDetectors)
		diagnostics.Append(d...)
		result.CustomDetectors = &memoryCustomDetectors{
			Enabled:   cmp.Or(optionalBool(m.CustomDetectorsEnabled), optionalBool(previous.CustomDetectorsEnabled)),
			Detectors: detectors,
		}
	}
	if result == (memoryExclusions{}) {
		return nil, diagnostics
	}
	// Nested groups carry their required enabled flag. Omission would disable
	// exclusions on Iris even when only a detector or semantic prompt changed.
	result.Enabled = cmp.Or(optionalBool(m.ExclusionsEnabled), optionalBool(previous.ExclusionsEnabled))
	return &result, diagnostics
}

// Both null and empty repeated blocks mean no detector selections.
func detectorSelectionsChanged(next, previous types.Set) bool {
	return (len(next.Elements()) > 0 || len(previous.Elements()) > 0) && !next.Equal(previous)
}

func builtInDetectorsToAPI(ctx context.Context, value types.Set) ([]memoryBuiltInDetector, diag.Diagnostics) {
	result := make([]memoryBuiltInDetector, 0)
	if value.IsNull() || value.IsUnknown() {
		return result, nil
	}
	var models []BuiltInDetectorModel
	d := value.ElementsAs(ctx, &models, false)
	for _, m := range models {
		result = append(result, memoryBuiltInDetector{ID: m.ID.ValueString(), Enabled: m.Enabled.ValueBool(), Action: optionalString(m.Action)})
	}
	return result, d
}

func customDetectorsToAPI(ctx context.Context, value types.Set) ([]memoryCustomDetector, diag.Diagnostics) {
	result := make([]memoryCustomDetector, 0)
	if value.IsNull() || value.IsUnknown() {
		return result, nil
	}
	var models []CustomDetectorModel
	d := value.ElementsAs(ctx, &models, false)
	for _, m := range models {
		result = append(result, memoryCustomDetector{Name: m.Name.ValueString(), Enabled: m.Enabled.ValueBool(), Action: optionalString(m.Action), Matcher: memoryMatcher{Kind: "REGEX", Regex: memoryRegex{Pattern: m.Pattern.ValueString()}}})
	}
	return result, d
}
