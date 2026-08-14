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

// Compile-time assertions keep the custom list type and value aligned with the Framework interfaces.
var (
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
	Mode   types.String `tfsdk:"mode"`
	Window types.List   `tfsdk:"window"`
}

// maintenanceWindowAttrTypes defines the Terraform attribute types for one maintenance window.
// Seed Days with its element type because an uninitialised list cannot describe its elements.
var maintenanceWindowAttrTypes = AttrTypesOf(MaintenanceWindowModel{
	Days: types.ListNull(types.StringType),
})

// maintenanceWindowElemType is the object type used for each entry in the nested window list.
var maintenanceWindowElemType = types.ObjectType{AttrTypes: maintenanceWindowAttrTypes}

// maintenanceAttrTypes defines the Terraform attribute types for the maintenance block.
// Seed Window with its element type so the derived list type matches the nested block schema.
var maintenanceAttrTypes = AttrTypesOf(MaintenanceModel{
	Window: types.ListNull(maintenanceWindowElemType),
})

// maintenanceElemType is the object type used for the entry in the outer maintenance list.
var maintenanceElemType = types.ObjectType{AttrTypes: maintenanceAttrTypes}

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

// AsModels decodes each known maintenance list element into its typed model.
// Callers should check IsNull and IsUnknown before using this method.
func (v MaintenanceListValue) AsModels(ctx context.Context) ([]MaintenanceModel, diag.Diagnostics) {
	var models []MaintenanceModel
	diags := v.ListValue.ElementsAs(ctx, &models, false)
	return models, diags
}

// NewMaintenanceList converts an API maintenance response into a reusable custom value.
func NewMaintenanceList(ctx context.Context, apiMaintenance *maintenance.Maintenance) (MaintenanceListValue, diag.Diagnostics) {
	if apiMaintenance == nil {
		return MaintenanceListValue{ListValue: types.ListNull(maintenanceElemType)}, nil
	}

	var diags diag.Diagnostics
	windows := make([]attr.Value, 0, len(apiMaintenance.Windows))
	for _, apiWindow := range apiMaintenance.Windows {
		days, daysDiags := types.ListValueFrom(ctx, types.StringType, apiWindow.Days)
		diags.Append(daysDiags...)
		if diags.HasError() {
			return nullMaintenanceListValue(), diags
		}

		window, windowDiags := types.ObjectValueFrom(ctx, maintenanceWindowAttrTypes, MaintenanceWindowModel{
			StartHour:       types.Int64Value(int64(redis.IntValue(apiWindow.StartHour))),
			DurationInHours: types.Int64Value(int64(redis.IntValue(apiWindow.DurationInHours))),
			Days:            days,
		})
		diags.Append(windowDiags...)
		if diags.HasError() {
			return nullMaintenanceListValue(), diags
		}

		windows = append(windows, window)
	}

	windowList, windowListDiags := types.ListValue(maintenanceWindowElemType, windows)
	diags.Append(windowListDiags...)
	if diags.HasError() {
		return nullMaintenanceListValue(), diags
	}

	maintenanceValue, maintenanceDiags := types.ObjectValueFrom(ctx, maintenanceAttrTypes, MaintenanceModel{
		Mode:   types.StringValue(redis.StringValue(apiMaintenance.Mode)),
		Window: windowList,
	})
	diags.Append(maintenanceDiags...)
	if diags.HasError() {
		return nullMaintenanceListValue(), diags
	}

	list, listDiags := types.ListValue(maintenanceElemType, []attr.Value{maintenanceValue})
	diags.Append(listDiags...)
	if diags.HasError() {
		return nullMaintenanceListValue(), diags
	}

	return MaintenanceListValue{ListValue: list}, diags
}

func nullMaintenanceListValue() MaintenanceListValue {
	return MaintenanceListValue{ListValue: types.ListNull(maintenanceElemType)}
}
