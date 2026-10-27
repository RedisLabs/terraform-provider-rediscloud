package agentmemory

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/stretchr/testify/require"
)

func testAgentMemoryDiags(t *testing.T) *diag.Diagnostics {
	t.Helper()
	diags := new(diag.Diagnostics)
	t.Cleanup(func() { require.False(t, diags.HasError(), diags.Errors()) })
	return diags
}

func mustCustomMemoryTypeList(t *testing.T, models []AgentMemoryCustomMemoryTypeModel) CustomMemoryTypeListValue {
	t.Helper()
	list, diags := NewCustomMemoryTypeList(t.Context(), models)
	require.False(t, diags.HasError(), diags.Errors())
	return list
}

func mustCustomFieldList(t *testing.T, models []AgentMemoryCustomFieldModel) CustomFieldListValue {
	t.Helper()
	list, diags := NewCustomFieldList(t.Context(), models)
	require.False(t, diags.HasError(), diags.Errors())
	return list
}

func mustCustomFieldModels(t *testing.T, list CustomFieldListValue) []AgentMemoryCustomFieldModel {
	t.Helper()
	if list.IsNull() {
		return nil
	}
	models, diags := list.AsModels(t.Context())
	require.False(t, diags.HasError(), diags.Errors())
	return models
}

func mustCustomDetectorList(t *testing.T, models []AgentMemoryCustomDetectorModel) CustomDetectorListValue {
	t.Helper()
	list, diags := NewCustomDetectorList(t.Context(), models)
	require.False(t, diags.HasError(), diags.Errors())
	return list
}

func mustCustomDetectorModels(t *testing.T, list CustomDetectorListValue) []AgentMemoryCustomDetectorModel {
	t.Helper()
	if list.IsNull() {
		return nil
	}
	models, diags := list.AsModels(t.Context())
	require.False(t, diags.HasError(), diags.Errors())
	return models
}

func mustDetectorSelectionList(t *testing.T, models []AgentMemoryDetectorSelectionModel) DetectorSelectionListValue {
	t.Helper()
	list, diags := NewDetectorSelectionList(t.Context(), models)
	require.False(t, diags.HasError(), diags.Errors())
	return list
}

func mustDetectorSelectionModels(t *testing.T, list DetectorSelectionListValue) []AgentMemoryDetectorSelectionModel {
	t.Helper()
	if list.IsNull() {
		return nil
	}
	models, diags := list.AsModels(t.Context())
	require.False(t, diags.HasError(), diags.Errors())
	return models
}

func testMemoryTimeoutsList(t *testing.T, value timeouts.Value) MemoryTimeoutsListValue {
	t.Helper()
	var models []timeouts.Value
	if !value.IsNull() {
		models = []timeouts.Value{value}
	}
	list, diags := NewMemoryTimeoutsList(t.Context(), models)
	require.False(t, diags.HasError(), diags.Errors())
	return list
}
func testApiKeyTimeoutsList(t *testing.T, value timeouts.Value) APIKeyTimeoutsListValue {
	t.Helper()
	var models []timeouts.Value
	if !value.IsNull() {
		models = []timeouts.Value{value}
	}
	list, diags := NewAPIKeyTimeoutsList(t.Context(), models)
	require.False(t, diags.HasError(), diags.Errors())
	return list
}

func mustModelConfig(t *testing.T, list ModelConfigListValue) *AgentMemoryModelConfigModel {
	t.Helper()
	return modelConfigBlock(t.Context(), list, testAgentMemoryDiags(t))
}
func mustModelConfigList(t *testing.T, model *AgentMemoryModelConfigModel) ModelConfigListValue {
	t.Helper()
	return modelConfigList(t.Context(), model, testAgentMemoryDiags(t))
}

func mustModelCredentials(t *testing.T, list ModelCredentialsListValue) *AgentMemoryModelCredentialsModel {
	t.Helper()
	return modelCredentialsBlock(t.Context(), list, testAgentMemoryDiags(t))
}
func mustModelCredentialsList(t *testing.T, model *AgentMemoryModelCredentialsModel) ModelCredentialsListValue {
	t.Helper()
	return modelCredentialsList(t.Context(), model, testAgentMemoryDiags(t))
}

