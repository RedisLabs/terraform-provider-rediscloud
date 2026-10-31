package agentmemory

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

var customFieldElementType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(AgentMemoryCustomFieldModel{})}
var detectorSelectionElementType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(AgentMemoryDetectorSelectionModel{})}
var customMemoryTypeElementType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(AgentMemoryCustomMemoryTypeModel{})}
var customDetectorElementType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(AgentMemoryCustomDetectorModel{})}

var (
	_ basetypes.ListTypable  = CustomFieldListType{}
	_ basetypes.ListValuable = CustomFieldListValue{}
)

type CustomFieldListType struct{ basetypes.ListType }

func NewCustomFieldListType() CustomFieldListType {
	return CustomFieldListType{ListType: basetypes.ListType{ElemType: customFieldElementType}}
}

func (t CustomFieldListType) Equal(other attr.Type) bool {
	otherType, ok := other.(CustomFieldListType)
	return ok && t.ListType.Equal(otherType.ListType)
}

func (t CustomFieldListType) String() string { return "rediscloud.agentmemory.CustomFieldListType" }

func (t CustomFieldListType) ValueFromList(_ context.Context, value basetypes.ListValue) (basetypes.ListValuable, diag.Diagnostics) {
	return CustomFieldListValue{ListValue: value}, nil
}

func (t CustomFieldListType) ValueFromTerraform(ctx context.Context, value tftypes.Value) (attr.Value, error) {
	converted, err := t.ListType.ValueFromTerraform(ctx, value)
	if err != nil {
		return nil, err
	}
	list, ok := converted.(basetypes.ListValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T, expected basetypes.ListValue", converted)
	}
	return CustomFieldListValue{ListValue: list}, nil
}

func (t CustomFieldListType) ValueType(context.Context) attr.Value { return CustomFieldListValue{} }

type CustomFieldListValue struct{ basetypes.ListValue }

func (v CustomFieldListValue) Equal(other attr.Value) bool {
	otherValue, ok := other.(CustomFieldListValue)
	return ok && v.ListValue.Equal(otherValue.ListValue)
}

func (v CustomFieldListValue) Type(context.Context) attr.Type { return NewCustomFieldListType() }

// ToTerraformValue keeps a zero-value model field typed when it represents null.
func (v CustomFieldListValue) ToTerraformValue(ctx context.Context) (tftypes.Value, error) {
	if v.IsNull() {
		return tftypes.NewValue(v.Type(ctx).TerraformType(ctx), nil), nil
	}
	return v.ListValue.ToTerraformValue(ctx)
}

// AsModels requires a known, non-null list; callers retain unknowns during planning.
func (v CustomFieldListValue) AsModels(ctx context.Context) ([]AgentMemoryCustomFieldModel, diag.Diagnostics) {
	var models []AgentMemoryCustomFieldModel
	diags := v.ElementsAs(ctx, &models, false)
	return models, diags
}

func NewCustomFieldList(ctx context.Context, models []AgentMemoryCustomFieldModel) (CustomFieldListValue, diag.Diagnostics) {
	list, diags := types.ListValueFrom(ctx, customFieldElementType, models)
	return CustomFieldListValue{ListValue: list}, diags
}

var (
	_ basetypes.ListTypable  = CustomMemoryTypeListType{}
	_ basetypes.ListValuable = CustomMemoryTypeListValue{}
)

type CustomMemoryTypeListType struct{ basetypes.ListType }

func NewCustomMemoryTypeListType() CustomMemoryTypeListType {
	return CustomMemoryTypeListType{ListType: basetypes.ListType{ElemType: customMemoryTypeElementType}}
}

func (t CustomMemoryTypeListType) Equal(other attr.Type) bool {
	otherType, ok := other.(CustomMemoryTypeListType)
	return ok && t.ListType.Equal(otherType.ListType)
}

func (t CustomMemoryTypeListType) String() string {
	return "rediscloud.agentmemory.CustomMemoryTypeListType"
}

func (t CustomMemoryTypeListType) ValueFromList(_ context.Context, value basetypes.ListValue) (basetypes.ListValuable, diag.Diagnostics) {
	return CustomMemoryTypeListValue{ListValue: value}, nil
}

func (t CustomMemoryTypeListType) ValueFromTerraform(ctx context.Context, value tftypes.Value) (attr.Value, error) {
	converted, err := t.ListType.ValueFromTerraform(ctx, value)
	if err != nil {
		return nil, err
	}
	list, ok := converted.(basetypes.ListValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T, expected basetypes.ListValue", converted)
	}
	return CustomMemoryTypeListValue{ListValue: list}, nil
}

func (t CustomMemoryTypeListType) ValueType(context.Context) attr.Value {
	return CustomMemoryTypeListValue{}
}

