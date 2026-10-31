package agentmemory

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const (
	agentMemoryMutationDefaultTimeout = 30 * time.Minute
	agentMemoryReadDefaultTimeout     = 5 * time.Minute
	agentMemoryAPIKeyDefaultTimeout   = 5 * time.Minute
)

var (
	_ basetypes.ListTypable  = MemoryTimeoutsListType{}
	_ basetypes.ListValuable = MemoryTimeoutsListValue{}
)

type MemoryTimeoutsListType struct{ basetypes.ListType }

func memoryTimeoutsElementType() timeouts.Type {
	return timeouts.Block(context.Background(), timeouts.Opts{Create: true, Read: true, Update: true, Delete: true}).Type().(timeouts.Type)
}
func NewMemoryTimeoutsListType() MemoryTimeoutsListType {
	return MemoryTimeoutsListType{ListType: basetypes.ListType{ElemType: memoryTimeoutsElementType()}}
}
func (t MemoryTimeoutsListType) Equal(other attr.Type) bool {
	otherType, ok := other.(MemoryTimeoutsListType)
	return ok && t.ListType.Equal(otherType.ListType)
}
func (t MemoryTimeoutsListType) String() string {
	return "rediscloud.agentmemory.MemoryTimeoutsListType"
}
func (t MemoryTimeoutsListType) ValueFromList(_ context.Context, v basetypes.ListValue) (basetypes.ListValuable, diag.Diagnostics) {
	return MemoryTimeoutsListValue{ListValue: v}, nil
}
func (t MemoryTimeoutsListType) ValueFromTerraform(ctx context.Context, v tftypes.Value) (attr.Value, error) {
	value, err := t.ListType.ValueFromTerraform(ctx, v)
	if err != nil {
		return nil, err
	}
	list, ok := value.(basetypes.ListValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T, expected basetypes.ListValue", value)
	}
	return MemoryTimeoutsListValue{ListValue: list}, nil
}
func (t MemoryTimeoutsListType) ValueType(context.Context) attr.Value {
	return MemoryTimeoutsListValue{}
}

type MemoryTimeoutsListValue struct{ basetypes.ListValue }

func (v MemoryTimeoutsListValue) Equal(other attr.Value) bool {
	otherValue, ok := other.(MemoryTimeoutsListValue)
	return ok && v.ListValue.Equal(otherValue.ListValue)
}
func (v MemoryTimeoutsListValue) Type(context.Context) attr.Type { return NewMemoryTimeoutsListType() }
func (v MemoryTimeoutsListValue) ToTerraformValue(ctx context.Context) (tftypes.Value, error) {
	if v.IsNull() {
		return tftypes.NewValue(v.Type(ctx).TerraformType(ctx), nil), nil
	}
	return v.ListValue.ToTerraformValue(ctx)
}
func (v MemoryTimeoutsListValue) AsModels(ctx context.Context) ([]timeouts.Value, diag.Diagnostics) {
	var models []timeouts.Value
	diags := v.ElementsAs(ctx, &models, false)
	return models, diags
}
func NewMemoryTimeoutsList(ctx context.Context, models []timeouts.Value) (MemoryTimeoutsListValue, diag.Diagnostics) {
	list, diags := types.ListValueFrom(ctx, memoryTimeoutsElementType(), models)
	return MemoryTimeoutsListValue{ListValue: list}, diags
}
func memoryTimeoutsBlock(ctx context.Context) schema.Block {
	block := timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true}).(schema.SingleNestedBlock)
	return schema.ListNestedBlock{
		CustomType:   NewMemoryTimeoutsListType(),
		Validators:   []validator.List{listvalidator.SizeAtMost(1)},
		NestedObject: schema.NestedBlockObject{Attributes: block.Attributes, CustomType: block.CustomType},
	}
}

func (v MemoryTimeoutsListValue) Create(ctx context.Context, fallback time.Duration) (time.Duration, diag.Diagnostics) {
	return timeoutDuration(ctx, v.ListValue, memoryTimeoutsElementType(), "create", fallback)
}

func (v MemoryTimeoutsListValue) Read(ctx context.Context, fallback time.Duration) (time.Duration, diag.Diagnostics) {
	return timeoutDuration(ctx, v.ListValue, memoryTimeoutsElementType(), "read", fallback)
}

func (v MemoryTimeoutsListValue) Update(ctx context.Context, fallback time.Duration) (time.Duration, diag.Diagnostics) {
	return timeoutDuration(ctx, v.ListValue, memoryTimeoutsElementType(), "update", fallback)
}

func (v MemoryTimeoutsListValue) Delete(ctx context.Context, fallback time.Duration) (time.Duration, diag.Diagnostics) {
	return timeoutDuration(ctx, v.ListValue, memoryTimeoutsElementType(), "delete", fallback)
}

var (
	_ basetypes.ListTypable  = APIKeyTimeoutsListType{}
	_ basetypes.ListValuable = APIKeyTimeoutsListValue{}
)

type APIKeyTimeoutsListType struct{ basetypes.ListType }

