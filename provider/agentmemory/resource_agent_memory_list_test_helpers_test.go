package agentmemory

import (
	"testing"

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
