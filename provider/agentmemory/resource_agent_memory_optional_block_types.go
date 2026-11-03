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

// Optional blocks use zero-or-one-element lists so Terraform can preserve an
// unknown block count until apply. Go pointers cannot represent that third state.

var (
	_ basetypes.ListTypable  = ModelConfigListType{}
	_ basetypes.ListValuable = ModelConfigListValue{}
)

type ModelConfigListType struct{ basetypes.ListType }

func NewModelConfigListType() ModelConfigListType {
	return ModelConfigListType{ListType: basetypes.ListType{ElemType: modelConfigElementType()}}
}

func modelConfigElementType() types.ObjectType {
	return types.ObjectType{AttrTypes: customtypes.AttrTypesOf(AgentMemoryModelConfigModel{})}
}

func (t ModelConfigListType) Equal(other attr.Type) bool {
	otherType, ok := other.(ModelConfigListType)
	return ok && t.ListType.Equal(otherType.ListType)
}

func (t ModelConfigListType) String() string { return "rediscloud.agentmemory.ModelConfigListType" }

func (t ModelConfigListType) ValueFromList(_ context.Context, value basetypes.ListValue) (basetypes.ListValuable, diag.Diagnostics) {
	return ModelConfigListValue{ListValue: value}, nil
}

func (t ModelConfigListType) ValueFromTerraform(ctx context.Context, value tftypes.Value) (attr.Value, error) {
	converted, err := t.ListType.ValueFromTerraform(ctx, value)
	if err != nil {
		return nil, err
	}
	list, ok := converted.(basetypes.ListValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T, expected basetypes.ListValue", converted)
	}
	return ModelConfigListValue{ListValue: list}, nil
}

func (t ModelConfigListType) ValueType(context.Context) attr.Value { return ModelConfigListValue{} }

type ModelConfigListValue struct{ basetypes.ListValue }

func (v ModelConfigListValue) Equal(other attr.Value) bool {
	otherValue, ok := other.(ModelConfigListValue)
	return ok && v.ListValue.Equal(otherValue.ListValue)
}

func (v ModelConfigListValue) Type(context.Context) attr.Type { return NewModelConfigListType() }

func (v ModelConfigListValue) ToTerraformValue(ctx context.Context) (tftypes.Value, error) {
	if v.IsNull() {
		return tftypes.NewValue(v.Type(ctx).TerraformType(ctx), nil), nil
	}
	return v.ListValue.ToTerraformValue(ctx)
}

func (v ModelConfigListValue) AsModels(ctx context.Context) ([]AgentMemoryModelConfigModel, diag.Diagnostics) {
	var models []AgentMemoryModelConfigModel
	diags := v.ElementsAs(ctx, &models, false)
	return models, diags
}

func NewModelConfigList(ctx context.Context, models []AgentMemoryModelConfigModel) (ModelConfigListValue, diag.Diagnostics) {
	list, diags := types.ListValueFrom(ctx, modelConfigElementType(), models)
	return ModelConfigListValue{ListValue: list}, diags
}

func modelConfigBlock(ctx context.Context, list ModelConfigListValue, diags *diag.Diagnostics) *AgentMemoryModelConfigModel {
	if list.IsNull() {
		return nil
	}
	if list.IsUnknown() {
		diags.AddError("Unknown Agent Memory configuration", "The model configuration block must be known before its values can be used.")
		return nil
	}
	models, modelDiags := list.AsModels(ctx)
	diags.Append(modelDiags...)
	if modelDiags.HasError() || len(models) == 0 {
		return nil
	}
	if len(models) != 1 {
		diags.AddError("Invalid Agent Memory configuration", "Only one model configuration block can be configured.")
		return nil
	}
	return &models[0]
}

func modelConfigList(ctx context.Context, model *AgentMemoryModelConfigModel, diags *diag.Diagnostics) ModelConfigListValue {
	models := make([]AgentMemoryModelConfigModel, 0, 1)
	if model != nil {
		models = append(models, *model)
	}
	list, listDiags := NewModelConfigList(ctx, models)
	diags.Append(listDiags...)
	return list
}

var (
	_ basetypes.ListTypable  = ModelCredentialsListType{}
	_ basetypes.ListValuable = ModelCredentialsListValue{}
)

type ModelCredentialsListType struct{ basetypes.ListType }

func NewModelCredentialsListType() ModelCredentialsListType {
	return ModelCredentialsListType{ListType: basetypes.ListType{ElemType: modelCredentialsElementType()}}
}

func modelCredentialsElementType() types.ObjectType {
	return types.ObjectType{AttrTypes: customtypes.AttrTypesOf(AgentMemoryModelCredentialsModel{})}
}

func (t ModelCredentialsListType) Equal(other attr.Type) bool {
	otherType, ok := other.(ModelCredentialsListType)
	return ok && t.ListType.Equal(otherType.ListType)
}

func (t ModelCredentialsListType) String() string {
	return "rediscloud.agentmemory.ModelCredentialsListType"
}

func (t ModelCredentialsListType) ValueFromList(_ context.Context, value basetypes.ListValue) (basetypes.ListValuable, diag.Diagnostics) {
	return ModelCredentialsListValue{ListValue: value}, nil
}

func (t ModelCredentialsListType) ValueFromTerraform(ctx context.Context, value tftypes.Value) (attr.Value, error) {
	converted, err := t.ListType.ValueFromTerraform(ctx, value)
	if err != nil {
		return nil, err
	}
	list, ok := converted.(basetypes.ListValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T, expected basetypes.ListValue", converted)
	}
	return ModelCredentialsListValue{ListValue: list}, nil
}

