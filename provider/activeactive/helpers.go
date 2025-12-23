package activeactive

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

// Default RFC1918 private IP ranges used when public_endpoint_access is false.
var defaultPrivateIPRanges = []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "100.64.0.0/10"}

// parseResourceId parses a resource ID in the format "subId/dbId" and returns the components.
func parseResourceId(id string) (subId int, dbId int, err error) {
	parts := strings.Split(id, "/")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid resource ID format: %s, expected subscription_id/db_id", id)
	}
	subId, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid subscription_id in resource ID: %s", parts[0])
	}
	dbId, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid db_id in resource ID: %s", parts[1])
	}
	return subId, dbId, nil
}

// buildResourceId creates a resource ID in the format "subId/dbId".
func buildResourceId(subId, dbId int) string {
	return fmt.Sprintf("%d/%d", subId, dbId)
}

// isDefaultGlobalSourceIPs checks if the given source IPs match one of the default patterns:
// - RFC1918 private ranges (when public_endpoint_access is false)
// - Public access "0.0.0.0/0" (when public_endpoint_access is true)
// This is used to determine if global_source_ips should be re-computed when public_endpoint_access changes.
func isDefaultGlobalSourceIPs(sourceIPs []string) bool {
	if len(sourceIPs) == 0 {
		return true
	}

	// Check if it's the default public access pattern
	if len(sourceIPs) == 1 && sourceIPs[0] == "0.0.0.0/0" {
		return true
	}

	// Check if it matches the RFC1918 private ranges
	if len(sourceIPs) == len(defaultPrivateIPRanges) {
		privateIPSet := make(map[string]bool)
		for _, ip := range defaultPrivateIPRanges {
			privateIPSet[ip] = true
		}
		for _, ip := range sourceIPs {
			if !privateIPSet[ip] {
				return false
			}
		}
		return true
	}

	return false
}

// flattenModulesToNames converts a slice of Module objects to a slice of module names.
func flattenModulesToNames(modules []*databases.Module) []string {
	moduleNames := make([]string, 0, len(modules))
	for _, module := range modules {
		moduleNames = append(moduleNames, redis.StringValue(module.Name))
	}
	return moduleNames
}

// setToStringSlice converts a types.Set to a slice of strings.
func setToStringSlice(ctx context.Context, set types.Set) ([]string, diag.Diagnostics) {
	if set.IsNull() || set.IsUnknown() {
		return nil, nil
	}

	var result []string
	diags := set.ElementsAs(ctx, &result, false)
	return result, diags
}

// listToStringSlice converts a types.List to a slice of strings.
func listToStringSlice(ctx context.Context, list types.List) ([]string, diag.Diagnostics) {
	if list.IsNull() || list.IsUnknown() {
		return nil, nil
	}

	var result []string
	diags := list.ElementsAs(ctx, &result, false)
	return result, diags
}

// stringSliceToSet converts a slice of strings to a types.Set.
func stringSliceToSet(ctx context.Context, slice []string) (types.Set, diag.Diagnostics) {
	if slice == nil {
		return types.SetNull(types.StringType), nil
	}

	elements := make([]attr.Value, len(slice))
	for i, s := range slice {
		elements[i] = types.StringValue(s)
	}

	return types.SetValue(types.StringType, elements)
}

// stringSliceToList converts a slice of strings to a types.List.
func stringSliceToList(ctx context.Context, slice []string) (types.List, diag.Diagnostics) {
	if slice == nil {
		return types.ListNull(types.StringType), nil
	}

	elements := make([]attr.Value, len(slice))
	for i, s := range slice {
		elements[i] = types.StringValue(s)
	}

	return types.ListValue(types.StringType, elements)
}

// stringMapToMap converts a map[string]string to a types.Map.
func stringMapToMap(ctx context.Context, m map[string]string) (types.Map, diag.Diagnostics) {
	if m == nil {
		return types.MapNull(types.StringType), nil
	}

	elements := make(map[string]attr.Value, len(m))
	for k, v := range m {
		elements[k] = types.StringValue(v)
	}

	return types.MapValue(types.StringType, elements)
}

// mapToStringMap converts a types.Map to a map[string]string.
func mapToStringMap(ctx context.Context, m types.Map) (map[string]string, diag.Diagnostics) {
	if m.IsNull() || m.IsUnknown() {
		return nil, nil
	}

	result := make(map[string]string)
	diags := m.ElementsAs(ctx, &result, false)
	return result, diags
}

