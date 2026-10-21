package agentmemory

import (
	"github.com/RedisLabs/rediscloud-go-api/redis"
	agentmemoryapi "github.com/RedisLabs/rediscloud-go-api/service/agentmemory"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func modelConfigFromModel(model *AgentMemoryModelConfigModel) *agentmemoryapi.ModelConfig {
	if model == nil {
		return nil
	}

	config := &agentmemoryapi.ModelConfig{
		Provider: stringPointerFromValue(model.Provider),
		Model:    stringPointerFromValue(model.Model),
	}
	if model.Credentials != nil {
		config.Credentials = &agentmemoryapi.ModelCredentials{
			Type: stringPointerFromValue(model.Credentials.Type),
		}
	}

	return config
}

func modelConfigToModel(status *agentmemoryapi.ModelStatus, existing *AgentMemoryModelConfigModel) *AgentMemoryModelConfigModel {
	if status == nil {
		return existing
	}

	// The API does not return credentials. Preserve configured values across refresh;
	// an imported store has no credentials to recover.
	model := &AgentMemoryModelConfigModel{
		Provider:    types.StringPointerValue(status.Provider),
		Model:       types.StringPointerValue(status.Model),
		Credentials: modelCredentialsFromExisting(existing),
	}
	return model
}

func modelCredentialsFromExisting(existing *AgentMemoryModelConfigModel) *AgentMemoryModelCredentialsModel {
	if existing == nil || existing.Credentials == nil {
		return nil
	}
	return &AgentMemoryModelCredentialsModel{
		Type:          existing.Credentials.Type,
		APIKey:        existing.Credentials.APIKey,
		APIKeyVersion: existing.Credentials.APIKeyVersion,
	}
}

func modelConfigEqual(left, right *AgentMemoryModelConfigModel) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	if !left.Provider.Equal(right.Provider) || !left.Model.Equal(right.Model) {
		return false
	}
	return modelCredentialsEqual(left.Credentials, right.Credentials)
}

func modelCredentialsEqual(left, right *AgentMemoryModelCredentialsModel) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Type.Equal(right.Type) && left.APIKey.Equal(right.APIKey) && left.APIKeyVersion.Equal(right.APIKeyVersion)
}

func summarizationFromModel(model *AgentMemorySummarizationModel) *agentmemoryapi.SummarizationConfig {
	if model == nil {
		return nil
	}

	config := &agentmemoryapi.SummarizationConfig{
		Enabled: redis.Bool(model.Enabled.ValueBool()),
	}
	if !model.TriggerStrategy.IsNull() && !model.TriggerStrategy.IsUnknown() {
		config.TriggerStrategy = model.TriggerStrategy.ValueStringPointer()
	}
	if model.EventCount != nil {
		config.EventCount = &agentmemoryapi.SummarizationThresholdConfig{
			Threshold:   redis.Int(int(model.EventCount.Threshold.ValueInt64())),
			RetainCount: redis.Int(int(model.EventCount.RetainCount.ValueInt64())),
		}
	}

	return config
}

func summarizationToModel(config *agentmemoryapi.SummarizationConfig, existing *AgentMemorySummarizationModel, includeAPIOnlyConfig bool) *AgentMemorySummarizationModel {
	if config == nil {
		return nil
	}

	model := &AgentMemorySummarizationModel{
		Enabled:         types.BoolPointerValue(config.Enabled),
		TriggerStrategy: types.StringPointerValue(config.TriggerStrategy),
	}
	// Managed reads must not add optional blocks absent from the plan or state.
	if config.EventCount != nil && (includeAPIOnlyConfig || (existing != nil && existing.EventCount != nil)) {
		model.EventCount = &AgentMemoryEventCountModel{
			Threshold:   intPointerValue(config.EventCount.Threshold),
			RetainCount: intPointerValue(config.EventCount.RetainCount),
		}
	}

	// A disabled policy can still contain explicitly configured definitions.
	// Omitted children must not acquire stale API-only values during refresh.
	if config.Enabled != nil && !*config.Enabled && !includeAPIOnlyConfig {
		if existing == nil || existing.TriggerStrategy.IsNull() || existing.TriggerStrategy.IsUnknown() {
			model.TriggerStrategy = types.StringNull()
		}
	}
	return model
}