func (t ModelCredentialsListType) ValueType(context.Context) attr.Value {
	return ModelCredentialsListValue{}
}

type ModelCredentialsListValue struct{ basetypes.ListValue }

func (v ModelCredentialsListValue) Equal(other attr.Value) bool {
	otherValue, ok := other.(ModelCredentialsListValue)
	return ok && v.ListValue.Equal(otherValue.ListValue)
}

func (v ModelCredentialsListValue) Type(context.Context) attr.Type {
	return NewModelCredentialsListType()
}

func (v ModelCredentialsListValue) ToTerraformValue(ctx context.Context) (tftypes.Value, error) {
	if v.IsNull() {
		return tftypes.NewValue(v.Type(ctx).TerraformType(ctx), nil), nil
	}
	return v.ListValue.ToTerraformValue(ctx)
}

func (v ModelCredentialsListValue) AsModels(ctx context.Context) ([]AgentMemoryModelCredentialsModel, diag.Diagnostics) {
	var models []AgentMemoryModelCredentialsModel
	diags := v.ElementsAs(ctx, &models, false)
	return models, diags
}

func NewModelCredentialsList(ctx context.Context, models []AgentMemoryModelCredentialsModel) (ModelCredentialsListValue, diag.Diagnostics) {
	list, diags := types.ListValueFrom(ctx, modelCredentialsElementType(), models)
	return ModelCredentialsListValue{ListValue: list}, diags
}

func modelCredentialsBlock(ctx context.Context, list ModelCredentialsListValue, diags *diag.Diagnostics) *AgentMemoryModelCredentialsModel {
	if list.IsNull() {
		return nil
	}
	if list.IsUnknown() {
		diags.AddError("Unknown Agent Memory configuration", "The credentials block must be known before its values can be used.")
		return nil
	}
	models, modelDiags := list.AsModels(ctx)
	diags.Append(modelDiags...)
	if modelDiags.HasError() || len(models) == 0 {
		return nil
	}
	if len(models) != 1 {
		diags.AddError("Invalid Agent Memory configuration", "Only one credentials block can be configured.")
		return nil
	}
	return &models[0]
}

func modelCredentialsList(ctx context.Context, model *AgentMemoryModelCredentialsModel, diags *diag.Diagnostics) ModelCredentialsListValue {
	models := make([]AgentMemoryModelCredentialsModel, 0, 1)
	if model != nil {
		models = append(models, *model)
	}
	list, listDiags := NewModelCredentialsList(ctx, models)
	diags.Append(listDiags...)
	return list
}

var (
	_ basetypes.ListTypable  = SummarizationListType{}
	_ basetypes.ListValuable = SummarizationListValue{}
)

type SummarizationListType struct{ basetypes.ListType }

func NewSummarizationListType() SummarizationListType {
	return SummarizationListType{ListType: basetypes.ListType{ElemType: summarizationElementType()}}
}

func summarizationElementType() types.ObjectType {
	return types.ObjectType{AttrTypes: customtypes.AttrTypesOf(AgentMemorySummarizationModel{})}
}

func (t SummarizationListType) Equal(other attr.Type) bool {
	otherType, ok := other.(SummarizationListType)
	return ok && t.ListType.Equal(otherType.ListType)
}

func (t SummarizationListType) String() string { return "rediscloud.agentmemory.SummarizationListType" }

func (t SummarizationListType) ValueFromList(_ context.Context, value basetypes.ListValue) (basetypes.ListValuable, diag.Diagnostics) {
	return SummarizationListValue{ListValue: value}, nil
}

func (t SummarizationListType) ValueFromTerraform(ctx context.Context, value tftypes.Value) (attr.Value, error) {
	converted, err := t.ListType.ValueFromTerraform(ctx, value)
	if err != nil {
		return nil, err
	}
	list, ok := converted.(basetypes.ListValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T, expected basetypes.ListValue", converted)
	}
	return SummarizationListValue{ListValue: list}, nil
}

func (t SummarizationListType) ValueType(context.Context) attr.Value { return SummarizationListValue{} }

type SummarizationListValue struct{ basetypes.ListValue }

func (v SummarizationListValue) Equal(other attr.Value) bool {
	otherValue, ok := other.(SummarizationListValue)
	return ok && v.ListValue.Equal(otherValue.ListValue)
}

func (v SummarizationListValue) Type(context.Context) attr.Type { return NewSummarizationListType() }

func (v SummarizationListValue) ToTerraformValue(ctx context.Context) (tftypes.Value, error) {
	if v.IsNull() {
		return tftypes.NewValue(v.Type(ctx).TerraformType(ctx), nil), nil
	}
	return v.ListValue.ToTerraformValue(ctx)
}

func (v SummarizationListValue) AsModels(ctx context.Context) ([]AgentMemorySummarizationModel, diag.Diagnostics) {
	var models []AgentMemorySummarizationModel
	diags := v.ElementsAs(ctx, &models, false)
	return models, diags
}

func NewSummarizationList(ctx context.Context, models []AgentMemorySummarizationModel) (SummarizationListValue, diag.Diagnostics) {
	list, diags := types.ListValueFrom(ctx, summarizationElementType(), models)
	return SummarizationListValue{ListValue: list}, diags
}