func apiKeyTimeoutsElementType() timeouts.Type {
	return timeouts.Block(context.Background(), timeouts.Opts{Create: true, Read: true, Delete: true}).Type().(timeouts.Type)
}
func NewAPIKeyTimeoutsListType() APIKeyTimeoutsListType {
	return APIKeyTimeoutsListType{ListType: basetypes.ListType{ElemType: apiKeyTimeoutsElementType()}}
}
func (t APIKeyTimeoutsListType) Equal(other attr.Type) bool {
	otherType, ok := other.(APIKeyTimeoutsListType)
	return ok && t.ListType.Equal(otherType.ListType)
}
func (t APIKeyTimeoutsListType) String() string {
	return "rediscloud.agentmemory.APIKeyTimeoutsListType"
}
func (t APIKeyTimeoutsListType) ValueFromList(_ context.Context, v basetypes.ListValue) (basetypes.ListValuable, diag.Diagnostics) {
	return APIKeyTimeoutsListValue{ListValue: v}, nil
}
func (t APIKeyTimeoutsListType) ValueFromTerraform(ctx context.Context, v tftypes.Value) (attr.Value, error) {
	value, err := t.ListType.ValueFromTerraform(ctx, v)
	if err != nil {
		return nil, err
	}
	list, ok := value.(basetypes.ListValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T, expected basetypes.ListValue", value)
	}
	return APIKeyTimeoutsListValue{ListValue: list}, nil
}
func (t APIKeyTimeoutsListType) ValueType(context.Context) attr.Value {
	return APIKeyTimeoutsListValue{}
}

type APIKeyTimeoutsListValue struct{ basetypes.ListValue }

func (v APIKeyTimeoutsListValue) Equal(other attr.Value) bool {
	otherValue, ok := other.(APIKeyTimeoutsListValue)
	return ok && v.ListValue.Equal(otherValue.ListValue)
}
func (v APIKeyTimeoutsListValue) Type(context.Context) attr.Type { return NewAPIKeyTimeoutsListType() }
func (v APIKeyTimeoutsListValue) ToTerraformValue(ctx context.Context) (tftypes.Value, error) {
	if v.IsNull() {
		return tftypes.NewValue(v.Type(ctx).TerraformType(ctx), nil), nil
	}
	return v.ListValue.ToTerraformValue(ctx)
}
func (v APIKeyTimeoutsListValue) AsModels(ctx context.Context) ([]timeouts.Value, diag.Diagnostics) {
	var models []timeouts.Value
	diags := v.ElementsAs(ctx, &models, false)
	return models, diags
}
func NewAPIKeyTimeoutsList(ctx context.Context, models []timeouts.Value) (APIKeyTimeoutsListValue, diag.Diagnostics) {
	list, diags := types.ListValueFrom(ctx, apiKeyTimeoutsElementType(), models)
	return APIKeyTimeoutsListValue{ListValue: list}, diags
}
func apiKeyTimeoutsBlock(ctx context.Context) schema.Block {
	block := timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Delete: true}).(schema.SingleNestedBlock)
	return schema.ListNestedBlock{
		CustomType:   NewAPIKeyTimeoutsListType(),
		Validators:   []validator.List{listvalidator.SizeAtMost(1)},
		NestedObject: schema.NestedBlockObject{Attributes: block.Attributes, CustomType: block.CustomType},
	}
}

func (v APIKeyTimeoutsListValue) Create(ctx context.Context, fallback time.Duration) (time.Duration, diag.Diagnostics) {
	return timeoutDuration(ctx, v.ListValue, apiKeyTimeoutsElementType(), "create", fallback)
}

func (v APIKeyTimeoutsListValue) Read(ctx context.Context, fallback time.Duration) (time.Duration, diag.Diagnostics) {
	return timeoutDuration(ctx, v.ListValue, apiKeyTimeoutsElementType(), "read", fallback)
}

func (v APIKeyTimeoutsListValue) Delete(ctx context.Context, fallback time.Duration) (time.Duration, diag.Diagnostics) {
	return timeoutDuration(ctx, v.ListValue, apiKeyTimeoutsElementType(), "delete", fallback)
}

func timeoutDuration(ctx context.Context, list basetypes.ListValue, typ timeouts.Type, operation string, fallback time.Duration) (time.Duration, diag.Diagnostics) {
	value := timeouts.Value{Object: types.ObjectNull(typ.AttrTypes)}
	var diags diag.Diagnostics
	if list.IsUnknown() {
		diags.AddError("Unknown Agent Memory timeout", "The timeouts block must be known before an operation starts.")
		return 0, diags
	}
	if !list.IsNull() {
		var models []timeouts.Value
		diags.Append(list.ElementsAs(ctx, &models, false)...)
		if diags.HasError() {
			return 0, diags
		}
		if len(models) > 1 {
			diags.AddError("Invalid Agent Memory timeouts", "Only one timeouts block can be configured.")
			return 0, diags
		}
		if len(models) == 1 {
			value = models[0]
		}
	}
	switch operation {
	case "create":
		return value.Create(ctx, fallback)
	case "read":
		return value.Read(ctx, fallback)
	case "update":
		return value.Update(ctx, fallback)
	default:
		return value.Delete(ctx, fallback)
	}
}
