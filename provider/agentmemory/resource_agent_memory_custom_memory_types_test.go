package agentmemory

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateCustomMemoryTypesPlanRejectsStrategyRemoval(t *testing.T) {
	enabled := types.BoolValue(true)
	state := AgentMemoryResourceModel{
		CustomMemoryTypes: []AgentMemoryCustomMemoryTypeModel{
			{
				Name:        types.StringValue("custom_name"),
				Description: types.StringValue("test"),
				Fields: []AgentMemoryCustomFieldModel{
					{
						Name:        types.StringValue("field1"),
						Type:        types.StringValue("str"),
						Description: types.StringValue("capture the field"),
					},
				},
				ExtractionStrategy: &AgentMemoryCustomExtractionStrategy{
					Enabled: enabled,
					Prompt:  types.StringValue("test"),
				},
			},
		},
	}
	plan := AgentMemoryResourceModel{
		CustomMemoryTypes: []AgentMemoryCustomMemoryTypeModel{
			{
				Name:        types.StringValue("custom_name"),
				Description: types.StringValue("test"),
				Fields: []AgentMemoryCustomFieldModel{
					{
						Name:        types.StringValue("field1"),
						Type:        types.StringValue("str"),
						Description: types.StringValue("capture the field"),
					},
				},
			},
		},
	}

	var resp resource.ModifyPlanResponse
	validateCustomMemoryTypesPlan(plan, state, &resp)

	require.True(t, resp.Diagnostics.HasError())
	require.Len(t, resp.Diagnostics.Errors(), 1)
	assert.Equal(t, "Unsupported Agent Memory custom memory type strategy removal", resp.Diagnostics.Errors()[0].Summary())
}

func TestValidateCustomMemoryTypesPlanAllowsFieldReorder(t *testing.T) {
	state := AgentMemoryResourceModel{
		CustomMemoryTypes: []AgentMemoryCustomMemoryTypeModel{
			customMemoryTypeModelWithFields([]AgentMemoryCustomFieldModel{
				customMemoryFieldModel("cloud", "str", "preferred cloud provider"),
				customMemoryFieldModel("region", "str", "preferred cloud region"),
			}),
		},
	}
	plan := AgentMemoryResourceModel{
		CustomMemoryTypes: []AgentMemoryCustomMemoryTypeModel{
			customMemoryTypeModelWithFields([]AgentMemoryCustomFieldModel{
				customMemoryFieldModel("region", "str", "preferred cloud region"),
				customMemoryFieldModel("cloud", "str", "preferred cloud provider"),
			}),
		},
	}

	var resp resource.ModifyPlanResponse
	validateCustomMemoryTypesPlan(plan, state, &resp)

	require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics.Errors())
}

func TestOrderCustomMemoryTypesLikeExistingPreservesTerraformOrder(t *testing.T) {
	apiModels := []AgentMemoryCustomMemoryTypeModel{
		customMemoryTypeModel("support_case", []AgentMemoryCustomFieldModel{
			customMemoryFieldModel("affected_region", "str", "cloud or application region affected by the issue"),
			customMemoryFieldModel("case_priority", "int", "case priority from one to five"),
		}),
		customMemoryTypeModel("customer_profile", []AgentMemoryCustomFieldModel{
			customMemoryFieldModel("region", "str", "preferred cloud region"),
			customMemoryFieldModel("cloud", "str", "preferred cloud provider"),
		}),
	}
	existing := []AgentMemoryCustomMemoryTypeModel{
		customMemoryTypeModel("customer_profile", []AgentMemoryCustomFieldModel{
			customMemoryFieldModel("cloud", "str", "preferred cloud provider"),
			customMemoryFieldModel("region", "str", "preferred cloud region"),
		}),
		customMemoryTypeModel("support_case", []AgentMemoryCustomFieldModel{
			customMemoryFieldModel("case_priority", "int", "case priority from one to five"),
			customMemoryFieldModel("affected_region", "str", "cloud or application region affected by the issue"),
		}),
	}

	ordered := orderCustomMemoryTypesLikeExisting(apiModels, existing)

	require.Len(t, ordered, 2)
	assert.Equal(t, "customer_profile", ordered[0].Name.ValueString())
	assert.Equal(t, "cloud", ordered[0].Fields[0].Name.ValueString())
	assert.Equal(t, "region", ordered[0].Fields[1].Name.ValueString())
	assert.Equal(t, "support_case", ordered[1].Name.ValueString())
	assert.Equal(t, "case_priority", ordered[1].Fields[0].Name.ValueString())
	assert.Equal(t, "affected_region", ordered[1].Fields[1].Name.ValueString())
}

func customMemoryTypeModel(name string, fields []AgentMemoryCustomFieldModel) AgentMemoryCustomMemoryTypeModel {
	model := customMemoryTypeModelWithFields(fields)
	model.Name = types.StringValue(name)
	return model
}

func customMemoryTypeModelWithFields(fields []AgentMemoryCustomFieldModel) AgentMemoryCustomMemoryTypeModel {
	return AgentMemoryCustomMemoryTypeModel{
		Name:        types.StringValue("custom_name"),
		Description: types.StringValue("test"),
		Fields:      fields,
	}
}

func customMemoryFieldModel(name, fieldType, description string) AgentMemoryCustomFieldModel {
	return AgentMemoryCustomFieldModel{
		Name:        types.StringValue(name),
		Type:        types.StringValue(fieldType),
		Description: types.StringValue(description),
	}
}