func summarizationBlock(ctx context.Context, list SummarizationListValue, diags *diag.Diagnostics) *AgentMemorySummarizationModel {
	if list.IsNull() {
		return nil
	}
	if list.IsUnknown() {
		diags.AddError("Unknown Agent Memory configuration", "The summarization block must be known before its values can be used.")
		return nil
	}
	models, modelDiags := list.AsModels(ctx)
	diags.Append(modelDiags...)
	if modelDiags.HasError() || len(models) == 0 {
		return nil
	}
	if len(models) != 1 {
		diags.AddError("Invalid Agent Memory configuration", "Only one summarization block can be configured.")
		return nil
	}
	return &models[0]
}

func summarizationList(ctx context.Context, model *AgentMemorySummarizationModel, diags *diag.Diagnostics) SummarizationListValue {
	models := make([]AgentMemorySummarizationModel, 0, 1)
	if model != nil {
		models = append(models, *model)
	}
	list, listDiags := NewSummarizationList(ctx, models)
	diags.Append(listDiags...)
	return list
}

var (
	_ basetypes.ListTypable  = EventCountListType{}
	_ basetypes.ListValuable = EventCountListValue{}
)

type EventCountListType struct{ basetypes.ListType }

func NewEventCountListType() EventCountListType {
	return EventCountListType{ListType: basetypes.ListType{ElemType: eventCountElementType()}}
}

func eventCountElementType() types.ObjectType {
	return types.ObjectType{AttrTypes: customtypes.AttrTypesOf(AgentMemoryEventCountModel{})}
}

func (t EventCountListType) Equal(other attr.Type) bool {
	otherType, ok := other.(EventCountListType)
	return ok && t.ListType.Equal(otherType.ListType)
}

func (t EventCountListType) String() string { return "rediscloud.agentmemory.EventCountListType" }

func (t EventCountListType) ValueFromList(_ context.Context, value basetypes.ListValue) (basetypes.ListValuable, diag.Diagnostics) {
	return EventCountListValue{ListValue: value}, nil
}

func (t EventCountListType) ValueFromTerraform(ctx context.Context, value tftypes.Value) (attr.Value, error) {
	converted, err := t.ListType.ValueFromTerraform(ctx, value)
	if err != nil {
		return nil, err
	}
	list, ok := converted.(basetypes.ListValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T, expected basetypes.ListValue", converted)
	}
	return EventCountListValue{ListValue: list}, nil
}

func (t EventCountListType) ValueType(context.Context) attr.Value { return EventCountListValue{} }

type EventCountListValue struct{ basetypes.ListValue }

func (v EventCountListValue) Equal(other attr.Value) bool {
	otherValue, ok := other.(EventCountListValue)
	return ok && v.ListValue.Equal(otherValue.ListValue)
}

func (v EventCountListValue) Type(context.Context) attr.Type { return NewEventCountListType() }

func (v EventCountListValue) ToTerraformValue(ctx context.Context) (tftypes.Value, error) {
	if v.IsNull() {
		return tftypes.NewValue(v.Type(ctx).TerraformType(ctx), nil), nil
	}
	return v.ListValue.ToTerraformValue(ctx)
}

func (v EventCountListValue) AsModels(ctx context.Context) ([]AgentMemoryEventCountModel, diag.Diagnostics) {
	var models []AgentMemoryEventCountModel
	diags := v.ElementsAs(ctx, &models, false)
	return models, diags
}

func NewEventCountList(ctx context.Context, models []AgentMemoryEventCountModel) (EventCountListValue, diag.Diagnostics) {
	list, diags := types.ListValueFrom(ctx, eventCountElementType(), models)
	return EventCountListValue{ListValue: list}, diags
}

func eventCountBlock(ctx context.Context, list EventCountListValue, diags *diag.Diagnostics) *AgentMemoryEventCountModel {
	if list.IsNull() {
		return nil
	}
	if list.IsUnknown() {
		diags.AddError("Unknown Agent Memory configuration", "The event_count block must be known before its values can be used.")
		return nil
	}
	models, modelDiags := list.AsModels(ctx)
	diags.Append(modelDiags...)
	if modelDiags.HasError() || len(models) == 0 {
		return nil
	}
	if len(models) != 1 {
		diags.AddError("Invalid Agent Memory configuration", "Only one event_count block can be configured.")
		return nil
	}
	return &models[0]
}

func eventCountList(ctx context.Context, model *AgentMemoryEventCountModel, diags *diag.Diagnostics) EventCountListValue {
	models := make([]AgentMemoryEventCountModel, 0, 1)
	if model != nil {
		models = append(models, *model)
	}
	list, listDiags := NewEventCountList(ctx, models)
	diags.Append(listDiags...)
	return list
}

var (
	_ basetypes.ListTypable  = ExclusionsListType{}
	_ basetypes.ListValuable = ExclusionsListValue{}
)

type ExclusionsListType struct{ basetypes.ListType }

func NewExclusionsListType() ExclusionsListType {
	return ExclusionsListType{ListType: basetypes.ListType{ElemType: exclusionsElementType()}}
}

func exclusionsElementType() types.ObjectType {
	return types.ObjectType{AttrTypes: customtypes.AttrTypesOf(AgentMemoryExclusionsModel{})}
}

func (t ExclusionsListType) Equal(other attr.Type) bool {
	otherType, ok := other.(ExclusionsListType)
	return ok && t.ListType.Equal(otherType.ListType)
}

func (t ExclusionsListType) String() string { return "rediscloud.agentmemory.ExclusionsListType" }

func (t ExclusionsListType) ValueFromList(_ context.Context, value basetypes.ListValue) (basetypes.ListValuable, diag.Diagnostics) {
	return ExclusionsListValue{ListValue: value}, nil
}

