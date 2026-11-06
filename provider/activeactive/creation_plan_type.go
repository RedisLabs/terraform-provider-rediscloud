package activeactive

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
)

// Creation-plan regions are a configurable set, and the plan itself is a configurable
// list. Both get a custom collection type so the resource model can decode them with
// AsModels. The pro subscription creation plan is a different shape and does not share
// these types.

var (
	_ basetypes.SetTypable   = CreationPlanRegionSetType{}
	_ basetypes.SetValuable  = CreationPlanRegionSetValue{}
	_ basetypes.ListTypable  = CreationPlanListType{}
	_ basetypes.ListValuable = CreationPlanListValue{}
)

// CreationPlanRegionModel is one region entry in an Active-Active creation plan.
type CreationPlanRegionModel struct {
	Region                   types.String `tfsdk:"region"`
	NetworkingDeploymentCIDR types.String `tfsdk:"networking_deployment_cidr"`
	WriteOperationsPerSecond types.Int64  `tfsdk:"write_operations_per_second"`
	ReadOperationsPerSecond  types.Int64  `tfsdk:"read_operations_per_second"`
}

// CreationPlanModel is the single creation-plan block.
type CreationPlanModel struct {
	MemoryLimitInGB types.Float64              `tfsdk:"memory_limit_in_gb"`
	DatasetSizeInGB types.Float64              `tfsdk:"dataset_size_in_gb"`
	Quantity        types.Int64                `tfsdk:"quantity"`
	Modules         types.List                 `tfsdk:"modules"`
	Region          CreationPlanRegionSetValue `tfsdk:"region"`
}

var creationPlanRegionAttrTypes = customtypes.AttrTypesOf(CreationPlanRegionModel{})

var creationPlanRegionElemType = types.ObjectType{AttrTypes: creationPlanRegionAttrTypes}

// Seed Modules with its element type. An uninitialised list cannot describe its elements.
var creationPlanAttrTypes = customtypes.AttrTypesOf(CreationPlanModel{
	Modules: types.ListNull(types.StringType),
	Region:  nullCreationPlanRegionSetValue(),
})

var creationPlanElemType = types.ObjectType{AttrTypes: creationPlanAttrTypes}

// CreationPlanRegionSetType is the custom set type for creation-plan regions.
type CreationPlanRegionSetType struct {
	basetypes.SetType
}

// NewCreationPlanRegionSetType returns the custom type for creation-plan regions.
func NewCreationPlanRegionSetType() CreationPlanRegionSetType {
	return CreationPlanRegionSetType{SetType: basetypes.SetType{ElemType: creationPlanRegionElemType}}
}

func (t CreationPlanRegionSetType) Equal(other attr.Type) bool {
	otherType, ok := other.(CreationPlanRegionSetType)
	if !ok {
		return false
	}
	return t.SetType.Equal(otherType.SetType)
}

func (t CreationPlanRegionSetType) String() string {
	return "rediscloud.activeactive.CreationPlanRegionSetType"
}

func (t CreationPlanRegionSetType) ValueFromSet(_ context.Context, value basetypes.SetValue) (basetypes.SetValuable, diag.Diagnostics) {
	return CreationPlanRegionSetValue{SetValue: value}, nil
}

func (t CreationPlanRegionSetType) ValueFromTerraform(ctx context.Context, value tftypes.Value) (attr.Value, error) {
	terraformValue, err := t.SetType.ValueFromTerraform(ctx, value)
	if err != nil {
		return nil, err
	}

	setValue, ok := terraformValue.(basetypes.SetValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T, expected basetypes.SetValue", terraformValue)
	}

	return CreationPlanRegionSetValue{SetValue: setValue}, nil
}

func (t CreationPlanRegionSetType) ValueType(context.Context) attr.Value {
	return CreationPlanRegionSetValue{}
}

// CreationPlanRegionSetValue stores creation-plan regions.
type CreationPlanRegionSetValue struct {
	basetypes.SetValue
}

func (v CreationPlanRegionSetValue) Equal(other attr.Value) bool {
	otherValue, ok := other.(CreationPlanRegionSetValue)
	if !ok {
		return false
	}
	return v.SetValue.Equal(otherValue.SetValue)
}