func exclusionsFromModel(model *AgentMemoryExclusionsModel) *agentmemoryapi.LongTermMemoryExclusions {
	if model == nil {
		return nil
	}

	config := &agentmemoryapi.LongTermMemoryExclusions{
		Enabled: redis.Bool(model.Enabled.ValueBool()),
	}
	if model.Semantic != nil {
		config.Semantic = &agentmemoryapi.SemanticExclusion{
			Enabled: redis.Bool(model.Semantic.Enabled.ValueBool()),
			Prompt:  stringPointerFromValue(model.Semantic.Prompt),
		}
	}
	if model.BuiltInDetectors != nil {
		config.BuiltInDetectors = &agentmemoryapi.BuiltInDetectorsConfig{
			Enabled:   redis.Bool(model.BuiltInDetectors.Enabled.ValueBool()),
			Detectors: detectorSelectionsFromModel(model.BuiltInDetectors.Detectors),
		}
	}
	if model.CustomDetectors != nil {
		config.CustomDetectors = &agentmemoryapi.CustomDetectorsConfig{
			Enabled:   redis.Bool(model.CustomDetectors.Enabled.ValueBool()),
			Detectors: customDetectorsFromModel(model.CustomDetectors.Detectors),
		}
	}

	return config
}

func exclusionsToModel(config *agentmemoryapi.LongTermMemoryExclusions, existing *AgentMemoryExclusionsModel, includeAPIOnlyConfig bool) *AgentMemoryExclusionsModel {
	if config == nil {
		return nil
	}

	model := &AgentMemoryExclusionsModel{
		Enabled: types.BoolPointerValue(config.Enabled),
	}
	// The API retains omitted groups. Only import should adopt unconfigured blocks.
	if config.Semantic != nil && (includeAPIOnlyConfig || (existing != nil && existing.Semantic != nil)) {
		model.Semantic = &AgentMemorySemanticExclusionModel{
			Enabled: types.BoolPointerValue(config.Semantic.Enabled),
			Prompt:  types.StringPointerValue(config.Semantic.Prompt),
		}
		if (config.Semantic.Enabled != nil && !*config.Semantic.Enabled) && !includeAPIOnlyConfig && (existing == nil || existing.Semantic == nil || existing.Semantic.Prompt.IsNull() || existing.Semantic.Prompt.IsUnknown()) {
			model.Semantic.Prompt = types.StringNull()
		}
	}
	if config.BuiltInDetectors != nil && (includeAPIOnlyConfig || (existing != nil && existing.BuiltInDetectors != nil)) {
		model.BuiltInDetectors = &AgentMemoryBuiltInDetectorsModel{
			Enabled:   types.BoolPointerValue(config.BuiltInDetectors.Enabled),
			Detectors: detectorSelectionsToModel(config.BuiltInDetectors.Detectors),
		}
		if existing != nil && existing.BuiltInDetectors != nil {
			model.BuiltInDetectors.Detectors = orderDetectorSelectionsLikeExisting(model.BuiltInDetectors.Detectors, existing.BuiltInDetectors.Detectors)
		}
		if (config.BuiltInDetectors.Enabled != nil && !*config.BuiltInDetectors.Enabled) && !includeAPIOnlyConfig && (existing == nil || existing.BuiltInDetectors == nil || len(existing.BuiltInDetectors.Detectors) == 0) {
			model.BuiltInDetectors.Detectors = nil
		}
	}
	if config.CustomDetectors != nil && (includeAPIOnlyConfig || (existing != nil && existing.CustomDetectors != nil)) {
		model.CustomDetectors = &AgentMemoryCustomDetectorsModel{
			Enabled:   types.BoolPointerValue(config.CustomDetectors.Enabled),
			Detectors: customDetectorsToModel(config.CustomDetectors.Detectors),
		}
		if existing != nil && existing.CustomDetectors != nil {
			model.CustomDetectors.Detectors = orderCustomDetectorsLikeExisting(model.CustomDetectors.Detectors, existing.CustomDetectors.Detectors)
		}
		if (config.CustomDetectors.Enabled != nil && !*config.CustomDetectors.Enabled) && !includeAPIOnlyConfig && (existing == nil || existing.CustomDetectors == nil || len(existing.CustomDetectors.Detectors) == 0) {
			model.CustomDetectors.Detectors = nil
		}
	}

	return model
}