type CustomMemoryTypeListValue struct{ basetypes.ListValue }

func (v CustomMemoryTypeListValue) Equal(other attr.Value) bool {
	otherValue, ok := other.(CustomMemoryTypeListValue)
	return ok && v.ListValue.Equal(otherValue.ListValue)
}

func (v CustomMemoryTypeListValue) Type(context.Context) attr.Type {
	return NewCustomMemoryTypeListType()
}

// ToTerraformValue keeps a zero-value model field typed when it represents null.
func (v CustomMemoryTypeListValue) ToTerraformValue(ctx context.Context) (tftypes.Value, error) {
	if v.IsNull() {
		return tftypes.NewValue(v.Type(ctx).TerraformType(ctx), nil), nil
	}
	return v.ListValue.ToTerraformValue(ctx)
}

// AsModels requires a known, non-null list; callers retain unknowns during planning.
func (v CustomMemoryTypeListValue) AsModels(ctx context.Context) ([]AgentMemoryCustomMemoryTypeModel, diag.Diagnostics) {
	var models []AgentMemoryCustomMemoryTypeModel
	diags := v.ElementsAs(ctx, &models, false)
	return models, diags
}

func NewCustomMemoryTypeList(ctx context.Context, models []AgentMemoryCustomMemoryTypeModel) (CustomMemoryTypeListValue, diag.Diagnostics) {
	list, diags := types.ListValueFrom(ctx, customMemoryTypeElementType, models)
	return CustomMemoryTypeListValue{ListValue: list}, diags
}

var (
	_ basetypes.ListTypable  = DetectorSelectionListType{}
	_ basetypes.ListValuable = DetectorSelectionListValue{}
)

type DetectorSelectionListType struct{ basetypes.ListType }

func NewDetectorSelectionListType() DetectorSelectionListType {
	return DetectorSelectionListType{ListType: basetypes.ListType{ElemType: detectorSelectionElementType}}
}

func (t DetectorSelectionListType) Equal(other attr.Type) bool {
	otherType, ok := other.(DetectorSelectionListType)
	return ok && t.ListType.Equal(otherType.ListType)
}

func (t DetectorSelectionListType) String() string {
	return "rediscloud.agentmemory.DetectorSelectionListType"
}

func (t DetectorSelectionListType) ValueFromList(_ context.Context, value basetypes.ListValue) (basetypes.ListValuable, diag.Diagnostics) {
	return DetectorSelectionListValue{ListValue: value}, nil
}

func (t DetectorSelectionListType) ValueFromTerraform(ctx context.Context, value tftypes.Value) (attr.Value, error) {
	converted, err := t.ListType.ValueFromTerraform(ctx, value)
	if err != nil {
		return nil, err
	}
	list, ok := converted.(basetypes.ListValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T, expected basetypes.ListValue", converted)
	}
	return DetectorSelectionListValue{ListValue: list}, nil
}

func (t DetectorSelectionListType) ValueType(context.Context) attr.Value {
	return DetectorSelectionListValue{}
}

type DetectorSelectionListValue struct{ basetypes.ListValue }

func (v DetectorSelectionListValue) Equal(other attr.Value) bool {
	otherValue, ok := other.(DetectorSelectionListValue)
	return ok && v.ListValue.Equal(otherValue.ListValue)
}

func (v DetectorSelectionListValue) Type(context.Context) attr.Type {
	return NewDetectorSelectionListType()
}

// ToTerraformValue keeps a zero-value model field typed when it represents null.
func (v DetectorSelectionListValue) ToTerraformValue(ctx context.Context) (tftypes.Value, error) {
	if v.IsNull() {
		return tftypes.NewValue(v.Type(ctx).TerraformType(ctx), nil), nil
	}
	return v.ListValue.ToTerraformValue(ctx)
}

// AsModels requires a known, non-null list; callers retain unknowns during planning.
func (v DetectorSelectionListValue) AsModels(ctx context.Context) ([]AgentMemoryDetectorSelectionModel, diag.Diagnostics) {
	var models []AgentMemoryDetectorSelectionModel
	diags := v.ElementsAs(ctx, &models, false)
	return models, diags
}

func NewDetectorSelectionList(ctx context.Context, models []AgentMemoryDetectorSelectionModel) (DetectorSelectionListValue, diag.Diagnostics) {
	list, diags := types.ListValueFrom(ctx, detectorSelectionElementType, models)
	return DetectorSelectionListValue{ListValue: list}, diags
}

var (
	_ basetypes.ListTypable  = CustomDetectorListType{}
	_ basetypes.ListValuable = CustomDetectorListValue{}
)

type CustomDetectorListType struct{ basetypes.ListType }