func (v CreationPlanRegionSetValue) Type(context.Context) attr.Type {
	return NewCreationPlanRegionSetType()
}

// AsModels decodes a known region set. Callers must handle null and unknown first.
func (v CreationPlanRegionSetValue) AsModels(ctx context.Context) ([]CreationPlanRegionModel, diag.Diagnostics) {
	var models []CreationPlanRegionModel
	diags := v.ElementsAs(ctx, &models, false)
	return models, diags
}

func nullCreationPlanRegionSetValue() CreationPlanRegionSetValue {
	return CreationPlanRegionSetValue{SetValue: types.SetNull(creationPlanRegionElemType)}
}

func unknownCreationPlanRegionSetValue() CreationPlanRegionSetValue {
	return CreationPlanRegionSetValue{SetValue: types.SetUnknown(creationPlanRegionElemType)}
}

// NewCreationPlanRegionSetValue builds a known set. A nil slice stays null.
func NewCreationPlanRegionSetValue(ctx context.Context, models []CreationPlanRegionModel) (CreationPlanRegionSetValue, diag.Diagnostics) {
	if models == nil {
		return nullCreationPlanRegionSetValue(), nil
	}
	set, diags := types.SetValueFrom(ctx, creationPlanRegionElemType, models)
	return CreationPlanRegionSetValue{SetValue: set}, diags
}

// CreationPlanListType is the custom list type for the creation-plan block.
type CreationPlanListType struct {
	basetypes.ListType
}

// NewCreationPlanListType returns the custom type for a creation plan.
func NewCreationPlanListType() CreationPlanListType {
	return CreationPlanListType{ListType: basetypes.ListType{ElemType: creationPlanElemType}}
}

func (t CreationPlanListType) Equal(other attr.Type) bool {
	otherType, ok := other.(CreationPlanListType)
	if !ok {
		return false
	}
	return t.ListType.Equal(otherType.ListType)
}

func (t CreationPlanListType) String() string {
	return "rediscloud.activeactive.CreationPlanListType"
}

func (t CreationPlanListType) ValueFromList(_ context.Context, value basetypes.ListValue) (basetypes.ListValuable, diag.Diagnostics) {
	return CreationPlanListValue{ListValue: value}, nil
}

func (t CreationPlanListType) ValueFromTerraform(ctx context.Context, value tftypes.Value) (attr.Value, error) {
	terraformValue, err := t.ListType.ValueFromTerraform(ctx, value)
	if err != nil {
		return nil, err
	}

	listValue, ok := terraformValue.(basetypes.ListValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T, expected basetypes.ListValue", terraformValue)
	}

	return CreationPlanListValue{ListValue: listValue}, nil
}

func (t CreationPlanListType) ValueType(context.Context) attr.Value {
	return CreationPlanListValue{}
}

// CreationPlanListValue stores the creation-plan block.
type CreationPlanListValue struct {
	basetypes.ListValue
}

func (v CreationPlanListValue) Equal(other attr.Value) bool {
	otherValue, ok := other.(CreationPlanListValue)
	if !ok {
		return false
	}
	return v.ListValue.Equal(otherValue.ListValue)
}

func (v CreationPlanListValue) Type(context.Context) attr.Type {
	return NewCreationPlanListType()
}

// AsModels decodes a known creation plan. Callers must handle null and unknown first.
func (v CreationPlanListValue) AsModels(ctx context.Context) ([]CreationPlanModel, diag.Diagnostics) {
	var models []CreationPlanModel
	diags := v.ElementsAs(ctx, &models, false)
	return models, diags
}

func nullCreationPlanListValue() CreationPlanListValue {
	return CreationPlanListValue{ListValue: types.ListNull(creationPlanElemType)}
}

func unknownCreationPlanListValue() CreationPlanListValue {
	return CreationPlanListValue{ListValue: types.ListUnknown(creationPlanElemType)}
}

// NewCreationPlanListValue builds a known creation plan. A nil slice stays null.
func NewCreationPlanListValue(ctx context.Context, models []CreationPlanModel) (CreationPlanListValue, diag.Diagnostics) {
	if models == nil {
		return nullCreationPlanListValue(), nil
	}
	list, diags := types.ListValueFrom(ctx, creationPlanElemType, models)
	return CreationPlanListValue{ListValue: list}, diags
}
