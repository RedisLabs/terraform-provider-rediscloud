package pro

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
	_ basetypes.SetTypable   = AlertSetType{}
	_ basetypes.SetValuable  = AlertSetValue{}
	_ basetypes.SetTypable   = ModuleSetType{}
	_ basetypes.SetValuable  = ModuleSetValue{}
	_ basetypes.ListTypable  = RemoteBackupListType{}
	_ basetypes.ListValuable = RemoteBackupListValue{}
)

var alertElementType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(AlertModel{})}

// AlertSetType is the custom set type for the alert block.
type AlertSetType struct{ basetypes.SetType }

// NewAlertSetType returns the custom type used by the alert block.
func NewAlertSetType() AlertSetType {
	return AlertSetType{SetType: basetypes.SetType{ElemType: alertElementType}}
}

func (t AlertSetType) Equal(other attr.Type) bool {
	otherType, ok := other.(AlertSetType)
	return ok && t.SetType.Equal(otherType.SetType)
}

func (t AlertSetType) String() string { return "rediscloud.pro.AlertSetType" }

func (t AlertSetType) ValueFromSet(_ context.Context, value basetypes.SetValue) (basetypes.SetValuable, diag.Diagnostics) {
	return AlertSetValue{SetValue: value}, nil
}

func (t AlertSetType) ValueFromTerraform(ctx context.Context, value tftypes.Value) (attr.Value, error) {
	converted, err := t.SetType.ValueFromTerraform(ctx, value)
	if err != nil {
		return nil, err
	}
	set, ok := converted.(basetypes.SetValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T, expected basetypes.SetValue", converted)
	}
	return AlertSetValue{SetValue: set}, nil
}

func (t AlertSetType) ValueType(context.Context) attr.Value { return AlertSetValue{} }

// AlertSetValue is a set of alert blocks.
type AlertSetValue struct{ basetypes.SetValue }

func (v AlertSetValue) Equal(other attr.Value) bool {
	otherValue, ok := other.(AlertSetValue)
	return ok && v.SetValue.Equal(otherValue.SetValue)
}

func (v AlertSetValue) Type(context.Context) attr.Type { return NewAlertSetType() }

func (v AlertSetValue) ToTerraformValue(ctx context.Context) (tftypes.Value, error) {
	if v.IsNull() {
		return tftypes.NewValue(v.Type(ctx).TerraformType(ctx), nil), nil
	}
	return v.SetValue.ToTerraformValue(ctx)
}

// AsModels decodes a known alert set. Callers check null and unknown first.
func (v AlertSetValue) AsModels(ctx context.Context) ([]AlertModel, diag.Diagnostics) {
	var models []AlertModel
	diags := v.ElementsAs(ctx, &models, false)
	return models, diags
}

// NewAlertSet builds a known alert set. A nil slice becomes an empty set.
func NewAlertSet(ctx context.Context, models []AlertModel) (AlertSetValue, diag.Diagnostics) {
	if models == nil {
		models = make([]AlertModel, 0)
	}
	set, diags := types.SetValueFrom(ctx, alertElementType, models)
	return AlertSetValue{SetValue: set}, diags
}

func nullAlertSet() AlertSetValue {
	return AlertSetValue{SetValue: types.SetNull(alertElementType)}
}

var moduleElementType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(ModuleModel{})}

// ModuleSetType is the custom set type for the modules block.
type ModuleSetType struct{ basetypes.SetType }

// NewModuleSetType returns the custom type used by the modules block.
func NewModuleSetType() ModuleSetType {
	return ModuleSetType{SetType: basetypes.SetType{ElemType: moduleElementType}}
}

func (t ModuleSetType) Equal(other attr.Type) bool {
	otherType, ok := other.(ModuleSetType)
	return ok && t.SetType.Equal(otherType.SetType)
}

func (t ModuleSetType) String() string { return "rediscloud.pro.ModuleSetType" }

func (t ModuleSetType) ValueFromSet(_ context.Context, value basetypes.SetValue) (basetypes.SetValuable, diag.Diagnostics) {
	return ModuleSetValue{SetValue: value}, nil
}

func (t ModuleSetType) ValueFromTerraform(ctx context.Context, value tftypes.Value) (attr.Value, error) {
	converted, err := t.SetType.ValueFromTerraform(ctx, value)
	if err != nil {
		return nil, err
	}
	set, ok := converted.(basetypes.SetValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T, expected basetypes.SetValue", converted)
	}
	return ModuleSetValue{SetValue: set}, nil
}

func (t ModuleSetType) ValueType(context.Context) attr.Value { return ModuleSetValue{} }

// ModuleSetValue is a set of module blocks.
type ModuleSetValue struct{ basetypes.SetValue }

