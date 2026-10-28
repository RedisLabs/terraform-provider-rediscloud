package agentmemory

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateCustomMemoryTypesPlanReplacesStrategyRemoval(t *testing.T) {
	enabled := types.BoolValue(true)
	state := AgentMemoryResourceModel{
		CustomMemoryTypes: mustCustomMemoryTypeList(t, []AgentMemoryCustomMemoryTypeModel{
			{
				Name:        types.StringValue("custom_name"),
				Description: types.StringValue("test"),
				Fields: mustCustomFieldList(t, []AgentMemoryCustomFieldModel{
					{
						Name:        types.StringValue("field1"),
						Type:        types.StringValue("str"),
						Description: types.StringValue("capture the field"),
					},
				}),
				ExtractionStrategy: mustCustomExtractionStrategyList(t, &AgentMemoryCustomExtractionStrategy{
					Enabled: enabled,
					Prompt:  types.StringValue("test"),
				}),
			},
		}),
	}
	plan := AgentMemoryResourceModel{
		CustomMemoryTypes: mustCustomMemoryTypeList(t, []AgentMemoryCustomMemoryTypeModel{
			{
				Name:        types.StringValue("custom_name"),
				Description: types.StringValue("test"),
				Fields: mustCustomFieldList(t, []AgentMemoryCustomFieldModel{
					{
						Name:        types.StringValue("field1"),
						Type:        types.StringValue("str"),
						Description: types.StringValue("capture the field"),
					},
				}),
			},
		}),
	}

	var resp resource.ModifyPlanResponse
	validateCustomMemoryTypesPlan(t.Context(), plan, state, &resp)

	require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics.Errors())
	require.Len(t, resp.RequiresReplace, 1)
	assert.Equal(t, "custom_memory_types[0].extraction_strategy", resp.RequiresReplace[0].String())
}

func TestValidateCustomMemoryTypesPlanAllowsFieldReorder(t *testing.T) {
	state := AgentMemoryResourceModel{
		CustomMemoryTypes: mustCustomMemoryTypeList(t, []AgentMemoryCustomMemoryTypeModel{
			customMemoryTypeModelWithFields(t, []AgentMemoryCustomFieldModel{
				customMemoryFieldModel("cloud", "str", "preferred cloud provider"),
				customMemoryFieldModel("region", "str", "preferred cloud region"),
			}),
		}),
	}
	plan := AgentMemoryResourceModel{
		CustomMemoryTypes: mustCustomMemoryTypeList(t, []AgentMemoryCustomMemoryTypeModel{
			customMemoryTypeModelWithFields(t, []AgentMemoryCustomFieldModel{
				customMemoryFieldModel("region", "str", "preferred cloud region"),
				customMemoryFieldModel("cloud", "str", "preferred cloud provider"),
			}),
		}),
	}

	var resp resource.ModifyPlanResponse
	validateCustomMemoryTypesPlan(t.Context(), plan, state, &resp)

	require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics.Errors())
	require.Empty(t, resp.RequiresReplace)
}

func TestOrderCustomMemoryTypesLikeExistingPreservesTerraformOrder(t *testing.T) {
	apiModels := []AgentMemoryCustomMemoryTypeModel{
		customMemoryTypeModel(t, "support_case", []AgentMemoryCustomFieldModel{
			customMemoryFieldModel("affected_region", "str", "cloud or application region affected by the issue"),
			customMemoryFieldModel("case_priority", "int", "case priority from one to five"),
		}),
		customMemoryTypeModel(t, "customer_profile", []AgentMemoryCustomFieldModel{
			customMemoryFieldModel("region", "str", "preferred cloud region"),
			customMemoryFieldModel("cloud", "str", "preferred cloud provider"),
		}),
	}
	existing := []AgentMemoryCustomMemoryTypeModel{
		customMemoryTypeModel(t, "customer_profile", []AgentMemoryCustomFieldModel{
			customMemoryFieldModel("cloud", "str", "preferred cloud provider"),
			customMemoryFieldModel("region", "str", "preferred cloud region"),
		}),
		customMemoryTypeModel(t, "support_case", []AgentMemoryCustomFieldModel{
			customMemoryFieldModel("case_priority", "int", "case priority from one to five"),
			customMemoryFieldModel("affected_region", "str", "cloud or application region affected by the issue"),
		}),
	}

	ordered := orderCustomMemoryTypesLikeExisting(t.Context(), apiModels, existing, testAgentMemoryDiags(t))

	require.Len(t, ordered, 2)
	assert.Equal(t, "customer_profile", ordered[0].Name.ValueString())
	assert.Equal(t, "cloud", mustCustomFieldModels(t, ordered[0].Fields)[0].Name.ValueString())
	assert.Equal(t, "region", mustCustomFieldModels(t, ordered[0].Fields)[1].Name.ValueString())
	assert.Equal(t, "support_case", ordered[1].Name.ValueString())
	assert.Equal(t, "case_priority", mustCustomFieldModels(t, ordered[1].Fields)[0].Name.ValueString())
	assert.Equal(t, "affected_region", mustCustomFieldModels(t, ordered[1].Fields)[1].Name.ValueString())
}

func customMemoryTypeModel(t *testing.T, name string, fields []AgentMemoryCustomFieldModel) AgentMemoryCustomMemoryTypeModel {
	model := customMemoryTypeModelWithFields(t, fields)
	model.Name = types.StringValue(name)
	return model
}

func customMemoryTypeModelWithFields(t *testing.T, fields []AgentMemoryCustomFieldModel) AgentMemoryCustomMemoryTypeModel {
	return AgentMemoryCustomMemoryTypeModel{
		Name:        types.StringValue("custom_name"),
		Description: types.StringValue("test"),
		Fields:      mustCustomFieldList(t, fields),
	}
}

func customMemoryFieldModel(name, fieldType, description string) AgentMemoryCustomFieldModel {
	return AgentMemoryCustomFieldModel{
		Name:        types.StringValue(name),
		Type:        types.StringValue(fieldType),
		Description: types.StringValue(description),
	}
}

func TestOrderCustomTypeAndFieldEmptyNames(t *testing.T) {
	knownField := customMemoryFieldModel("known", "str", "Known field")
	emptyField := customMemoryFieldModel("", "str", "Empty field")
	otherType := customMemoryTypeModel(t, "other", []AgentMemoryCustomFieldModel{knownField})
	apiModels := []AgentMemoryCustomMemoryTypeModel{
		otherType,
		customMemoryTypeModel(t, "", []AgentMemoryCustomFieldModel{knownField, emptyField}),
	}
	existing := []AgentMemoryCustomMemoryTypeModel{
		customMemoryTypeModel(t, "", []AgentMemoryCustomFieldModel{emptyField, knownField}),
		otherType,
	}
	ordered := orderCustomMemoryTypesLikeExisting(t.Context(), apiModels, existing, testAgentMemoryDiags(t))
	require.Equal(t, existing, ordered)
}
