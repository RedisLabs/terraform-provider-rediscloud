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

// Customer-managed keys are configuration on the subscription resource. The list is
// not returned by the read API, so the resource keeps the configured value in state.

var (
	_ basetypes.ListTypable  = CustomerManagedKeyListType{}
	_ basetypes.ListValuable = CustomerManagedKeyListValue{}
)

// CustomerManagedKeyModel is one customer-managed key block.
type CustomerManagedKeyModel struct {
	ResourceName types.String `tfsdk:"resource_name"`
	Region       types.String `tfsdk:"region"`
}

var customerManagedKeyAttrTypes = customtypes.AttrTypesOf(CustomerManagedKeyModel{})

var customerManagedKeyElemType = types.ObjectType{AttrTypes: customerManagedKeyAttrTypes}

// CustomerManagedKeyListType is the custom list type for customer-managed keys.
type CustomerManagedKeyListType struct {
	basetypes.ListType
}

// NewCustomerManagedKeyListType returns the custom type for customer-managed keys.
func NewCustomerManagedKeyListType() CustomerManagedKeyListType {
	return CustomerManagedKeyListType{ListType: basetypes.ListType{ElemType: customerManagedKeyElemType}}
}

func (t CustomerManagedKeyListType) Equal(other attr.Type) bool {
	otherType, ok := other.(CustomerManagedKeyListType)
	if !ok {
		return false
	}
	return t.ListType.Equal(otherType.ListType)
}

func (t CustomerManagedKeyListType) String() string {
	return "rediscloud.activeactive.CustomerManagedKeyListType"
}

func (t CustomerManagedKeyListType) ValueFromList(_ context.Context, value basetypes.ListValue) (basetypes.ListValuable, diag.Diagnostics) {
	return CustomerManagedKeyListValue{ListValue: value}, nil
}

func (t CustomerManagedKeyListType) ValueFromTerraform(ctx context.Context, value tftypes.Value) (attr.Value, error) {
	terraformValue, err := t.ListType.ValueFromTerraform(ctx, value)
	if err != nil {
		return nil, err
	}

	listValue, ok := terraformValue.(basetypes.ListValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T, expected basetypes.ListValue", terraformValue)
	}

	return CustomerManagedKeyListValue{ListValue: listValue}, nil
}

func (t CustomerManagedKeyListType) ValueType(context.Context) attr.Value {
	return CustomerManagedKeyListValue{}
}

// CustomerManagedKeyListValue stores customer-managed keys.
type CustomerManagedKeyListValue struct {
	basetypes.ListValue
}

func (v CustomerManagedKeyListValue) Equal(other attr.Value) bool {
	otherValue, ok := other.(CustomerManagedKeyListValue)
	if !ok {
		return false
	}
	return v.ListValue.Equal(otherValue.ListValue)
}

func (v CustomerManagedKeyListValue) Type(context.Context) attr.Type {
	return NewCustomerManagedKeyListType()
}

// AsModels decodes a known key list. Callers must handle null and unknown first.
func (v CustomerManagedKeyListValue) AsModels(ctx context.Context) ([]CustomerManagedKeyModel, diag.Diagnostics) {
	var models []CustomerManagedKeyModel
	diags := v.ElementsAs(ctx, &models, false)
	return models, diags
}

func nullCustomerManagedKeyListValue() CustomerManagedKeyListValue {
	return CustomerManagedKeyListValue{ListValue: types.ListNull(customerManagedKeyElemType)}
}

func unknownCustomerManagedKeyListValue() CustomerManagedKeyListValue {
	return CustomerManagedKeyListValue{ListValue: types.ListUnknown(customerManagedKeyElemType)}
}

// NewCustomerManagedKeyListValue builds a known key list. A nil slice stays null.
func NewCustomerManagedKeyListValue(ctx context.Context, models []CustomerManagedKeyModel) (CustomerManagedKeyListValue, diag.Diagnostics) {
	if models == nil {
		return nullCustomerManagedKeyListValue(), nil
	}
	list, diags := types.ListValueFrom(ctx, customerManagedKeyElemType, models)
	return CustomerManagedKeyListValue{ListValue: list}, diags
}
