package agentmemory

import (
	"context"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	agentmemoryapi "github.com/RedisLabs/rediscloud-go-api/service/agentmemory"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func modelConfigFromModel(ctx context.Context, model *AgentMemoryModelConfigModel, diags *diag.Diagnostics) *agentmemoryapi.ModelConfig {
	if model == nil {
		return nil
	}
	config := &agentmemoryapi.ModelConfig{Provider: stringPointerFromValue(model.Provider), Model: stringPointerFromValue(model.Model)}
	if credentials := modelCredentialsBlock(ctx, model.Credentials, diags); credentials != nil {
		config.Credentials = &agentmemoryapi.ModelCredentials{Type: stringPointerFromValue(credentials.Type)}
	}
	return config
}

func modelConfigToModel(ctx context.Context, status *agentmemoryapi.ModelStatus, existing *AgentMemoryModelConfigModel, diags *diag.Diagnostics) *AgentMemoryModelConfigModel {
	if status == nil {
		return existing
	}
	// The API does not return credentials. Keep only their type and rotation version.
	return &AgentMemoryModelConfigModel{Provider: types.StringPointerValue(status.Provider), Model: types.StringPointerValue(status.Model), Credentials: modelCredentialsFromExisting(ctx, existing, diags)}
}

func modelCredentialsFromExisting(ctx context.Context, existing *AgentMemoryModelConfigModel, diags *diag.Diagnostics) ModelCredentialsListValue {
	var model *AgentMemoryModelCredentialsModel
	if existing != nil {
		if credentials := modelCredentialsBlock(ctx, existing.Credentials, diags); credentials != nil {
			model = &AgentMemoryModelCredentialsModel{Type: credentials.Type, APIKeyVersion: credentials.APIKeyVersion}
		}
	}
	return modelCredentialsList(ctx, model, diags)
}

func modelConfigEqual(ctx context.Context, left, right ModelConfigListValue, diags *diag.Diagnostics) bool {
	if left.IsUnknown() || right.IsUnknown() {
		return false
	}
	a, b := modelConfigBlock(ctx, left, diags), modelConfigBlock(ctx, right, diags)
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Provider.Equal(b.Provider) && a.Model.Equal(b.Model) && modelCredentialsEqual(ctx, a.Credentials, b.Credentials, diags)
}

func modelCredentialsEqual(ctx context.Context, left, right ModelCredentialsListValue, diags *diag.Diagnostics) bool {
	if left.IsUnknown() || right.IsUnknown() {
		return false
	}
	a, b := modelCredentialsBlock(ctx, left, diags), modelCredentialsBlock(ctx, right, diags)
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Type.Equal(b.Type) && a.APIKeyVersion.Equal(b.APIKeyVersion)
}

func summarizationFromModel(ctx context.Context, model *AgentMemorySummarizationModel, diags *diag.Diagnostics) *agentmemoryapi.SummarizationConfig {
	if model == nil {
		return nil
	}
	config := &agentmemoryapi.SummarizationConfig{Enabled: redis.Bool(model.Enabled.ValueBool())}
	if !model.TriggerStrategy.IsNull() && !model.TriggerStrategy.IsUnknown() {
		config.TriggerStrategy = model.TriggerStrategy.ValueStringPointer()
	}
	if event := eventCountBlock(ctx, model.EventCount, diags); event != nil {
		config.EventCount = &agentmemoryapi.SummarizationThresholdConfig{Threshold: redis.Int(int(event.Threshold.ValueInt64())), RetainCount: redis.Int(int(event.RetainCount.ValueInt64()))}
	}
	return config
}

func summarizationToModel(ctx context.Context, config *agentmemoryapi.SummarizationConfig, existing *AgentMemorySummarizationModel, includeAPIOnlyConfig bool, diags *diag.Diagnostics) *AgentMemorySummarizationModel {
	if config == nil {
		return nil
	}
	model := &AgentMemorySummarizationModel{Enabled: types.BoolPointerValue(config.Enabled), TriggerStrategy: types.StringPointerValue(config.TriggerStrategy)}
	if config.EventCount != nil && (includeAPIOnlyConfig || (existing != nil && len(existing.EventCount.Elements()) > 0)) {
		model.EventCount = eventCountList(ctx, &AgentMemoryEventCountModel{Threshold: intPointerValue(config.EventCount.Threshold), RetainCount: intPointerValue(config.EventCount.RetainCount)}, diags)
	}
	// Reads must not add API-only children omitted from managed configuration.
	if config.Enabled != nil && !*config.Enabled && !includeAPIOnlyConfig && (existing == nil || existing.TriggerStrategy.IsNull() || existing.TriggerStrategy.IsUnknown()) {
		model.TriggerStrategy = types.StringNull()
	}
	return model
}

func exclusionsFromModel(ctx context.Context, model *AgentMemoryExclusionsModel, diags *diag.Diagnostics) *agentmemoryapi.LongTermMemoryExclusions {
	if model == nil {
		return nil
	}
	config := &agentmemoryapi.LongTermMemoryExclusions{Enabled: redis.Bool(model.Enabled.ValueBool())}
	if semantic := semanticExclusionBlock(ctx, model.Semantic, diags); semantic != nil {
		config.Semantic = &agentmemoryapi.SemanticExclusion{Enabled: redis.Bool(semantic.Enabled.ValueBool()), Prompt: stringPointerFromValue(semantic.Prompt)}
	}
	if builtIn := builtInDetectorsBlock(ctx, model.BuiltInDetectors, diags); builtIn != nil {
		config.BuiltInDetectors = &agentmemoryapi.BuiltInDetectorsConfig{Enabled: redis.Bool(builtIn.Enabled.ValueBool()), Detectors: detectorSelectionsFromModel(detectorSelectionModels(ctx, builtIn.Detectors, diags))}
	}
	if custom := customDetectorsBlock(ctx, model.CustomDetectors, diags); custom != nil {
		config.CustomDetectors = &agentmemoryapi.CustomDetectorsConfig{Enabled: redis.Bool(custom.Enabled.ValueBool()), Detectors: customDetectorsFromModel(ctx, customDetectorModels(ctx, custom.Detectors, diags), diags)}
	}
	return config
}

func exclusionsToModel(ctx context.Context, config *agentmemoryapi.LongTermMemoryExclusions, existing *AgentMemoryExclusionsModel, includeAPIOnlyConfig bool, diags *diag.Diagnostics) *AgentMemoryExclusionsModel {
	if config == nil {
		return nil
	}
	model := &AgentMemoryExclusionsModel{Enabled: types.BoolPointerValue(config.Enabled)}
	var semantic *AgentMemorySemanticExclusionModel
	var builtIn *AgentMemoryBuiltInDetectorsModel
	var custom *AgentMemoryCustomDetectorsModel
	if existing != nil {
		semantic = semanticExclusionBlock(ctx, existing.Semantic, diags)
		builtIn = builtInDetectorsBlock(ctx, existing.BuiltInDetectors, diags)
		custom = customDetectorsBlock(ctx, existing.CustomDetectors, diags)
	}
	if config.Semantic != nil && (includeAPIOnlyConfig || semantic != nil) {
		m := &AgentMemorySemanticExclusionModel{Enabled: types.BoolPointerValue(config.Semantic.Enabled), Prompt: types.StringPointerValue(config.Semantic.Prompt)}
		if config.Semantic.Enabled != nil && !*config.Semantic.Enabled && !includeAPIOnlyConfig && (semantic == nil || semantic.Prompt.IsNull() || semantic.Prompt.IsUnknown()) {
			m.Prompt = types.StringNull()
		}
		model.Semantic = semanticExclusionList(ctx, m, diags)
	}
	if config.BuiltInDetectors != nil && (includeAPIOnlyConfig || builtIn != nil) {
		m := &AgentMemoryBuiltInDetectorsModel{Enabled: types.BoolPointerValue(config.BuiltInDetectors.Enabled), Detectors: detectorSelectionListFromAPI(ctx, config.BuiltInDetectors.Detectors, builtIn, diags)}
		if config.BuiltInDetectors.Enabled != nil && !*config.BuiltInDetectors.Enabled && !includeAPIOnlyConfig && (builtIn == nil || len(builtIn.Detectors.Elements()) == 0) {
			m.Detectors = DetectorSelectionListValue{ListValue: types.ListNull(detectorSelectionElementType)}
		}
		model.BuiltInDetectors = builtInDetectorsList(ctx, m, diags)
	}
	if config.CustomDetectors != nil && (includeAPIOnlyConfig || custom != nil) {
		m := &AgentMemoryCustomDetectorsModel{Enabled: types.BoolPointerValue(config.CustomDetectors.Enabled), Detectors: customDetectorListFromAPI(ctx, config.CustomDetectors.Detectors, custom, diags)}
		if config.CustomDetectors.Enabled != nil && !*config.CustomDetectors.Enabled && !includeAPIOnlyConfig && (custom == nil || len(custom.Detectors.Elements()) == 0) {
			m.Detectors = CustomDetectorListValue{ListValue: types.ListNull(customDetectorElementType)}
		}
		model.CustomDetectors = customDetectorsList(ctx, m, diags)
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

func customDetectorsFromModel(ctx context.Context, models []AgentMemoryCustomDetectorModel, diags *diag.Diagnostics) []agentmemoryapi.CustomDetector {
	if len(models) == 0 {
		return []agentmemoryapi.CustomDetector{}
	}
	detectors := make([]agentmemoryapi.CustomDetector, 0, len(models))
	for _, model := range models {
		detectors = append(detectors, agentmemoryapi.CustomDetector{
			Name:    model.Name.ValueStringPointer(),
			Enabled: redis.Bool(model.Enabled.ValueBool()),
			Action:  stringPointerFromValue(model.Action),
			Matcher: matcherFromModel(ctx, matcherSpecBlock(ctx, model.Matcher, diags), diags),
		})
	}
	return detectors
}

func customDetectorsToModel(ctx context.Context, detectors []agentmemoryapi.CustomDetector, diags *diag.Diagnostics) []AgentMemoryCustomDetectorModel {
	if len(detectors) == 0 {
		return nil
	}
	models := make([]AgentMemoryCustomDetectorModel, 0, len(detectors))
	for _, detector := range detectors {
		models = append(models, AgentMemoryCustomDetectorModel{
			Name:    types.StringPointerValue(detector.Name),
			Enabled: types.BoolPointerValue(detector.Enabled),
			Action:  types.StringPointerValue(detector.Action),
			Matcher: matcherSpecList(ctx, matcherToModel(ctx, detector.Matcher, diags), diags),
		})
	}
	return models
}

func matcherFromModel(ctx context.Context, model *AgentMemoryMatcherSpecModel, diags *diag.Diagnostics) *agentmemoryapi.MatcherSpec {
	if model == nil {
		return nil
	}
	return &agentmemoryapi.MatcherSpec{
		Kind:  model.Kind.ValueStringPointer(),
		Regex: regexFromModel(regexSpecBlock(ctx, model.Regex, diags)),
	}
}

func matcherToModel(ctx context.Context, matcher *agentmemoryapi.MatcherSpec, diags *diag.Diagnostics) *AgentMemoryMatcherSpecModel {
	if matcher == nil {
		return nil
	}
	return &AgentMemoryMatcherSpecModel{
		Kind:  types.StringPointerValue(matcher.Kind),
		Regex: regexSpecList(ctx, regexToModel(matcher.Regex), diags),
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

func customMemoryTypesFromModel(ctx context.Context, list CustomMemoryTypeListValue, diags *diag.Diagnostics) []agentmemoryapi.CustomMemoryType {
	models := customMemoryTypeModels(ctx, list, diags)
	if len(models) == 0 {
		return nil
	}
	types := make([]agentmemoryapi.CustomMemoryType, 0, len(models))
	for _, model := range models {
		types = append(types, customMemoryTypeFromModel(ctx, model, diags))
	}
	return types
}

func customMemoryTypesToModel(ctx context.Context, memoryTypes []agentmemoryapi.CustomMemoryType, existing []AgentMemoryCustomMemoryTypeModel, diags *diag.Diagnostics) []AgentMemoryCustomMemoryTypeModel {
	if len(memoryTypes) == 0 {
		return nil
	}
	models := make([]AgentMemoryCustomMemoryTypeModel, 0, len(memoryTypes))
	for _, memoryType := range memoryTypes {
		models = append(models, AgentMemoryCustomMemoryTypeModel{
			Name:               types.StringPointerValue(memoryType.Name),
			Description:        types.StringPointerValue(memoryType.Description),
			Fields:             customFieldListFromAPI(ctx, memoryType.Fields, diags),
			ExtractionStrategy: customExtractionStrategyList(ctx, customExtractionStrategyToModel(memoryType.ExtractionStrategy), diags),
		})
	}
	if len(existing) > 0 {
		models = orderCustomMemoryTypesLikeExisting(ctx, models, existing, diags)
	}
	return models
}

func orderCustomMemoryTypesLikeExisting(ctx context.Context, models, existing []AgentMemoryCustomMemoryTypeModel, diags *diag.Diagnostics) []AgentMemoryCustomMemoryTypeModel {
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
		if customMemoryTypeDefinitionEqual(ctx, existingModel, model, diags) {
			fields := orderCustomFieldsLikeExisting(customFieldModels(ctx, model.Fields, diags), customFieldModels(ctx, existingModel.Fields, diags))
			model.Fields = customFieldListFromModels(ctx, fields, diags)
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

func customFieldsFromModel(ctx context.Context, list CustomFieldListValue, diags *diag.Diagnostics) []agentmemoryapi.CustomField {
	models := customFieldModels(ctx, list, diags)
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

func customFieldListFromModels(ctx context.Context, models []AgentMemoryCustomFieldModel, diags *diag.Diagnostics) CustomFieldListValue {
	list, listDiags := NewCustomFieldList(ctx, models)
	diags.Append(listDiags...)
	return list
}

func customFieldListFromAPI(ctx context.Context, fields []agentmemoryapi.CustomField, diags *diag.Diagnostics) CustomFieldListValue {
	return customFieldListFromModels(ctx, customFieldsToModel(fields), diags)
}

func detectorSelectionListFromAPI(ctx context.Context, selections []agentmemoryapi.DetectorSelection, existing *AgentMemoryBuiltInDetectorsModel, diags *diag.Diagnostics) DetectorSelectionListValue {
	models := detectorSelectionsToModel(selections)
	if existing != nil {
		models = orderDetectorSelectionsLikeExisting(models, detectorSelectionModels(ctx, existing.Detectors, diags))
	}
	list, listDiags := NewDetectorSelectionList(ctx, models)
	diags.Append(listDiags...)
	return list
}

func customDetectorListFromAPI(ctx context.Context, detectors []agentmemoryapi.CustomDetector, existing *AgentMemoryCustomDetectorsModel, diags *diag.Diagnostics) CustomDetectorListValue {
	models := customDetectorsToModel(ctx, detectors, diags)
	if existing != nil {
		models = orderCustomDetectorsLikeExisting(models, customDetectorModels(ctx, existing.Detectors, diags))
	}
	list, listDiags := NewCustomDetectorList(ctx, models)
	diags.Append(listDiags...)
	return list
}