func (t ExclusionsListType) ValueFromTerraform(ctx context.Context, value tftypes.Value) (attr.Value, error) {
	converted, err := t.ListType.ValueFromTerraform(ctx, value)
	if err != nil {
		return nil, err
	}
	list, ok := converted.(basetypes.ListValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T, expected basetypes.ListValue", converted)
	}
	return ExclusionsListValue{ListValue: list}, nil
}

func (t ExclusionsListType) ValueType(context.Context) attr.Value { return ExclusionsListValue{} }

type ExclusionsListValue struct{ basetypes.ListValue }

func (v ExclusionsListValue) Equal(other attr.Value) bool {
	otherValue, ok := other.(ExclusionsListValue)
	return ok && v.ListValue.Equal(otherValue.ListValue)
}

func (v ExclusionsListValue) Type(context.Context) attr.Type { return NewExclusionsListType() }

func (v ExclusionsListValue) ToTerraformValue(ctx context.Context) (tftypes.Value, error) {
	if v.IsNull() {
		return tftypes.NewValue(v.Type(ctx).TerraformType(ctx), nil), nil
	}
	return v.ListValue.ToTerraformValue(ctx)
}

func (v ExclusionsListValue) AsModels(ctx context.Context) ([]AgentMemoryExclusionsModel, diag.Diagnostics) {
	var models []AgentMemoryExclusionsModel
	diags := v.ElementsAs(ctx, &models, false)
	return models, diags
}

func NewExclusionsList(ctx context.Context, models []AgentMemoryExclusionsModel) (ExclusionsListValue, diag.Diagnostics) {
	list, diags := types.ListValueFrom(ctx, exclusionsElementType(), models)
	return ExclusionsListValue{ListValue: list}, diags
}

func exclusionsBlock(ctx context.Context, list ExclusionsListValue, diags *diag.Diagnostics) *AgentMemoryExclusionsModel {
	if list.IsNull() {
		return nil
	}
	if list.IsUnknown() {
		diags.AddError("Unknown Agent Memory configuration", "The long_term_memory_exclusions block must be known before its values can be used.")
		return nil
	}
	models, modelDiags := list.AsModels(ctx)
	diags.Append(modelDiags...)
	if modelDiags.HasError() || len(models) == 0 {
		return nil
	}
	if len(models) != 1 {
		diags.AddError("Invalid Agent Memory configuration", "Only one long_term_memory_exclusions block can be configured.")
		return nil
	}
	return &models[0]
}

func exclusionsList(ctx context.Context, model *AgentMemoryExclusionsModel, diags *diag.Diagnostics) ExclusionsListValue {
	models := make([]AgentMemoryExclusionsModel, 0, 1)
	if model != nil {
		models = append(models, *model)
	}
	list, listDiags := NewExclusionsList(ctx, models)
	diags.Append(listDiags...)
	return list
}

var (
	_ basetypes.ListTypable  = SemanticExclusionListType{}
	_ basetypes.ListValuable = SemanticExclusionListValue{}
)

type SemanticExclusionListType struct{ basetypes.ListType }

func NewSemanticExclusionListType() SemanticExclusionListType {
	return SemanticExclusionListType{ListType: basetypes.ListType{ElemType: semanticExclusionElementType()}}
}

func semanticExclusionElementType() types.ObjectType {
	return types.ObjectType{AttrTypes: customtypes.AttrTypesOf(AgentMemorySemanticExclusionModel{})}
}

func (t SemanticExclusionListType) Equal(other attr.Type) bool {
	otherType, ok := other.(SemanticExclusionListType)
	return ok && t.ListType.Equal(otherType.ListType)
}

func (t SemanticExclusionListType) String() string {
	return "rediscloud.agentmemory.SemanticExclusionListType"
}

func (t SemanticExclusionListType) ValueFromList(_ context.Context, value basetypes.ListValue) (basetypes.ListValuable, diag.Diagnostics) {
	return SemanticExclusionListValue{ListValue: value}, nil
}

func (t SemanticExclusionListType) ValueFromTerraform(ctx context.Context, value tftypes.Value) (attr.Value, error) {
	converted, err := t.ListType.ValueFromTerraform(ctx, value)
	if err != nil {
		return nil, err
	}
	list, ok := converted.(basetypes.ListValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T, expected basetypes.ListValue", converted)
	}
	return SemanticExclusionListValue{ListValue: list}, nil
}

func (t SemanticExclusionListType) ValueType(context.Context) attr.Value {
	return SemanticExclusionListValue{}
}

type SemanticExclusionListValue struct{ basetypes.ListValue }

func (v SemanticExclusionListValue) Equal(other attr.Value) bool {
	otherValue, ok := other.(SemanticExclusionListValue)
	return ok && v.ListValue.Equal(otherValue.ListValue)
}

func (v SemanticExclusionListValue) Type(context.Context) attr.Type {
	return NewSemanticExclusionListType()
}

func (v SemanticExclusionListValue) ToTerraformValue(ctx context.Context) (tftypes.Value, error) {
	if v.IsNull() {
		return tftypes.NewValue(v.Type(ctx).TerraformType(ctx), nil), nil
	}
	return v.ListValue.ToTerraformValue(ctx)
}

func (v SemanticExclusionListValue) AsModels(ctx context.Context) ([]AgentMemorySemanticExclusionModel, diag.Diagnostics) {
	var models []AgentMemorySemanticExclusionModel
	diags := v.ElementsAs(ctx, &models, false)
	return models, diags
}

