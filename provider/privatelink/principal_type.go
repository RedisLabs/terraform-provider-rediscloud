package privatelink

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

var (
	_ basetypes.SetTypable  = PrincipalSetType{}
	_ basetypes.SetValuable = PrincipalSetValue{}
)

// PrincipalModel represents one principal allowed to access a PrivateLink.
type PrincipalModel struct {
	Principal      types.String `tfsdk:"principal"`
	PrincipalType  types.String `tfsdk:"principal_type"`
	PrincipalAlias types.String `tfsdk:"principal_alias"`
}

var principalElemType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(PrincipalModel{})}

// PrincipalSetType gives PrivateLink principal sets their provider-specific value type.
type PrincipalSetType struct {
	basetypes.SetType
}

// NewPrincipalSetType returns the custom type for a PrivateLink principal set.
func NewPrincipalSetType() PrincipalSetType {
	return PrincipalSetType{SetType: basetypes.SetType{ElemType: principalElemType}}
}

func (t PrincipalSetType) Equal(other attr.Type) bool {
	otherType, ok := other.(PrincipalSetType)
	if !ok {
		return false
	}
	return t.SetType.Equal(otherType.SetType)
}

func (t PrincipalSetType) String() string {
	return "rediscloud.privatelink.PrincipalSetType"
}

func (t PrincipalSetType) ValueFromSet(_ context.Context, value basetypes.SetValue) (basetypes.SetValuable, diag.Diagnostics) {
	return PrincipalSetValue{SetValue: value}, nil
}

func (t PrincipalSetType) ValueFromTerraform(ctx context.Context, value tftypes.Value) (attr.Value, error) {
	terraformValue, err := t.SetType.ValueFromTerraform(ctx, value)
	if err != nil {
		return nil, err
	}

	setValue, ok := terraformValue.(basetypes.SetValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T, expected basetypes.SetValue", terraformValue)
	}

	return PrincipalSetValue{SetValue: setValue}, nil
}

func (t PrincipalSetType) ValueType(context.Context) attr.Value {
	return PrincipalSetValue{}
}

// PrincipalSetValue stores a set of PrivateLink principals.
type PrincipalSetValue struct {
	basetypes.SetValue
}

func (v PrincipalSetValue) Equal(other attr.Value) bool {
	otherValue, ok := other.(PrincipalSetValue)
	if !ok {
		return false
	}
	return v.SetValue.Equal(otherValue.SetValue)
}

func (v PrincipalSetValue) Type(context.Context) attr.Type {
	return NewPrincipalSetType()
}

// AsModels decodes a known principal set into typed models.
// Callers must handle null and unknown values first.
func (v PrincipalSetValue) AsModels(ctx context.Context) ([]PrincipalModel, diag.Diagnostics) {
	var models []PrincipalModel
	diags := v.ElementsAs(ctx, &models, false)
	return models, diags
}

// NewPrincipalSet builds a principal set from models, keeping an empty slice as a known empty set.
func NewPrincipalSet(ctx context.Context, models []PrincipalModel) (PrincipalSetValue, diag.Diagnostics) {
	set, diags := types.SetValueFrom(ctx, principalElemType, models)
	if diags.HasError() {
		return NullPrincipalSet(), diags
	}
	return PrincipalSetValue{SetValue: set}, diags
}

// NullPrincipalSet returns a null principal set.
func NullPrincipalSet() PrincipalSetValue {
	return PrincipalSetValue{SetValue: types.SetNull(principalElemType)}
}