func mustSummarization(t *testing.T, list SummarizationListValue) *AgentMemorySummarizationModel {
	t.Helper()
	return summarizationBlock(t.Context(), list, testAgentMemoryDiags(t))
}
func mustSummarizationList(t *testing.T, model *AgentMemorySummarizationModel) SummarizationListValue {
	t.Helper()
	return summarizationList(t.Context(), model, testAgentMemoryDiags(t))
}

func mustEventCount(t *testing.T, list EventCountListValue) *AgentMemoryEventCountModel {
	t.Helper()
	return eventCountBlock(t.Context(), list, testAgentMemoryDiags(t))
}
func mustEventCountList(t *testing.T, model *AgentMemoryEventCountModel) EventCountListValue {
	t.Helper()
	return eventCountList(t.Context(), model, testAgentMemoryDiags(t))
}

func mustExclusions(t *testing.T, list ExclusionsListValue) *AgentMemoryExclusionsModel {
	t.Helper()
	return exclusionsBlock(t.Context(), list, testAgentMemoryDiags(t))
}
func mustExclusionsList(t *testing.T, model *AgentMemoryExclusionsModel) ExclusionsListValue {
	t.Helper()
	return exclusionsList(t.Context(), model, testAgentMemoryDiags(t))
}

func mustSemanticExclusion(t *testing.T, list SemanticExclusionListValue) *AgentMemorySemanticExclusionModel {
	t.Helper()
	return semanticExclusionBlock(t.Context(), list, testAgentMemoryDiags(t))
}
func mustSemanticExclusionList(t *testing.T, model *AgentMemorySemanticExclusionModel) SemanticExclusionListValue {
	t.Helper()
	return semanticExclusionList(t.Context(), model, testAgentMemoryDiags(t))
}

func mustBuiltInDetectors(t *testing.T, list BuiltInDetectorsListValue) *AgentMemoryBuiltInDetectorsModel {
	t.Helper()
	return builtInDetectorsBlock(t.Context(), list, testAgentMemoryDiags(t))
}
func mustBuiltInDetectorsList(t *testing.T, model *AgentMemoryBuiltInDetectorsModel) BuiltInDetectorsListValue {
	t.Helper()
	return builtInDetectorsList(t.Context(), model, testAgentMemoryDiags(t))
}

func mustCustomDetectors(t *testing.T, list CustomDetectorsListValue) *AgentMemoryCustomDetectorsModel {
	t.Helper()
	return customDetectorsBlock(t.Context(), list, testAgentMemoryDiags(t))
}
func mustCustomDetectorsList(t *testing.T, model *AgentMemoryCustomDetectorsModel) CustomDetectorsListValue {
	t.Helper()
	return customDetectorsList(t.Context(), model, testAgentMemoryDiags(t))
}

func mustMatcherSpec(t *testing.T, list MatcherSpecListValue) *AgentMemoryMatcherSpecModel {
	t.Helper()
	return matcherSpecBlock(t.Context(), list, testAgentMemoryDiags(t))
}
func mustMatcherSpecList(t *testing.T, model *AgentMemoryMatcherSpecModel) MatcherSpecListValue {
	t.Helper()
	return matcherSpecList(t.Context(), model, testAgentMemoryDiags(t))
}

func mustRegexSpec(t *testing.T, list RegexSpecListValue) *AgentMemoryRegexSpecModel {
	t.Helper()
	return regexSpecBlock(t.Context(), list, testAgentMemoryDiags(t))
}
func mustRegexSpecList(t *testing.T, model *AgentMemoryRegexSpecModel) RegexSpecListValue {
	t.Helper()
	return regexSpecList(t.Context(), model, testAgentMemoryDiags(t))
}

func mustCustomExtractionStrategyList(t *testing.T, model *AgentMemoryCustomExtractionStrategy) CustomExtractionStrategyListValue {
	t.Helper()
	return customExtractionStrategyList(t.Context(), model, testAgentMemoryDiags(t))
}