func NewSemanticExclusionList(ctx context.Context, models []AgentMemorySemanticExclusionModel) (SemanticExclusionListValue, diag.Diagnostics) {
	list, diags := types.ListValueFrom(ctx, semanticExclusionElementType(), models)
	return SemanticExclusionListValue{ListValue: list}, diags
}

func semanticExclusionBlock(ctx context.Context, list SemanticExclusionListValue, diags *diag.Diagnostics) *AgentMemorySemanticExclusionModel {
	if list.IsNull() {
		return nil
	}
	if list.IsUnknown() {
		diags.AddError("Unknown Agent Memory configuration", "The semantic block must be known before its values can be used.")
		return nil
	}
	models, modelDiags := list.AsModels(ctx)
	diags.Append(modelDiags...)
	if modelDiags.HasError() || len(models) == 0 {
		return nil
	}
	if len(models) != 1 {
		diags.AddError("Invalid Agent Memory configuration", "Only one semantic block can be configured.")
		return nil
	}
	return &models[0]
}

func semanticExclusionList(ctx context.Context, model *AgentMemorySemanticExclusionModel, diags *diag.Diagnostics) SemanticExclusionListValue {
	models := make([]AgentMemorySemanticExclusionModel, 0, 1)
	if model != nil {
		models = append(models, *model)
	}
	list, listDiags := NewSemanticExclusionList(ctx, models)
	diags.Append(listDiags...)
	return list
}

var (
	_ basetypes.ListTypable  = BuiltInDetectorsListType{}
	_ basetypes.ListValuable = BuiltInDetectorsListValue{}
)

type BuiltInDetectorsListType struct{ basetypes.ListType }

func NewBuiltInDetectorsListType() BuiltInDetectorsListType {
	return BuiltInDetectorsListType{ListType: basetypes.ListType{ElemType: builtInDetectorsElementType()}}
}

func builtInDetectorsElementType() types.ObjectType {
	return types.ObjectType{AttrTypes: customtypes.AttrTypesOf(AgentMemoryBuiltInDetectorsModel{})}
}

func (t BuiltInDetectorsListType) Equal(other attr.Type) bool {
	otherType, ok := other.(BuiltInDetectorsListType)
	return ok && t.ListType.Equal(otherType.ListType)
}

func (t BuiltInDetectorsListType) String() string {
	return "rediscloud.agentmemory.BuiltInDetectorsListType"
}

func (t BuiltInDetectorsListType) ValueFromList(_ context.Context, value basetypes.ListValue) (basetypes.ListValuable, diag.Diagnostics) {
	return BuiltInDetectorsListValue{ListValue: value}, nil
}

func (t BuiltInDetectorsListType) ValueFromTerraform(ctx context.Context, value tftypes.Value) (attr.Value, error) {
	converted, err := t.ListType.ValueFromTerraform(ctx, value)
	if err != nil {
		return nil, err
	}
	list, ok := converted.(basetypes.ListValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T, expected basetypes.ListValue", converted)
	}
	return BuiltInDetectorsListValue{ListValue: list}, nil
}

func (t BuiltInDetectorsListType) ValueType(context.Context) attr.Value {
	return BuiltInDetectorsListValue{}
}

type BuiltInDetectorsListValue struct{ basetypes.ListValue }

func (v BuiltInDetectorsListValue) Equal(other attr.Value) bool {
	otherValue, ok := other.(BuiltInDetectorsListValue)
	return ok && v.ListValue.Equal(otherValue.ListValue)
}

func (v BuiltInDetectorsListValue) Type(context.Context) attr.Type {
	return NewBuiltInDetectorsListType()
}

func (v BuiltInDetectorsListValue) ToTerraformValue(ctx context.Context) (tftypes.Value, error) {
	if v.IsNull() {
		return tftypes.NewValue(v.Type(ctx).TerraformType(ctx), nil), nil
	}
	return v.ListValue.ToTerraformValue(ctx)
}

func (v BuiltInDetectorsListValue) AsModels(ctx context.Context) ([]AgentMemoryBuiltInDetectorsModel, diag.Diagnostics) {
	var models []AgentMemoryBuiltInDetectorsModel
	diags := v.ElementsAs(ctx, &models, false)
	return models, diags
}

func NewBuiltInDetectorsList(ctx context.Context, models []AgentMemoryBuiltInDetectorsModel) (BuiltInDetectorsListValue, diag.Diagnostics) {
	list, diags := types.ListValueFrom(ctx, builtInDetectorsElementType(), models)
	return BuiltInDetectorsListValue{ListValue: list}, diags
}

func builtInDetectorsBlock(ctx context.Context, list BuiltInDetectorsListValue, diags *diag.Diagnostics) *AgentMemoryBuiltInDetectorsModel {
	if list.IsNull() {
		return nil
	}
	if list.IsUnknown() {
		diags.AddError("Unknown Agent Memory configuration", "The built_in_detectors block must be known before its values can be used.")
		return nil
	}
	models, modelDiags := list.AsModels(ctx)
	diags.Append(modelDiags...)
	if modelDiags.HasError() || len(models) == 0 {
		return nil
	}
	if len(models) != 1 {
		diags.AddError("Invalid Agent Memory configuration", "Only one built_in_detectors block can be configured.")
		return nil
	}
	return &models[0]
}