func detectorSelectionsFromModel(models []AgentMemoryDetectorSelectionModel) []agentmemoryapi.DetectorSelection {
	if len(models) == 0 {
		// An explicitly managed empty group must clear the stored selections.
		return []agentmemoryapi.DetectorSelection{}
	}
	selections := make([]agentmemoryapi.DetectorSelection, 0, len(models))
	for _, model := range models {
		selections = append(selections, agentmemoryapi.DetectorSelection{
			ID:      model.ID.ValueStringPointer(),
			Enabled: redis.Bool(model.Enabled.ValueBool()),
			Action:  stringPointerFromValue(model.Action),
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
			ID:      types.StringPointerValue(selection.ID),
			Enabled: types.BoolPointerValue(selection.Enabled),
			Action:  types.StringPointerValue(selection.Action),
		})
	}
	return models
}

func customDetectorsFromModel(models []AgentMemoryCustomDetectorModel) []agentmemoryapi.CustomDetector {
	if len(models) == 0 {
		return []agentmemoryapi.CustomDetector{}
	}
	detectors := make([]agentmemoryapi.CustomDetector, 0, len(models))
	for _, model := range models {
		detectors = append(detectors, agentmemoryapi.CustomDetector{
			Name:    model.Name.ValueStringPointer(),
			Enabled: redis.Bool(model.Enabled.ValueBool()),
			Action:  stringPointerFromValue(model.Action),
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
			Name:    types.StringPointerValue(detector.Name),
			Enabled: types.BoolPointerValue(detector.Enabled),
			Action:  types.StringPointerValue(detector.Action),
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
		Kind:  model.Kind.ValueStringPointer(),
		Regex: regexFromModel(model.Regex),
	}
}

func matcherToModel(matcher *agentmemoryapi.MatcherSpec) *AgentMemoryMatcherSpecModel {
	if matcher == nil {
		return nil
	}
	return &AgentMemoryMatcherSpecModel{
		Kind:  types.StringPointerValue(matcher.Kind),
		Regex: regexToModel(matcher.Regex),
	}
}

func regexFromModel(model *AgentMemoryRegexSpecModel) *agentmemoryapi.RegexSpec {
	if model == nil {
		return nil
	}
	return &agentmemoryapi.RegexSpec{Pattern: model.Pattern.ValueStringPointer()}
}

func regexToModel(regex *agentmemoryapi.RegexSpec) *AgentMemoryRegexSpecModel {
	if regex == nil {
		return nil
	}
	return &AgentMemoryRegexSpecModel{Pattern: types.StringPointerValue(regex.Pattern)}
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

func customMemoryTypesToModel(memoryTypes []agentmemoryapi.CustomMemoryType, existing []AgentMemoryCustomMemoryTypeModel) []AgentMemoryCustomMemoryTypeModel {
	if len(memoryTypes) == 0 {
		return nil
	}
	models := make([]AgentMemoryCustomMemoryTypeModel, 0, len(memoryTypes))
	for _, memoryType := range memoryTypes {
		models = append(models, AgentMemoryCustomMemoryTypeModel{
			Name:               types.StringPointerValue(memoryType.Name),
			Description:        types.StringPointerValue(memoryType.Description),
			Fields:             customFieldsToModel(memoryType.Fields),
			ExtractionStrategy: customExtractionStrategyToModel(memoryType.ExtractionStrategy),
		})
	}
	if len(existing) > 0 {
		models = orderCustomMemoryTypesLikeExisting(models, existing)
	}
	return models
}

func orderCustomMemoryTypesLikeExisting(models, existing []AgentMemoryCustomMemoryTypeModel) []AgentMemoryCustomMemoryTypeModel {
	modelByName := make(map[string]AgentMemoryCustomMemoryTypeModel, len(models))
	for _, model := range models {
		name, ok := knownString(model.Name)
		if !ok {
			return models
		}
		if _, exists := modelByName[name]; exists {
			return models
		}
		modelByName[name] = model
	}

	ordered := make([]AgentMemoryCustomMemoryTypeModel, 0, len(models))
	used := make(map[string]struct{}, len(models))
	for _, existingModel := range existing {
		name, ok := knownString(existingModel.Name)
		if !ok {
			continue
		}
		model, exists := modelByName[name]
		if !exists {
			continue
		}
		if customMemoryTypeDefinitionEqual(existingModel, model) {
			model.Fields = orderCustomFieldsLikeExisting(model.Fields, existingModel.Fields)
		}
		ordered = append(ordered, model)
		used[name] = struct{}{}
	}

	for _, model := range models {
		name, ok := knownString(model.Name)
		if !ok {
			return models
		}
		if _, exists := used[name]; exists {
			continue
		}
		ordered = append(ordered, model)
	}

	return ordered
}

func customFieldsFromModel(models []AgentMemoryCustomFieldModel) []agentmemoryapi.CustomField {
	if len(models) == 0 {
		return nil
	}
	fields := make([]agentmemoryapi.CustomField, 0, len(models))
	for _, model := range models {
		fields = append(fields, agentmemoryapi.CustomField{
			Name:        model.Name.ValueStringPointer(),
			Description: model.Description.ValueStringPointer(),
			Type:        model.Type.ValueStringPointer(),
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
			Name:        types.StringPointerValue(field.Name),
			Description: types.StringPointerValue(field.Description),
			Type:        types.StringPointerValue(field.Type),
		})
	}
	return models
}

func orderCustomFieldsLikeExisting(models, existing []AgentMemoryCustomFieldModel) []AgentMemoryCustomFieldModel {
	if len(models) != len(existing) {
		return models
	}

	modelByName := make(map[string]AgentMemoryCustomFieldModel, len(models))
	for _, model := range models {
		name, ok := knownString(model.Name)
		if !ok {
			return models
		}
		if _, exists := modelByName[name]; exists {
			return models
		}
		modelByName[name] = model
	}

	ordered := make([]AgentMemoryCustomFieldModel, 0, len(models))
	for _, existingModel := range existing {
		name, ok := knownString(existingModel.Name)
		if !ok {
			return models
		}
		model, exists := modelByName[name]
		if !exists {
			return models
		}
		ordered = append(ordered, model)
	}

	return ordered
}

func customExtractionStrategyFromModel(model *AgentMemoryCustomExtractionStrategy) *agentmemoryapi.CustomExtractionStrategy {
	if model == nil {
		return nil
	}
	strategy := &agentmemoryapi.CustomExtractionStrategy{
		Prompt: model.Prompt.ValueStringPointer(),
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
		Prompt: types.StringPointerValue(strategy.Prompt),
	}
	if strategy.Enabled != nil {
		model.Enabled = types.BoolValue(*strategy.Enabled)
	} else {
		model.Enabled = types.BoolNull()
	}
	return model
}

func intPointerValue(value *int) types.Int64 {
	if value == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*value))
}

// Optional computed strings may still be unknown during creation. Do not turn
// them into explicit empty strings that suppress service defaults.
func stringPointerFromValue(value types.String) *string {
	if value.IsUnknown() {
		return nil
	}
	return value.ValueStringPointer()
}