// buildAlertsFromSet converts a types.Set of alerts to the API request format.
func buildAlertsFromSet(ctx context.Context, alertSet types.Set) ([]*databases.Alert, diag.Diagnostics) {
	if alertSet.IsNull() || alertSet.IsUnknown() {
		return []*databases.Alert{}, nil
	}

	var alertModels []AlertModel
	diags := alertSet.ElementsAs(ctx, &alertModels, false)
	if diags.HasError() {
		return nil, diags
	}

	alerts := make([]*databases.Alert, 0, len(alertModels))
	for _, alert := range alertModels {
		alerts = append(alerts, &databases.Alert{
			Name:  redis.String(alert.Name.ValueString()),
			Value: redis.Int(int(alert.Value.ValueInt64())),
		})
	}

	return alerts, nil
}

// flattenAlertsToSet converts API alert responses to a types.Set.
func flattenAlertsToSet(ctx context.Context, alerts []*databases.Alert) (types.Set, diag.Diagnostics) {
	alertAttrTypes := map[string]attr.Type{
		"name":  types.StringType,
		"value": types.Int64Type,
	}

	if alerts == nil || len(alerts) == 0 {
		return types.SetNull(types.ObjectType{AttrTypes: alertAttrTypes}), nil
	}

	elements := make([]attr.Value, 0, len(alerts))
	for _, alert := range alerts {
		obj, diags := types.ObjectValue(alertAttrTypes, map[string]attr.Value{
			"name":  types.StringValue(redis.StringValue(alert.Name)),
			"value": types.Int64Value(int64(redis.IntValue(alert.Value))),
		})
		if diags.HasError() {
			return types.SetNull(types.ObjectType{AttrTypes: alertAttrTypes}), diags
		}
		elements = append(elements, obj)
	}

	return types.SetValue(types.ObjectType{AttrTypes: alertAttrTypes}, elements)
}

// getOverrideRegionAttrTypes returns the attribute types for the override_region block.
func getOverrideRegionAttrTypes() map[string]attr.Type {
	alertAttrTypes := map[string]attr.Type{
		"name":  types.StringType,
		"value": types.Int64Type,
	}

	remoteBackupAttrTypes := map[string]attr.Type{
		"interval":     types.StringType,
		"time_utc":     types.StringType,
		"storage_type": types.StringType,
		"storage_path": types.StringType,
	}

	return map[string]attr.Type{
		"name":                             types.StringType,
		"override_global_alert":            types.SetType{ElemType: types.ObjectType{AttrTypes: alertAttrTypes}},
		"override_global_password":         types.StringType,
		"override_global_source_ips":       types.SetType{ElemType: types.StringType},
		"override_global_data_persistence": types.StringType,
		"enable_default_user":              types.BoolType,
		"remote_backup":                    types.ListType{ElemType: types.ObjectType{AttrTypes: remoteBackupAttrTypes}},
	}
}

// getOverrideRegionFromSet finds a specific region by name from the override_region set.
func getOverrideRegionFromSet(ctx context.Context, overrideRegionSet types.Set, regionName string) (*OverrideRegionModel, diag.Diagnostics) {
	if overrideRegionSet.IsNull() || overrideRegionSet.IsUnknown() {
		return nil, nil
	}

	var regions []OverrideRegionModel
	diags := overrideRegionSet.ElementsAs(ctx, &regions, false)
	if diags.HasError() {
		return nil, diags
	}

	for _, region := range regions {
		if region.Name.ValueString() == regionName {
			return &region, nil
		}
	}

	return nil, nil
}

// buildBackupPlan converts a RemoteBackupModel to the API request format.
func buildBackupPlan(ctx context.Context, remoteBackupList types.List) (*databases.DatabaseBackupConfig, diag.Diagnostics) {
	if remoteBackupList.IsNull() || remoteBackupList.IsUnknown() || len(remoteBackupList.Elements()) == 0 {
		return nil, nil
	}

	var backups []RemoteBackupModel
	diags := remoteBackupList.ElementsAs(ctx, &backups, false)
	if diags.HasError() {
		return nil, diags
	}

	if len(backups) == 0 {
		return nil, nil
	}

	backup := backups[0]
	config := &databases.DatabaseBackupConfig{
		Active:      redis.Bool(true),
		Interval:    redis.String(backup.Interval.ValueString()),
		StorageType: redis.String(backup.StorageType.ValueString()),
		StoragePath: redis.String(backup.StoragePath.ValueString()),
	}

	if !backup.TimeUTC.IsNull() && backup.TimeUTC.ValueString() != "" {
		config.TimeUTC = redis.String(backup.TimeUTC.ValueString())
	}

	return config, nil
}