func builtInDetectorsList(ctx context.Context, model *AgentMemoryBuiltInDetectorsModel, diags *diag.Diagnostics) BuiltInDetectorsListValue {
	models := make([]AgentMemoryBuiltInDetectorsModel, 0, 1)
	if model != nil {
		models = append(models, *model)
	}
	list, listDiags := NewBuiltInDetectorsList(ctx, models)
	diags.Append(listDiags...)
	return list
}

var (
	_ basetypes.ListTypable  = CustomDetectorsListType{}
	_ basetypes.ListValuable = CustomDetectorsListValue{}
)

type CustomDetectorsListType struct{ basetypes.ListType }

func NewCustomDetectorsListType() CustomDetectorsListType {
	return CustomDetectorsListType{ListType: basetypes.ListType{ElemType: customDetectorsElementType()}}
}

func customDetectorsElementType() types.ObjectType {
	return types.ObjectType{AttrTypes: customtypes.AttrTypesOf(AgentMemoryCustomDetectorsModel{})}
}

func (t CustomDetectorsListType) Equal(other attr.Type) bool {
	otherType, ok := other.(CustomDetectorsListType)
	return ok && t.ListType.Equal(otherType.ListType)
}

func (t CustomDetectorsListType) String() string {
	return "rediscloud.agentmemory.CustomDetectorsListType"
}

func (t CustomDetectorsListType) ValueFromList(_ context.Context, value basetypes.ListValue) (basetypes.ListValuable, diag.Diagnostics) {
	return CustomDetectorsListValue{ListValue: value}, nil
}

func (t CustomDetectorsListType) ValueFromTerraform(ctx context.Context, value tftypes.Value) (attr.Value, error) {
	converted, err := t.ListType.ValueFromTerraform(ctx, value)
	if err != nil {
		return nil, err
	}
	list, ok := converted.(basetypes.ListValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T, expected basetypes.ListValue", converted)
	}
	return CustomDetectorsListValue{ListValue: list}, nil
}

func (t CustomDetectorsListType) ValueType(context.Context) attr.Value {
	return CustomDetectorsListValue{}
}

type CustomDetectorsListValue struct{ basetypes.ListValue }

func (v CustomDetectorsListValue) Equal(other attr.Value) bool {
	otherValue, ok := other.(CustomDetectorsListValue)
	return ok && v.ListValue.Equal(otherValue.ListValue)
}

func (v CustomDetectorsListValue) Type(context.Context) attr.Type {
	return NewCustomDetectorsListType()
}

func (v CustomDetectorsListValue) ToTerraformValue(ctx context.Context) (tftypes.Value, error) {
	if v.IsNull() {
		return tftypes.NewValue(v.Type(ctx).TerraformType(ctx), nil), nil
	}
	return v.ListValue.ToTerraformValue(ctx)
}

func (v CustomDetectorsListValue) AsModels(ctx context.Context) ([]AgentMemoryCustomDetectorsModel, diag.Diagnostics) {
	var models []AgentMemoryCustomDetectorsModel
	diags := v.ElementsAs(ctx, &models, false)
	return models, diags
}

func NewCustomDetectorsList(ctx context.Context, models []AgentMemoryCustomDetectorsModel) (CustomDetectorsListValue, diag.Diagnostics) {
	list, diags := types.ListValueFrom(ctx, customDetectorsElementType(), models)
	return CustomDetectorsListValue{ListValue: list}, diags
}

func customDetectorsBlock(ctx context.Context, list CustomDetectorsListValue, diags *diag.Diagnostics) *AgentMemoryCustomDetectorsModel {
	if list.IsNull() {
		return nil
	}
	if list.IsUnknown() {
		diags.AddError("Unknown Agent Memory configuration", "The custom_detectors block must be known before its values can be used.")
		return nil
	}
	models, modelDiags := list.AsModels(ctx)
	diags.Append(modelDiags...)
	if modelDiags.HasError() || len(models) == 0 {
		return nil
	}
	if len(models) != 1 {
		diags.AddError("Invalid Agent Memory configuration", "Only one custom_detectors block can be configured.")
		return nil
	}
	return &models[0]
}

func customDetectorsList(ctx context.Context, model *AgentMemoryCustomDetectorsModel, diags *diag.Diagnostics) CustomDetectorsListValue {
	models := make([]AgentMemoryCustomDetectorsModel, 0, 1)
	if model != nil {
		models = append(models, *model)
	}
	list, listDiags := NewCustomDetectorsList(ctx, models)
	diags.Append(listDiags...)
	return list
}

var (
	_ basetypes.ListTypable  = MatcherSpecListType{}
	_ basetypes.ListValuable = MatcherSpecListValue{}
)

type MatcherSpecListType struct{ basetypes.ListType }

func NewMatcherSpecListType() MatcherSpecListType {
	return MatcherSpecListType{ListType: basetypes.ListType{ElemType: matcherSpecElementType()}}
}

func matcherSpecElementType() types.ObjectType {
	return types.ObjectType{AttrTypes: customtypes.AttrTypesOf(AgentMemoryMatcherSpecModel{})}
}

func (t MatcherSpecListType) Equal(other attr.Type) bool {
	otherType, ok := other.(MatcherSpecListType)
	return ok && t.ListType.Equal(otherType.ListType)
}

func (t MatcherSpecListType) String() string { return "rediscloud.agentmemory.MatcherSpecListType" }

func (t MatcherSpecListType) ValueFromList(_ context.Context, value basetypes.ListValue) (basetypes.ListValuable, diag.Diagnostics) {
	return MatcherSpecListValue{ListValue: value}, nil
}

