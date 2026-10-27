package unit_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/agentmemory"
)

func TestAgentMemoryListTypesPreserveValues(t *testing.T) {
	for _, typ := range []basetypes.ListTypable{
		agentmemory.NewCustomMemoryTypeListType(),
		agentmemory.NewCustomFieldListType(),
		agentmemory.NewDetectorSelectionListType(),
		agentmemory.NewCustomDetectorListType(),
	} {
		t.Run(typ.String(), func(t *testing.T) {
			ctx := t.Context()
			assert.True(t, typ.Equal(typ))
			assert.False(t, typ.Equal(types.ListType{ElemType: typ.(attr.TypeWithElementType).ElementType()}))
			for _, tc := range []struct {
				name  string
				value any
			}{
				{"null", nil}, {"unknown", tftypes.UnknownValue}, {"empty", []tftypes.Value{}},
			} {
				t.Run(tc.name, func(t *testing.T) {
					raw := tftypes.NewValue(typ.TerraformType(ctx), tc.value)
					value, err := typ.ValueFromTerraform(ctx, raw)
					require.NoError(t, err)
					assert.True(t, value.Type(ctx).Equal(typ))
					assert.Equal(t, tc.name == "null", value.IsNull())
					assert.Equal(t, tc.name == "unknown", value.IsUnknown())
					roundTrip, err := value.ToTerraformValue(ctx)
					require.NoError(t, err)
					assert.True(t, raw.Equal(roundTrip))
					list := value.(basetypes.ListValuable)
					base, diags := list.ToListValue(ctx)
					require.False(t, diags.HasError(), diags.Errors())
					converted, diags := typ.ValueFromList(ctx, base)
					require.False(t, diags.HasError(), diags.Errors())
					assert.True(t, value.Equal(converted))
				})
			}
		})
	}
}

func TestAgentMemoryListTypesDecodeModels(t *testing.T) {
	ctx := t.Context()
	fields, diags := agentmemory.NewCustomFieldList(ctx, []agentmemory.AgentMemoryCustomFieldModel{{Name: types.StringValue("topic"), Description: types.StringValue("Preference topic"), Type: types.StringValue("str")}})
	require.False(t, diags.HasError(), diags.Errors())
	fieldModels, diags := fields.AsModels(ctx)
	require.False(t, diags.HasError(), diags.Errors())
	require.Len(t, fieldModels, 1)
	assert.Equal(t, "topic", fieldModels[0].Name.ValueString())
	memoryTypes, diags := agentmemory.NewCustomMemoryTypeList(ctx, []agentmemory.AgentMemoryCustomMemoryTypeModel{{Name: types.StringValue("preference"), Description: types.StringValue("Preferences"), Fields: fields}})
	require.False(t, diags.HasError(), diags.Errors())
	models, diags := memoryTypes.AsModels(ctx)
	require.False(t, diags.HasError(), diags.Errors())
	require.Len(t, models, 1)
	assert.True(t, fields.Equal(models[0].Fields))
	selections, diags := agentmemory.NewDetectorSelectionList(ctx, []agentmemory.AgentMemoryDetectorSelectionModel{{ID: types.StringValue("email"), Enabled: types.BoolValue(true), Action: types.StringValue("redact")}})
	require.False(t, diags.HasError(), diags.Errors())
	selectionModels, diags := selections.AsModels(ctx)
	require.False(t, diags.HasError(), diags.Errors())
	require.Len(t, selectionModels, 1)
	assert.Equal(t, "email", selectionModels[0].ID.ValueString())
	detectors, diags := agentmemory.NewCustomDetectorList(ctx, []agentmemory.AgentMemoryCustomDetectorModel{{Name: types.StringValue("ticket"), Enabled: types.BoolValue(true), Matcher: &agentmemory.AgentMemoryMatcherSpecModel{Kind: types.StringValue("regex"), Regex: &agentmemory.AgentMemoryRegexSpecModel{Pattern: types.StringValue("ticket-[0-9]+")}}}})
	require.False(t, diags.HasError(), diags.Errors())
	detectorModels, diags := detectors.AsModels(ctx)
	require.False(t, diags.HasError(), diags.Errors())
	require.Len(t, detectorModels, 1)
	assert.Equal(t, "ticket-[0-9]+", detectorModels[0].Matcher.Regex.Pattern.ValueString())
}