// flattenBackupPlan converts API backup response to a types.List for remote_backup.
// Note: The API doesn't return storage_type, so we preserve it from state if available.
func flattenBackupPlan(ctx context.Context, backup *databases.Backup, stateStorageType string) (types.List, diag.Diagnostics) {
	remoteBackupAttrTypes := map[string]attr.Type{
		"interval":     types.StringType,
		"time_utc":     types.StringType,
		"storage_type": types.StringType,
		"storage_path": types.StringType,
	}

	if backup == nil || !redis.BoolValue(backup.Enabled) {
		return types.ListNull(types.ObjectType{AttrTypes: remoteBackupAttrTypes}), nil
	}

	timeUTC := types.StringNull()
	if backup.TimeUTC != nil {
		timeUTC = types.StringValue(redis.StringValue(backup.TimeUTC))
	}

	// Storage type is not returned by API, preserve from state
	storageType := types.StringValue(stateStorageType)

	obj, diags := types.ObjectValue(remoteBackupAttrTypes, map[string]attr.Value{
		"interval":     types.StringValue(redis.StringValue(backup.Interval)),
		"time_utc":     timeUTC,
		"storage_type": storageType,
		"storage_path": types.StringValue(redis.StringValue(backup.Destination)),
	})
	if diags.HasError() {
		return types.ListNull(types.ObjectType{AttrTypes: remoteBackupAttrTypes}), diags
	}

	return types.ListValue(types.ObjectType{AttrTypes: remoteBackupAttrTypes}, []attr.Value{obj})
}

// waitForDatabaseToBeDeleted waits for the database to be deleted using retry.StateChangeConf.
func waitForDatabaseToBeDeleted(ctx context.Context, subId, dbId int, api *client.ApiClient) error {
	wait := &retry.StateChangeConf{
		Delay:        30 * time.Second,
		Pending:      []string{"pending"},
		Target:       []string{"deleted"},
		Timeout:      10 * time.Minute,
		PollInterval: 30 * time.Second,

		Refresh: func() (result interface{}, state string, err error) {
			log.Printf("[DEBUG] Waiting for database %d to be deleted", dbId)

			_, err = api.Client.Database.Get(ctx, subId, dbId)
			if err != nil {
				if _, ok := err.(*databases.NotFound); ok {
					return "deleted", "deleted", nil
				}
				return nil, "", err
			}

			return "pending", "pending", nil
		},
	}
	if _, err := wait.WaitForStateContext(ctx); err != nil {
		return err
	}

	return nil
}

// stringPtrSlice converts a slice of strings to a slice of string pointers.
func stringPtrSlice(slice []string) []*string {
	if slice == nil {
		return nil
	}
	result := make([]*string, len(slice))
	for i, s := range slice {
		result[i] = redis.String(s)
	}
	return result
}

// stringSliceValue converts a slice of string pointers to a slice of strings.
func stringSliceValue(slice []*string) []string {
	if slice == nil {
		return nil
	}
	result := make([]string, len(slice))
	for i, s := range slice {
		result[i] = redis.StringValue(s)
	}
	return result
}

// getAlertsFromOverrideRegion extracts alerts from an override region model.
func getAlertsFromOverrideRegion(ctx context.Context, region *OverrideRegionModel) ([]*databases.Alert, diag.Diagnostics) {
	if region == nil || region.OverrideGlobalAlert.IsNull() || region.OverrideGlobalAlert.IsUnknown() {
		return nil, nil
	}

	return buildAlertsFromSet(ctx, region.OverrideGlobalAlert)
}

// boolPtrValue safely gets a bool value from a pointer, returning the default if nil.
func boolPtrValue(ptr *bool, defaultValue bool) bool {
	if ptr == nil {
		return defaultValue
	}
	return *ptr
}

// stringValue safely gets the value of a types.String, returning empty string if null/unknown.
func stringValue(s types.String) string {
	if s.IsNull() || s.IsUnknown() {
		return ""
	}
	return s.ValueString()
}

// int64Value safely gets the value of a types.Int64, returning 0 if null/unknown.
func int64Value(i types.Int64) int64 {
	if i.IsNull() || i.IsUnknown() {
		return 0
	}
	return i.ValueInt64()
}

// float64Value safely gets the value of a types.Float64, returning 0 if null/unknown.
func float64Value(f types.Float64) float64 {
	if f.IsNull() || f.IsUnknown() {
		return 0
	}
	return f.ValueFloat64()
}

// boolValue safely gets the value of a types.Bool, returning false if null/unknown.
func boolValue(b types.Bool) bool {
	if b.IsNull() || b.IsUnknown() {
		return false
	}
	return b.ValueBool()
}

// setStringIfNotEmpty sets the value in the model if the source string is not empty.
func setStringIfNotEmpty(source string, target *basetypes.StringValue) {
	if source != "" {
		*target = types.StringValue(source)
	}
}