func NewCustomDetectorListType() CustomDetectorListType {
	return CustomDetectorListType{ListType: basetypes.ListType{ElemType: customDetectorElementType}}
}

func (t CustomDetectorListType) Equal(other attr.Type) bool {
	otherType, ok := other.(CustomDetectorListType)
	return ok && t.ListType.Equal(otherType.ListType)
}

func (t CustomDetectorListType) String() string {
	return "rediscloud.agentmemory.CustomDetectorListType"
}

func (t CustomDetectorListType) ValueFromList(_ context.Context, value basetypes.ListValue) (basetypes.ListValuable, diag.Diagnostics) {
	return CustomDetectorListValue{ListValue: value}, nil
}

func (t CustomDetectorListType) ValueFromTerraform(ctx context.Context, value tftypes.Value) (attr.Value, error) {
	converted, err := t.ListType.ValueFromTerraform(ctx, value)
	if err != nil {
		return nil, err
	}
	list, ok := converted.(basetypes.ListValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T, expected basetypes.ListValue", converted)
	}
	return CustomDetectorListValue{ListValue: list}, nil
}

func (t CustomDetectorListType) ValueType(context.Context) attr.Value {
	return CustomDetectorListValue{}
}

type CustomDetectorListValue struct{ basetypes.ListValue }

func (v CustomDetectorListValue) Equal(other attr.Value) bool {
	otherValue, ok := other.(CustomDetectorListValue)
	return ok && v.ListValue.Equal(otherValue.ListValue)
}

func (v CustomDetectorListValue) Type(context.Context) attr.Type { return NewCustomDetectorListType() }

// ToTerraformValue keeps a zero-value model field typed when it represents null.
func (v CustomDetectorListValue) ToTerraformValue(ctx context.Context) (tftypes.Value, error) {
	if v.IsNull() {
		return tftypes.NewValue(v.Type(ctx).TerraformType(ctx), nil), nil
	}
	return v.ListValue.ToTerraformValue(ctx)
}

// AsModels requires a known, non-null list; callers retain unknowns during planning.
func (v CustomDetectorListValue) AsModels(ctx context.Context) ([]AgentMemoryCustomDetectorModel, diag.Diagnostics) {
	var models []AgentMemoryCustomDetectorModel
	diags := v.ElementsAs(ctx, &models, false)
	return models, diags
}

func NewCustomDetectorList(ctx context.Context, models []AgentMemoryCustomDetectorModel) (CustomDetectorListValue, diag.Diagnostics) {
	list, diags := types.ListValueFrom(ctx, customDetectorElementType, models)
	return CustomDetectorListValue{ListValue: list}, diags
}

func customMemoryTypeModels(ctx context.Context, list CustomMemoryTypeListValue, diags *diag.Diagnostics) []AgentMemoryCustomMemoryTypeModel {
	if list.IsNull() {
		return nil
	}
	if list.IsUnknown() {
		diags.AddError("Unknown Agent Memory configuration", "The custom memory types list must be known before an API request can be built.")
		return nil
	}
	models, modelDiags := list.AsModels(ctx)
	diags.Append(modelDiags...)
	if modelDiags.HasError() {
		return nil
	}
	return models
}

func customFieldModels(ctx context.Context, list CustomFieldListValue, diags *diag.Diagnostics) []AgentMemoryCustomFieldModel {
	if list.IsNull() {
		return nil
	}
	if list.IsUnknown() {
		diags.AddError("Unknown Agent Memory configuration", "The custom fields list must be known before an API request can be built.")
		return nil
	}
	models, modelDiags := list.AsModels(ctx)
	diags.Append(modelDiags...)
	if modelDiags.HasError() {
		return nil
	}
	return models
}

func customDetectorModels(ctx context.Context, list CustomDetectorListValue, diags *diag.Diagnostics) []AgentMemoryCustomDetectorModel {
	if list.IsNull() {
		return nil
	}
	if list.IsUnknown() {
		diags.AddError("Unknown Agent Memory configuration", "The custom detectors list must be known before an API request can be built.")
		return nil
	}
	models, modelDiags := list.AsModels(ctx)
	diags.Append(modelDiags...)
	if modelDiags.HasError() {
		return nil
	}
	return models
}

func detectorSelectionModels(ctx context.Context, list DetectorSelectionListValue, diags *diag.Diagnostics) []AgentMemoryDetectorSelectionModel {
	if list.IsNull() {
		return nil
	}
	if list.IsUnknown() {
		diags.AddError("Unknown Agent Memory configuration", "The built-in detectors list must be known before an API request can be built.")
		return nil
	}
	models, modelDiags := list.AsModels(ctx)
	diags.Append(modelDiags...)
	if modelDiags.HasError() {
		return nil
	}
	return models
}
