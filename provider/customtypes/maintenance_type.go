package customtypes

import (
	"context"
	"fmt"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/maintenance"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// Compile-time assertions keep the custom list types and values aligned with the Framework interfaces.
var (
	_ basetypes.ListTypable  = MaintenanceWindowListType{}
	_ basetypes.ListValuable = MaintenanceWindowListValue{}
	_ basetypes.ListTypable  = MaintenanceListType{}
	_ basetypes.ListValuable = MaintenanceListValue{}
)

// MaintenanceWindowModel represents a window within a maintenance configuration.
type MaintenanceWindowModel struct {
	StartHour       types.Int64 `tfsdk:"start_hour"`
	DurationInHours types.Int64 `tfsdk:"duration_in_hours"`
	Days            types.List  `tfsdk:"days"`
}

// MaintenanceModel represents the maintenance configuration used by subscriptions.
type MaintenanceModel struct {
	Mode   types.String               `tfsdk:"mode"`
	Window MaintenanceWindowListValue `tfsdk:"window"`
}

// maintenanceWindowAttrTypes defines the Terraform attribute types for one maintenance window.
// Seed Days with its element type because an uninitialised list cannot describe its elements.
var maintenanceWindowAttrTypes = AttrTypesOf(MaintenanceWindowModel{
	Days: types.ListNull(types.StringType),
})

// maintenanceWindowElemType is the object type used for each entry in the nested window list.
var maintenanceWindowElemType = types.ObjectType{AttrTypes: maintenanceWindowAttrTypes}

// maintenanceAttrTypes defines the Terraform attribute types for the maintenance block.
var maintenanceAttrTypes = AttrTypesOf(MaintenanceModel{})

// maintenanceElemType is the object type used for the entry in the outer maintenance list.
var maintenanceElemType = types.ObjectType{AttrTypes: maintenanceAttrTypes}

// MaintenanceWindowListType gives maintenance window lists their provider-specific value type.
type MaintenanceWindowListType struct {
	basetypes.ListType
}

// NewMaintenanceWindowListType returns the custom type for a maintenance window list.
func NewMaintenanceWindowListType() MaintenanceWindowListType {
	return MaintenanceWindowListType{ListType: basetypes.ListType{ElemType: maintenanceWindowElemType}}
}

func (t MaintenanceWindowListType) Equal(other attr.Type) bool {
	otherType, ok := other.(MaintenanceWindowListType)
	if !ok {
		return false
	}
	return t.ListType.Equal(otherType.ListType)
}

func (t MaintenanceWindowListType) String() string {
	return "rediscloud.customtypes.MaintenanceWindowListType"
}

func (t MaintenanceWindowListType) ValueFromList(_ context.Context, value basetypes.ListValue) (basetypes.ListValuable, diag.Diagnostics) {
	return MaintenanceWindowListValue{ListValue: value}, nil
}

func (t MaintenanceWindowListType) ValueFromTerraform(ctx context.Context, value tftypes.Value) (attr.Value, error) {
	terraformValue, err := t.ListType.ValueFromTerraform(ctx, value)
	if err != nil {
		return nil, err
	}

	listValue, ok := terraformValue.(basetypes.ListValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T, expected basetypes.ListValue", terraformValue)
	}

	return MaintenanceWindowListValue{ListValue: listValue}, nil
}

func (t MaintenanceWindowListType) ValueType(context.Context) attr.Value {
	return MaintenanceWindowListValue{}
}

// MaintenanceWindowListValue stores a list of maintenance window definitions.
type MaintenanceWindowListValue struct {
	basetypes.ListValue
}

func (v MaintenanceWindowListValue) Equal(other attr.Value) bool {
	otherValue, ok := other.(MaintenanceWindowListValue)
	if !ok {
		return false
	}
	return v.ListValue.Equal(otherValue.ListValue)
}

func (v MaintenanceWindowListValue) Type(context.Context) attr.Type {
	return NewMaintenanceWindowListType()
}

// AsModels decodes each element of a known maintenance window list into its typed model.
// Callers must handle null and unknown values first because a model slice cannot
// represent either Terraform state without losing information.
func (v MaintenanceWindowListValue) AsModels(ctx context.Context) ([]MaintenanceWindowModel, diag.Diagnostics) {
	var models []MaintenanceWindowModel
	diags := v.ElementsAs(ctx, &models, false)
	return models, diags
}

// MaintenanceListType gives maintenance lists their provider-specific value type.
type MaintenanceListType struct {
	basetypes.ListType
}

// NewMaintenanceListType returns the custom type for a subscription maintenance list.
func NewMaintenanceListType() MaintenanceListType {
	return MaintenanceListType{ListType: basetypes.ListType{ElemType: maintenanceElemType}}
}

func (t MaintenanceListType) Equal(other attr.Type) bool {
	otherType, ok := other.(MaintenanceListType)
	if !ok {
		return false
	}
	return t.ListType.Equal(otherType.ListType)
}

func (t MaintenanceListType) String() string {
	return "rediscloud.customtypes.MaintenanceListType"
}

func (t MaintenanceListType) ValueFromList(_ context.Context, value basetypes.ListValue) (basetypes.ListValuable, diag.Diagnostics) {
	return MaintenanceListValue{ListValue: value}, nil
}

func (t MaintenanceListType) ValueFromTerraform(ctx context.Context, value tftypes.Value) (attr.Value, error) {
	terraformValue, err := t.ListType.ValueFromTerraform(ctx, value)
	if err != nil {
		return nil, err
	}

	listValue, ok := terraformValue.(basetypes.ListValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T, expected basetypes.ListValue", terraformValue)
	}

	return MaintenanceListValue{ListValue: listValue}, nil
}

func (t MaintenanceListType) ValueType(context.Context) attr.Value {
	return MaintenanceListValue{}
}

// MaintenanceListValue stores a subscription maintenance list.
type MaintenanceListValue struct {
	basetypes.ListValue
}

func (v MaintenanceListValue) Equal(other attr.Value) bool {
	otherValue, ok := other.(MaintenanceListValue)
	if !ok {
		return false
	}
	return v.ListValue.Equal(otherValue.ListValue)
}

func (v MaintenanceListValue) Type(context.Context) attr.Type {
	return NewMaintenanceListType()
}

// AsModels decodes each element of a known maintenance list into its typed model.
// Callers must handle null and unknown values first because a model slice cannot
// represent either Terraform state without losing information.
func (v MaintenanceListValue) AsModels(ctx context.Context) ([]MaintenanceModel, diag.Diagnostics) {
	var models []MaintenanceModel
	diags := v.ElementsAs(ctx, &models, false)
	return models, diags
}

// NewMaintenanceList converts an API maintenance response into a reusable custom value.
func NewMaintenanceList(ctx context.Context, apiMaintenance *maintenance.Maintenance) (MaintenanceListValue, diag.Diagnostics) {
	if apiMaintenance == nil {
		return MaintenanceListValue{ListValue: types.ListNull(maintenanceElemType)}, nil
	}

	var diags diag.Diagnostics
	windows := make([]MaintenanceWindowModel, 0, len(apiMaintenance.Windows))
	for _, apiWindow := range apiMaintenance.Windows {
		days, daysDiags := types.ListValueFrom(ctx, types.StringType, apiWindow.Days)
		diags.Append(daysDiags...)
		if diags.HasError() {
			return nullMaintenanceListValue(), diags
		}

		windows = append(windows, MaintenanceWindowModel{
			StartHour:       types.Int64Value(int64(redis.IntValue(apiWindow.StartHour))),
			DurationInHours: types.Int64Value(int64(redis.IntValue(apiWindow.DurationInHours))),
			Days:            days,
		})
	}

	windowList, windowListDiags := types.ListValueFrom(ctx, maintenanceWindowElemType, windows)
	diags.Append(windowListDiags...)
	if diags.HasError() {
		return nullMaintenanceListValue(), diags
	}

	list, listDiags := types.ListValueFrom(ctx, maintenanceElemType, []MaintenanceModel{
		{
			Mode:   types.StringValue(redis.StringValue(apiMaintenance.Mode)),
			Window: MaintenanceWindowListValue{ListValue: windowList},
		},
	})
	diags.Append(listDiags...)
	if diags.HasError() {
		return nullMaintenanceListValue(), diags
	}

	return MaintenanceListValue{ListValue: list}, diags
}

func nullMaintenanceListValue() MaintenanceListValue {
	return MaintenanceListValue{ListValue: types.ListNull(maintenanceElemType)}
}