func (v ModuleSetValue) Equal(other attr.Value) bool {
	otherValue, ok := other.(ModuleSetValue)
	return ok && v.SetValue.Equal(otherValue.SetValue)
}

func (v ModuleSetValue) Type(context.Context) attr.Type { return NewModuleSetType() }

func (v ModuleSetValue) ToTerraformValue(ctx context.Context) (tftypes.Value, error) {
	if v.IsNull() {
		return tftypes.NewValue(v.Type(ctx).TerraformType(ctx), nil), nil
	}
	return v.SetValue.ToTerraformValue(ctx)
}

// AsModels decodes a known module set. Callers check null and unknown first.
func (v ModuleSetValue) AsModels(ctx context.Context) ([]ModuleModel, diag.Diagnostics) {
	var models []ModuleModel
	diags := v.ElementsAs(ctx, &models, false)
	return models, diags
}

// NewModuleSet builds a known module set. A nil slice becomes an empty set.
func NewModuleSet(ctx context.Context, models []ModuleModel) (ModuleSetValue, diag.Diagnostics) {
	if models == nil {
		models = make([]ModuleModel, 0)
	}
	set, diags := types.SetValueFrom(ctx, moduleElementType, models)
	return ModuleSetValue{SetValue: set}, diags
}

func nullModuleSet() ModuleSetValue {
	return ModuleSetValue{SetValue: types.SetNull(moduleElementType)}
}

var remoteBackupElementType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(RemoteBackupModel{})}

// RemoteBackupListType is the custom list type for the remote_backup block.
type RemoteBackupListType struct{ basetypes.ListType }

// NewRemoteBackupListType returns the custom type used by the remote_backup block.
func NewRemoteBackupListType() RemoteBackupListType {
	return RemoteBackupListType{ListType: basetypes.ListType{ElemType: remoteBackupElementType}}
}

func (t RemoteBackupListType) Equal(other attr.Type) bool {
	otherType, ok := other.(RemoteBackupListType)
	return ok && t.ListType.Equal(otherType.ListType)
}

func (t RemoteBackupListType) String() string { return "rediscloud.pro.RemoteBackupListType" }

func (t RemoteBackupListType) ValueFromList(_ context.Context, value basetypes.ListValue) (basetypes.ListValuable, diag.Diagnostics) {
	return RemoteBackupListValue{ListValue: value}, nil
}

func (t RemoteBackupListType) ValueFromTerraform(ctx context.Context, value tftypes.Value) (attr.Value, error) {
	converted, err := t.ListType.ValueFromTerraform(ctx, value)
	if err != nil {
		return nil, err
	}
	list, ok := converted.(basetypes.ListValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T, expected basetypes.ListValue", converted)
	}
	return RemoteBackupListValue{ListValue: list}, nil
}

func (t RemoteBackupListType) ValueType(context.Context) attr.Value { return RemoteBackupListValue{} }

// RemoteBackupListValue is a list of remote_backup blocks.
type RemoteBackupListValue struct{ basetypes.ListValue }

func (v RemoteBackupListValue) Equal(other attr.Value) bool {
	otherValue, ok := other.(RemoteBackupListValue)
	return ok && v.ListValue.Equal(otherValue.ListValue)
}

func (v RemoteBackupListValue) Type(context.Context) attr.Type { return NewRemoteBackupListType() }

func (v RemoteBackupListValue) ToTerraformValue(ctx context.Context) (tftypes.Value, error) {
	if v.IsNull() {
		return tftypes.NewValue(v.Type(ctx).TerraformType(ctx), nil), nil
	}
	return v.ListValue.ToTerraformValue(ctx)
}

// AsModels decodes a known remote_backup list. Callers check null and unknown first.
func (v RemoteBackupListValue) AsModels(ctx context.Context) ([]RemoteBackupModel, diag.Diagnostics) {
	var models []RemoteBackupModel
	diags := v.ElementsAs(ctx, &models, false)
	return models, diags
}

// NewRemoteBackupList builds a known remote_backup list. A nil slice becomes an empty list.
func NewRemoteBackupList(ctx context.Context, models []RemoteBackupModel) (RemoteBackupListValue, diag.Diagnostics) {
	if models == nil {
		models = make([]RemoteBackupModel, 0)
	}
	list, diags := types.ListValueFrom(ctx, remoteBackupElementType, models)
	return RemoteBackupListValue{ListValue: list}, diags
}

func nullRemoteBackupList() RemoteBackupListValue {
	return RemoteBackupListValue{ListValue: types.ListNull(remoteBackupElementType)}
}

func emptyRemoteBackupList(ctx context.Context) (RemoteBackupListValue, diag.Diagnostics) {
	return NewRemoteBackupList(ctx, []RemoteBackupModel{})
}
