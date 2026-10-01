package agentmemory

import (
	agentmemoryapi "github.com/RedisLabs/rediscloud-go-api/service/agentmemory"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func summarizationFromModel(model *AgentMemorySummarizationModel) *agentmemoryapi.SummarizationConfig {
	if model == nil {
		return nil
	}

	config := &agentmemoryapi.SummarizationConfig{
		Enabled: model.Enabled.ValueBool(),
	}
	if !model.TriggerStrategy.IsNull() && !model.TriggerStrategy.IsUnknown() {
		config.TriggerStrategy = model.TriggerStrategy.ValueString()
	}
	if model.EventCount != nil {
		config.EventCount = &agentmemoryapi.SummarizationThresholdConfig{
			Threshold:   int(model.EventCount.Threshold.ValueInt64()),
			RetainCount: int(model.EventCount.RetainCount.ValueInt64()),
		}
	}

	return config
}

func summarizationToModel(config *agentmemoryapi.SummarizationConfig) *AgentMemorySummarizationModel {
	if config == nil {
		return nil
	}

	model := &AgentMemorySummarizationModel{
		Enabled:         types.BoolValue(config.Enabled),
		TriggerStrategy: stringValue(config.TriggerStrategy),
	}
	if !config.Enabled {
		model.TriggerStrategy = types.StringNull()
		return model
	}
	if config.EventCount != nil {
		model.EventCount = &AgentMemoryEventCountModel{
			Threshold:   types.Int64Value(int64(config.EventCount.Threshold)),
			RetainCount: types.Int64Value(int64(config.EventCount.RetainCount)),
		}
	}

	return model
}

func exclusionsFromModel(model *AgentMemoryExclusionsModel) *agentmemoryapi.LongTermMemoryExclusions {
	if model == nil {
		return nil
	}

	config := &agentmemoryapi.LongTermMemoryExclusions{
		Enabled: model.Enabled.ValueBool(),
	}
	if model.Semantic != nil {
		config.Semantic = &agentmemoryapi.SemanticExclusion{
			Enabled: model.Semantic.Enabled.ValueBool(),
			Prompt:  stringFromValue(model.Semantic.Prompt),
		}
	}
	if model.BuiltInDetectors != nil {
		config.BuiltInDetectors = &agentmemoryapi.BuiltInDetectorsConfig{
			Enabled:   model.BuiltInDetectors.Enabled.ValueBool(),
			Detectors: detectorSelectionsFromModel(model.BuiltInDetectors.Detectors),
		}
	}
	if model.CustomDetectors != nil {
		config.CustomDetectors = &agentmemoryapi.CustomDetectorsConfig{
			Enabled:   model.CustomDetectors.Enabled.ValueBool(),
			Detectors: customDetectorsFromModel(model.CustomDetectors.Detectors),
		}
	}

	return config
}

func exclusionsToModel(config *agentmemoryapi.LongTermMemoryExclusions) *AgentMemoryExclusionsModel {
	if config == nil {
		return nil
	}

	model := &AgentMemoryExclusionsModel{
		Enabled: types.BoolValue(config.Enabled),
	}
	if config.Semantic != nil {
		model.Semantic = &AgentMemorySemanticExclusionModel{
			Enabled: types.BoolValue(config.Semantic.Enabled),
			Prompt:  stringValue(config.Semantic.Prompt),
		}
		if !config.Semantic.Enabled {
			model.Semantic.Prompt = types.StringNull()
		}
	}
	if config.BuiltInDetectors != nil {
		model.BuiltInDetectors = &AgentMemoryBuiltInDetectorsModel{
			Enabled:   types.BoolValue(config.BuiltInDetectors.Enabled),
			Detectors: detectorSelectionsToModel(config.BuiltInDetectors.Detectors),
		}
		if !config.BuiltInDetectors.Enabled {
			model.BuiltInDetectors.Detectors = nil
		}
	}
	if config.CustomDetectors != nil {
		model.CustomDetectors = &AgentMemoryCustomDetectorsModel{
			Enabled:   types.BoolValue(config.CustomDetectors.Enabled),
			Detectors: customDetectorsToModel(config.CustomDetectors.Detectors),
		}
		if !config.CustomDetectors.Enabled {
			model.CustomDetectors.Detectors = nil
		}
	}

	return model
}

func detectorSelectionsFromModel(models []AgentMemoryDetectorSelectionModel) []agentmemoryapi.DetectorSelection {
	if len(models) == 0 {
		return nil
	}
	selections := make([]agentmemoryapi.DetectorSelection, 0, len(models))
	for _, model := range models {
		selections = append(selections, agentmemoryapi.DetectorSelection{
			ID:      stringFromValue(model.ID),
			Enabled: model.Enabled.ValueBool(),
			Action:  stringFromValue(model.Action),
		})
	}
	return selections
}

func detectorSelectionsToModel(selections []agentmemoryapi.DetectorSelection) []AgentMemoryDetectorSelectionModel {
	if len(selections) == 0 {
		return nil
	}
	models := make([]AgentMemoryDetectorSelectionModel, 0, len(selections))
	for _, selection := range selections {
		models = append(models, AgentMemoryDetectorSelectionModel{
			ID:      stringValue(selection.ID),
			Enabled: types.BoolValue(selection.Enabled),
			Action:  stringValue(selection.Action),
		})
	}
	return models
}

func customDetectorsFromModel(models []AgentMemoryCustomDetectorModel) []agentmemoryapi.CustomDetector {
	if len(models) == 0 {
		return nil
	}
	detectors := make([]agentmemoryapi.CustomDetector, 0, len(models))
	for _, model := range models {
		detectors = append(detectors, agentmemoryapi.CustomDetector{
			Name:    stringFromValue(model.Name),
			Enabled: model.Enabled.ValueBool(),
			Action:  stringFromValue(model.Action),
			Matcher: matcherFromModel(model.Matcher),
		})
	}
	return detectors
}

func customDetectorsToModel(detectors []agentmemoryapi.CustomDetector) []AgentMemoryCustomDetectorModel {
	if len(detectors) == 0 {
		return nil
	}
	models := make([]AgentMemoryCustomDetectorModel, 0, len(detectors))
	for _, detector := range detectors {
		models = append(models, AgentMemoryCustomDetectorModel{
			Name:    stringValue(detector.Name),
			Enabled: types.BoolValue(detector.Enabled),
			Action:  stringValue(detector.Action),
			Matcher: matcherToModel(detector.Matcher),
		})
	}
	return models
}

func matcherFromModel(model *AgentMemoryMatcherSpecModel) *agentmemoryapi.MatcherSpec {
	if model == nil {
		return nil
	}
	return &agentmemoryapi.MatcherSpec{
		Kind:  stringFromValue(model.Kind),
		Regex: regexFromModel(model.Regex),
	}
}

func matcherToModel(matcher *agentmemoryapi.MatcherSpec) *AgentMemoryMatcherSpecModel {
	if matcher == nil {
		return nil
	}
	return &AgentMemoryMatcherSpecModel{
		Kind:  stringValue(matcher.Kind),
		Regex: regexToModel(matcher.Regex),
	}
}

func regexFromModel(model *AgentMemoryRegexSpecModel) *agentmemoryapi.RegexSpec {
	if model == nil {
		return nil
	}
	return &agentmemoryapi.RegexSpec{Pattern: stringFromValue(model.Pattern)}
}

func regexToModel(regex *agentmemoryapi.RegexSpec) *AgentMemoryRegexSpecModel {
	if regex == nil {
		return nil
	}
	return &AgentMemoryRegexSpecModel{Pattern: stringValue(regex.Pattern)}
}

func customMemoryTypesFromModel(models []AgentMemoryCustomMemoryTypeModel) []agentmemoryapi.CustomMemoryType {
	if len(models) == 0 {
		return nil
	}
	types := make([]agentmemoryapi.CustomMemoryType, 0, len(models))
	for _, model := range models {
		types = append(types, customMemoryTypeFromModel(model))
	}
	return types
}

func customMemoryTypesToModel(memoryTypes []agentmemoryapi.CustomMemoryType) []AgentMemoryCustomMemoryTypeModel {
	if len(memoryTypes) == 0 {
		return nil
	}
	models := make([]AgentMemoryCustomMemoryTypeModel, 0, len(memoryTypes))
	for _, memoryType := range memoryTypes {
		models = append(models, AgentMemoryCustomMemoryTypeModel{
			Name:               stringValue(memoryType.Name),
			Description:        stringValue(memoryType.Description),
			Fields:             customFieldsToModel(memoryType.Fields),
			ExtractionStrategy: customExtractionStrategyToModel(memoryType.ExtractionStrategy),
		})
	}
	return models
}

func customFieldsFromModel(models []AgentMemoryCustomFieldModel) []agentmemoryapi.CustomField {
	if len(models) == 0 {
		return nil
	}
	fields := make([]agentmemoryapi.CustomField, 0, len(models))
	for _, model := range models {
		fields = append(fields, agentmemoryapi.CustomField{
			Name:        stringFromValue(model.Name),
			Description: stringFromValue(model.Description),
			Type:        stringFromValue(model.Type),
		})
	}
	return fields
}

func customFieldsToModel(fields []agentmemoryapi.CustomField) []AgentMemoryCustomFieldModel {
	if len(fields) == 0 {
		return nil
	}
	models := make([]AgentMemoryCustomFieldModel, 0, len(fields))
	for _, field := range fields {
		models = append(models, AgentMemoryCustomFieldModel{
			Name:        stringValue(field.Name),
			Description: stringValue(field.Description),
			Type:        stringValue(field.Type),
		})
	}
	return models
}

func customExtractionStrategyFromModel(model *AgentMemoryCustomExtractionStrategy) *agentmemoryapi.CustomExtractionStrategy {
	if model == nil {
		return nil
	}
	strategy := &agentmemoryapi.CustomExtractionStrategy{
		Prompt: stringFromValue(model.Prompt),
	}
	if !model.Enabled.IsNull() && !model.Enabled.IsUnknown() {
		enabled := model.Enabled.ValueBool()
		strategy.Enabled = &enabled
	}
	return strategy
}

func customExtractionStrategyToModel(strategy *agentmemoryapi.CustomExtractionStrategy) *AgentMemoryCustomExtractionStrategy {
	if strategy == nil {
		return nil
	}
	model := &AgentMemoryCustomExtractionStrategy{
		Prompt: stringValue(strategy.Prompt),
	}
	if strategy.Enabled != nil {
		model.Enabled = types.BoolValue(*strategy.Enabled)
	} else {
		model.Enabled = types.BoolNull()
	}
	return model
}

func stringFromValue(value types.String) string {
	if value.IsNull() || value.IsUnknown() {
		return ""
	}
	return value.ValueString()
}

func stringValue(value string) types.String {
	if value == "" {
		return types.StringNull()
	}
	return types.StringValue(value)
}