func TestAgentMemoryListTypesMatchNestedSchemas(t *testing.T) {
	var response resource.SchemaResponse
	agentmemory.NewAgentMemoryResource().Schema(t.Context(), resource.SchemaRequest{}, &response)
	require.False(t, response.Diagnostics.HasError(), response.Diagnostics.Errors())
	memoryTypes := response.Schema.Blocks["custom_memory_types"].(schema.ListNestedBlock)
	fields := memoryTypes.NestedObject.Blocks["fields"].(schema.ListNestedBlock)
	exclusions := response.Schema.Blocks["long_term_memory_exclusions"].(schema.SingleNestedBlock)
	builtIn := exclusions.Blocks["built_in_detectors"].(schema.SingleNestedBlock).Blocks["detectors"].(schema.ListNestedBlock)
	custom := exclusions.Blocks["custom_detectors"].(schema.SingleNestedBlock).Blocks["detectors"].(schema.ListNestedBlock)
	for _, block := range []schema.ListNestedBlock{memoryTypes, fields, builtIn, custom} {
		typ := block.CustomType
		assert.True(t, typ.(attr.TypeWithElementType).ElementType().Equal(block.NestedObject.Type()), typ.String())
	}
}

func TestAgentMemoryUnknownListsDecodePlan(t *testing.T) {
	var response resource.SchemaResponse
	ctx := t.Context()
	agentmemory.NewAgentMemoryResource().Schema(ctx, resource.SchemaRequest{}, &response)
	require.False(t, response.Diagnostics.HasError(), response.Diagnostics.Errors())
	typ := response.Schema.Type().TerraformType(ctx)
	for _, path := range [][]string{
		{"custom_memory_types"},
		{"custom_memory_types", "fields"},
		{"long_term_memory_exclusions", "built_in_detectors", "detectors"},
		{"long_term_memory_exclusions", "custom_detectors", "detectors"},
	} {
		t.Run(path[len(path)-1]+"/"+path[0], func(t *testing.T) {
			for _, tc := range []struct {
				name  string
				value any
			}{
				{"null", nil}, {"unknown", tftypes.UnknownValue}, {"empty", []tftypes.Value{}},
			} {
				t.Run(tc.name, func(t *testing.T) {
					raw := listValueAtPath(typ, path, tc.value)
					plan := tfsdk.Plan{Schema: response.Schema, Raw: raw}
					var model agentmemory.AgentMemoryResourceModel
					diags := plan.Get(ctx, &model)
					require.False(t, diags.HasError(), diags.Errors())
					diags = plan.Set(ctx, &model)
					require.False(t, diags.HasError(), diags.Errors())
					assert.True(t, raw.Equal(plan.Raw), "decoding must preserve the collection state")
				})
			}
		})
	}
}

func TestAgentMemoryZeroListValuesAreTypedNull(t *testing.T) {
	for _, value := range []attr.Value{agentmemory.CustomMemoryTypeListValue{}, agentmemory.CustomFieldListValue{}, agentmemory.DetectorSelectionListValue{}, agentmemory.CustomDetectorListValue{}} {
		raw, err := value.ToTerraformValue(t.Context())
		require.NoError(t, err)
		assert.True(t, raw.IsNull())
		assert.True(t, value.Type(t.Context()).TerraformType(t.Context()).Equal(raw.Type()))
	}
}

func listValueAtPath(typ tftypes.Type, path []string, value any) tftypes.Value {
	if len(path) == 0 {
		return tftypes.NewValue(typ, value)
	}
	if list, ok := typ.(tftypes.List); ok {
		return tftypes.NewValue(typ, []tftypes.Value{listValueAtPath(list.ElementType, path, value)})
	}
	values := map[string]tftypes.Value{}
	for name, child := range typ.(tftypes.Object).AttributeTypes {
		values[name] = tftypes.NewValue(child, nil)
		if name == path[0] {
			values[name] = listValueAtPath(child, path[1:], value)
		}
	}
	return tftypes.NewValue(typ, values)
}