func (t MatcherSpecListType) ValueFromTerraform(ctx context.Context, value tftypes.Value) (attr.Value, error) {
	converted, err := t.ListType.ValueFromTerraform(ctx, value)
	if err != nil {
		return nil, err
	}
	list, ok := converted.(basetypes.ListValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T, expected basetypes.ListValue", converted)
	}
	return MatcherSpecListValue{ListValue: list}, nil
}

func (t MatcherSpecListType) ValueType(context.Context) attr.Value { return MatcherSpecListValue{} }

type MatcherSpecListValue struct{ basetypes.ListValue }

func (v MatcherSpecListValue) Equal(other attr.Value) bool {
	otherValue, ok := other.(MatcherSpecListValue)
	return ok && v.ListValue.Equal(otherValue.ListValue)
}

func (v MatcherSpecListValue) Type(context.Context) attr.Type { return NewMatcherSpecListType() }

func (v MatcherSpecListValue) ToTerraformValue(ctx context.Context) (tftypes.Value, error) {
	if v.IsNull() {
		return tftypes.NewValue(v.Type(ctx).TerraformType(ctx), nil), nil
	}
	return v.ListValue.ToTerraformValue(ctx)
}

func (v MatcherSpecListValue) AsModels(ctx context.Context) ([]AgentMemoryMatcherSpecModel, diag.Diagnostics) {
	var models []AgentMemoryMatcherSpecModel
	diags := v.ElementsAs(ctx, &models, false)
	return models, diags
}

func NewMatcherSpecList(ctx context.Context, models []AgentMemoryMatcherSpecModel) (MatcherSpecListValue, diag.Diagnostics) {
	list, diags := types.ListValueFrom(ctx, matcherSpecElementType(), models)
	return MatcherSpecListValue{ListValue: list}, diags
}

func matcherSpecBlock(ctx context.Context, list MatcherSpecListValue, diags *diag.Diagnostics) *AgentMemoryMatcherSpecModel {
	if list.IsNull() {
		return nil
	}
	if list.IsUnknown() {
		diags.AddError("Unknown Agent Memory configuration", "The matcher block must be known before its values can be used.")
		return nil
	}
	models, modelDiags := list.AsModels(ctx)
	diags.Append(modelDiags...)
	if modelDiags.HasError() || len(models) == 0 {
		return nil
	}
	if len(models) != 1 {
		diags.AddError("Invalid Agent Memory configuration", "Only one matcher block can be configured.")
		return nil
	}
	return &models[0]
}

func matcherSpecList(ctx context.Context, model *AgentMemoryMatcherSpecModel, diags *diag.Diagnostics) MatcherSpecListValue {
	models := make([]AgentMemoryMatcherSpecModel, 0, 1)
	if model != nil {
		models = append(models, *model)
	}
	list, listDiags := NewMatcherSpecList(ctx, models)
	diags.Append(listDiags...)
	return list
}

var (
	_ basetypes.ListTypable  = RegexSpecListType{}
	_ basetypes.ListValuable = RegexSpecListValue{}
)

type RegexSpecListType struct{ basetypes.ListType }

func NewRegexSpecListType() RegexSpecListType {
	return RegexSpecListType{ListType: basetypes.ListType{ElemType: regexSpecElementType()}}
}

func regexSpecElementType() types.ObjectType {
	return types.ObjectType{AttrTypes: customtypes.AttrTypesOf(AgentMemoryRegexSpecModel{})}
}

func (t RegexSpecListType) Equal(other attr.Type) bool {
	otherType, ok := other.(RegexSpecListType)
	return ok && t.ListType.Equal(otherType.ListType)
}

func (t RegexSpecListType) String() string { return "rediscloud.agentmemory.RegexSpecListType" }

func (t RegexSpecListType) ValueFromList(_ context.Context, value basetypes.ListValue) (basetypes.ListValuable, diag.Diagnostics) {
	return RegexSpecListValue{ListValue: value}, nil
}

func (t RegexSpecListType) ValueFromTerraform(ctx context.Context, value tftypes.Value) (attr.Value, error) {
	converted, err := t.ListType.ValueFromTerraform(ctx, value)
	if err != nil {
		return nil, err
	}
	list, ok := converted.(basetypes.ListValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T, expected basetypes.ListValue", converted)
	}
	return RegexSpecListValue{ListValue: list}, nil
}

func (t RegexSpecListType) ValueType(context.Context) attr.Value { return RegexSpecListValue{} }

type RegexSpecListValue struct{ basetypes.ListValue }

func (v RegexSpecListValue) Equal(other attr.Value) bool {
	otherValue, ok := other.(RegexSpecListValue)
	return ok && v.ListValue.Equal(otherValue.ListValue)
}

func (v RegexSpecListValue) Type(context.Context) attr.Type { return NewRegexSpecListType() }

func (v RegexSpecListValue) ToTerraformValue(ctx context.Context) (tftypes.Value, error) {
	if v.IsNull() {
		return tftypes.NewValue(v.Type(ctx).TerraformType(ctx), nil), nil
	}
	return v.ListValue.ToTerraformValue(ctx)
}

func (v RegexSpecListValue) AsModels(ctx context.Context) ([]AgentMemoryRegexSpecModel, diag.Diagnostics) {
	var models []AgentMemoryRegexSpecModel
	diags := v.ElementsAs(ctx, &models, false)
	return models, diags
}

func NewRegexSpecList(ctx context.Context, models []AgentMemoryRegexSpecModel) (RegexSpecListValue, diag.Diagnostics) {
	list, diags := types.ListValueFrom(ctx, regexSpecElementType(), models)
	return RegexSpecListValue{ListValue: list}, diags
}

func regexSpecBlock(ctx context.Context, list RegexSpecListValue, diags *diag.Diagnostics) *AgentMemoryRegexSpecModel {
	if list.IsNull() {
		return nil
	}
	if list.IsUnknown() {
		diags.AddError("Unknown Agent Memory configuration", "The regex block must be known before its values can be used.")
		return nil
	}
	models, modelDiags := list.AsModels(ctx)
	diags.Append(modelDiags...)
	if modelDiags.HasError() || len(models) == 0 {
		return nil
	}
	if len(models) != 1 {
		diags.AddError("Invalid Agent Memory configuration", "Only one regex block can be configured.")
		return nil
	}
	return &models[0]
}

func regexSpecList(ctx context.Context, model *AgentMemoryRegexSpecModel, diags *diag.Diagnostics) RegexSpecListValue {
	models := make([]AgentMemoryRegexSpecModel, 0, 1)
	if model != nil {
		models = append(models, *model)
	}
	list, listDiags := NewRegexSpecList(ctx, models)
	diags.Append(listDiags...)
	return list
}

var (
	_ basetypes.ListTypable  = CustomExtractionStrategyListType{}
	_ basetypes.ListValuable = CustomExtractionStrategyListValue{}
)

type CustomExtractionStrategyListType struct{ basetypes.ListType }

func NewCustomExtractionStrategyListType() CustomExtractionStrategyListType {
	return CustomExtractionStrategyListType{ListType: basetypes.ListType{ElemType: customExtractionStrategyElementType()}}
}

func customExtractionStrategyElementType() types.ObjectType {
	return types.ObjectType{AttrTypes: customtypes.AttrTypesOf(AgentMemoryCustomExtractionStrategy{})}
}

func (t CustomExtractionStrategyListType) Equal(other attr.Type) bool {
	otherType, ok := other.(CustomExtractionStrategyListType)
	return ok && t.ListType.Equal(otherType.ListType)
}

func (t CustomExtractionStrategyListType) String() string {
	return "rediscloud.agentmemory.CustomExtractionStrategyListType"
}

func (t CustomExtractionStrategyListType) ValueFromList(_ context.Context, value basetypes.ListValue) (basetypes.ListValuable, diag.Diagnostics) {
	return CustomExtractionStrategyListValue{ListValue: value}, nil
}

func (t CustomExtractionStrategyListType) ValueFromTerraform(ctx context.Context, value tftypes.Value) (attr.Value, error) {
	converted, err := t.ListType.ValueFromTerraform(ctx, value)
	if err != nil {
		return nil, err
	}
	list, ok := converted.(basetypes.ListValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T, expected basetypes.ListValue", converted)
	}
	return CustomExtractionStrategyListValue{ListValue: list}, nil
}

func (t CustomExtractionStrategyListType) ValueType(context.Context) attr.Value {
	return CustomExtractionStrategyListValue{}
}

type CustomExtractionStrategyListValue struct{ basetypes.ListValue }

func (v CustomExtractionStrategyListValue) Equal(other attr.Value) bool {
	otherValue, ok := other.(CustomExtractionStrategyListValue)
	return ok && v.ListValue.Equal(otherValue.ListValue)
}

func (v CustomExtractionStrategyListValue) Type(context.Context) attr.Type {
	return NewCustomExtractionStrategyListType()
}

func (v CustomExtractionStrategyListValue) ToTerraformValue(ctx context.Context) (tftypes.Value, error) {
	if v.IsNull() {
		return tftypes.NewValue(v.Type(ctx).TerraformType(ctx), nil), nil
	}
	return v.ListValue.ToTerraformValue(ctx)
}

func (v CustomExtractionStrategyListValue) AsModels(ctx context.Context) ([]AgentMemoryCustomExtractionStrategy, diag.Diagnostics) {
	var models []AgentMemoryCustomExtractionStrategy
	diags := v.ElementsAs(ctx, &models, false)
	return models, diags
}

func NewCustomExtractionStrategyList(ctx context.Context, models []AgentMemoryCustomExtractionStrategy) (CustomExtractionStrategyListValue, diag.Diagnostics) {
	list, diags := types.ListValueFrom(ctx, customExtractionStrategyElementType(), models)
	return CustomExtractionStrategyListValue{ListValue: list}, diags
}

func customExtractionStrategyBlock(ctx context.Context, list CustomExtractionStrategyListValue, diags *diag.Diagnostics) *AgentMemoryCustomExtractionStrategy {
	if list.IsNull() {
		return nil
	}
	if list.IsUnknown() {
		diags.AddError("Unknown Agent Memory configuration", "The extraction_strategy block must be known before its values can be used.")
		return nil
	}
	models, modelDiags := list.AsModels(ctx)
	diags.Append(modelDiags...)
	if modelDiags.HasError() || len(models) == 0 {
		return nil
	}
	if len(models) != 1 {
		diags.AddError("Invalid Agent Memory configuration", "Only one extraction_strategy block can be configured.")
		return nil
	}
	return &models[0]
}

func customExtractionStrategyList(ctx context.Context, model *AgentMemoryCustomExtractionStrategy, diags *diag.Diagnostics) CustomExtractionStrategyListValue {
	models := make([]AgentMemoryCustomExtractionStrategy, 0, 1)
	if model != nil {
		models = append(models, *model)
	}
	list, listDiags := NewCustomExtractionStrategyList(ctx, models)
	diags.Append(listDiags...)
	return list
}
