package provider

import (
	"context"
	"fmt"
	"sort"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/pro"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// findRegionFieldInCtyValue navigates through a cty.Value structure to find a field
// in the override_region Set for a specific region.
//
// This generic helper is used to check both raw config (GetRawConfig) and raw state (GetRawState)
// without triggering SDK v2's TypeSet materialization that adds all schema fields with zero-values.
//
// Parameters:
//   - ctyVal: The cty.Value to search (from GetRawConfig or GetRawState)
//   - regionName: The name of the region to find (e.g., "us-east-1")
//   - fieldName: The field name to check within the region (e.g., "enable_default_user")
//
// Returns:
//   - fieldValue: The cty.Value of the field if found
//   - exists: true if the field was found and is not null, false otherwise
func findRegionFieldInCtyValue(ctyVal cty.Value, regionName string, fieldName string) (cty.Value, bool) {
	if ctyVal.IsNull() {
		return cty.NilVal, false
	}

	if !ctyVal.Type().HasAttribute("override_region") {
		return cty.NilVal, false
	}

	overrideRegionAttr := ctyVal.GetAttr("override_region")
	if overrideRegionAttr.IsNull() {
		return cty.NilVal, false
	}

	if !overrideRegionAttr.Type().IsSetType() {
		return cty.NilVal, false
	}

	iter := overrideRegionAttr.ElementIterator()
	for iter.Next() {
		_, regionVal := iter.Element()

		// Check if this is the region we're looking for
		if regionVal.Type().HasAttribute("name") {
			nameAttr := regionVal.GetAttr("name")
			if !nameAttr.IsNull() && nameAttr.AsString() == regionName {
				// Found matching region, check for field
				if regionVal.Type().HasAttribute(fieldName) {
					fieldAttr := regionVal.GetAttr(fieldName)
					if !fieldAttr.IsNull() {
						return fieldAttr, true
					}
				}
				return cty.NilVal, false
			}
		}
	}

	return cty.NilVal, false
}

// isEnableDefaultUserExplicitlySetInConfig checks if enable_default_user was explicitly
// set in the Terraform configuration for a specific region in the override_region block.
//
// This is used by the Update function to determine whether to send the field to the API.
// We only need this for Update operations where GetRawConfig() is available.
func isEnableDefaultUserExplicitlySetInConfig(d *schema.ResourceData, regionName string) bool {
	_, exists := findRegionFieldInCtyValue(d.GetRawConfig(), regionName, "enable_default_user")
	return exists
}

// isEnableDefaultUserInActualPersistedState checks if enable_default_user was in the ACTUAL
// persisted Terraform state (not the materialized Go map) for a specific region.
// Uses GetRawState to bypass TypeSet materialization that adds all fields with zero-values.
func isEnableDefaultUserInActualPersistedState(d *schema.ResourceData, regionName string) bool {
	_, exists := findRegionFieldInCtyValue(d.GetRawState(), regionName, "enable_default_user")
	return exists
}

// enableDefaultUserDecision encapsulates the decision result for whether to include
// enable_default_user in the region config.
type enableDefaultUserDecision struct {
	shouldInclude bool
	reason        string
}

// decideEnableDefaultUserInclusion determines whether to include enable_default_user
// in the region state based on config/state context and API values.
//
// This implements the hybrid GetRawConfig/GetRawState strategy:
//   - During Apply/Update (when GetRawConfig available): Check if explicitly set in config
//   - During Refresh (when GetRawConfig unavailable): Check if was in persisted state
//
// Parameters:
//   - d: ResourceData containing config and state
//   - region: The region name (e.g., "us-east-1")
//   - regionValue: The enable_default_user value from the API for this region
//   - globalValue: The global_enable_default_user value from the API
//
// Returns:
//   - Decision indicating whether to include the field and why
func decideEnableDefaultUserInclusion(
	d *schema.ResourceData,
	region string,
	regionValue bool,
	globalValue bool,
) enableDefaultUserDecision {
	rawConfig := d.GetRawConfig()
	valuesDiffer := regionValue != globalValue

	// Try config-based detection first (available during Apply/Update)
	if !rawConfig.IsNull() {
		if isEnableDefaultUserExplicitlySetInConfig(d, region) {
			return enableDefaultUserDecision{
				shouldInclude: true,
				reason:        "explicitly set in config",
			}
		}
		if valuesDiffer {
			return enableDefaultUserDecision{
				shouldInclude: true,
				reason:        "differs from global (API override)",
			}
		}
		return enableDefaultUserDecision{
			shouldInclude: false,
			reason:        "matches global (inherited)",
		}
	}

	// Fall back to state-based detection (during Refresh)
	wasInState := isEnableDefaultUserInActualPersistedState(d, region)

	if wasInState {
		reason := "was in state, preserving (user explicit)"
		if valuesDiffer {
			reason = "was in state, differs from global"
		}
		return enableDefaultUserDecision{
			shouldInclude: true,
			reason:        reason,
		}
	}

	if valuesDiffer {
		return enableDefaultUserDecision{
			shouldInclude: true,
			reason:        "not in state, but differs from global",
		}
	}

	return enableDefaultUserDecision{
		shouldInclude: false,
		reason:        "not in state, matches global (inherited)",
	}
}

// filterDefaultSourceIPs removes default source IP values that should not be in state.
// Returns empty slice if IPs are default values (private ranges or 0.0.0.0/0).
//
// The API returns different defaults based on public_endpoint_access:
//   - When public access disabled: Returns private IP ranges
//   - When public access enabled: Returns ["0.0.0.0/0"]
//   - When explicitly set by user: Returns user's values
//
// This filtering prevents drift from API-generated defaults.
func filterDefaultSourceIPs(apiSourceIPs []*string) []string {
	privateIPRanges := []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "100.64.0.0/10"}

	// Check for default public access ["0.0.0.0/0"]
	if len(apiSourceIPs) == 1 && redis.StringValue(apiSourceIPs[0]) == "0.0.0.0/0" {
		return []string{}
	}

	// Check for default private IP ranges
	if len(apiSourceIPs) == len(privateIPRanges) {
		isPrivateDefault := true
		for i, ip := range apiSourceIPs {
			if redis.StringValue(ip) != privateIPRanges[i] {
				isPrivateDefault = false
				break
			}
		}
		if isPrivateDefault {
			return []string{}
		}
	}

	return redis.StringSliceValue(apiSourceIPs...)
}

// addSourceIPsIfOverridden adds override_global_source_ips to region config if it differs from global.
func addSourceIPsIfOverridden(ctx context.Context, regionDbConfig map[string]interface{}, d *schema.ResourceData, regionDb *databases.CrdbDatabase, region string) {
	sourceIPs := filterDefaultSourceIPs(regionDb.Security.SourceIPs)

	tflog.Debug(ctx, "Read: addSourceIPsIfOverridden", map[string]interface{}{
		"region":              region,
		"apiSourceIPsCount":   len(regionDb.Security.SourceIPs),
		"filteredIPsCount":    len(sourceIPs),
		"filteredIPs":         sourceIPs,
	})

	if len(sourceIPs) == 0 {
		tflog.Debug(ctx, "Read: Skipping source IPs (filtered to empty)", map[string]interface{}{"region": region})
		return
	}

	globalSourceIPsPtrs := utils.SetToStringSlice(d.Get("global_source_ips").(*schema.Set))
	globalSourceIPs := redis.StringSliceValue(globalSourceIPsPtrs...)

	shouldAdd := !stringSlicesEqual(sourceIPs, globalSourceIPs)
	tflog.Debug(ctx, "Read: Source IPs comparison", map[string]interface{}{
		"region":       region,
		"regionIPs":    sourceIPs,
		"globalIPs":    globalSourceIPs,
		"shouldAdd":    shouldAdd,
	})

	if shouldAdd {
		regionDbConfig["override_global_source_ips"] = sourceIPs
	}
}

// addDataPersistenceIfOverridden adds override_global_data_persistence to region config if it differs from global.
func addDataPersistenceIfOverridden(
	ctx context.Context,
	regionDbConfig map[string]interface{},
	db *databases.ActiveActiveDatabase,
	regionDb *databases.CrdbDatabase,
	region string,
) {
	if regionDb.DataPersistence != nil && db.GlobalDataPersistence != nil {
		regionValue := redis.StringValue(regionDb.DataPersistence)
		globalValue := redis.StringValue(db.GlobalDataPersistence)
		shouldAdd := regionValue != globalValue

		tflog.Debug(ctx, "Read: Data persistence comparison", map[string]interface{}{
			"region":       region,
			"regionValue":  regionValue,
			"globalValue":  globalValue,
			"shouldAdd":    shouldAdd,
		})

		if shouldAdd {
			regionDbConfig["override_global_data_persistence"] = regionDb.DataPersistence
		}
	} else {
		tflog.Debug(ctx, "Read: Skipping data persistence (nil values)", map[string]interface{}{
			"region":           region,
			"regionIsNil":      regionDb.DataPersistence == nil,
			"globalIsNil":      db.GlobalDataPersistence == nil,
		})
	}
}

// addPasswordIfOverridden adds override_global_password to region config if it differs from global.
func addPasswordIfOverridden(
	ctx context.Context,
	regionDbConfig map[string]interface{},
	db *databases.ActiveActiveDatabase,
	regionDb *databases.CrdbDatabase,
	region string,
) {
	if regionDb.Security.Password != nil && db.GlobalPassword != nil {
		regionValue := *regionDb.Security.Password
		globalValue := redis.StringValue(db.GlobalPassword)
		shouldAdd := regionValue != globalValue

		tflog.Debug(ctx, "Read: Password comparison", map[string]interface{}{
			"region":       region,
			"regionValue":  "[REDACTED]",
			"globalValue":  "[REDACTED]",
			"valuesDiffer": shouldAdd,
			"shouldAdd":    shouldAdd,
		})

		if shouldAdd {
			regionDbConfig["override_global_password"] = regionValue
		}
	} else {
		tflog.Debug(ctx, "Read: Skipping password (nil values)", map[string]interface{}{
			"region":      region,
			"regionIsNil": regionDb.Security.Password == nil,
			"globalIsNil": db.GlobalPassword == nil,
		})
	}
}

// addAlertsIfOverridden adds override_global_alert to region config if alerts differ from global.
func addAlertsIfOverridden(
	ctx context.Context,
	regionDbConfig map[string]interface{},
	d *schema.ResourceData,
	regionDb *databases.CrdbDatabase,
	region string,
) {
	globalAlerts := d.Get("global_alert").(*schema.Set).List()
	regionAlerts := pro.FlattenAlerts(regionDb.Alerts)

	// Compare alert content, not just counts
	shouldAdd := !alertsEqualContent(globalAlerts, regionAlerts)

	tflog.Debug(ctx, "Read: Alerts comparison", map[string]interface{}{
		"region":            region,
		"globalAlertsCount": len(globalAlerts),
		"globalAlerts":      globalAlerts,
		"regionAlertsCount": len(regionAlerts),
		"regionAlerts":      regionAlerts,
		"shouldAdd":         shouldAdd,
	})

	if shouldAdd {
		regionDbConfig["override_global_alert"] = regionAlerts
	}
}

// alertsEqualContent compares two alert lists for equality by comparing their content.
// Takes []interface{} (from state) and []map[string]interface{} (from API flatten) for comparison.
func alertsEqualContent(globalAlerts []interface{}, regionAlerts []map[string]interface{}) bool {
	if len(globalAlerts) != len(regionAlerts) {
		return false
	}

	// Convert global alerts to a set of "name:value" strings
	globalSet := make(map[string]bool, len(globalAlerts))
	for _, alert := range globalAlerts {
		alertMap := alert.(map[string]interface{})
		name := alertMap["name"].(string)
		value := alertMap["value"]
		key := fmt.Sprintf("%s:%v", name, value)
		globalSet[key] = true
	}

	// Check if all region alerts exist in global set
	for _, alertMap := range regionAlerts {
		name := alertMap["name"].(string)
		value := alertMap["value"]
		key := fmt.Sprintf("%s:%v", name, value)
		if !globalSet[key] {
			return false
		}
	}

	return true
}

// addRemoteBackupIfConfigured adds remote_backup to region config if it exists in both API and state.
func addRemoteBackupIfConfigured(
	ctx context.Context,
	regionDbConfig map[string]interface{},
	regionDb *databases.CrdbDatabase,
	stateOverrideRegion map[string]interface{},
	region string,
) {
	tflog.Debug(ctx, "Read: Checking remote backup", map[string]interface{}{
		"region":      region,
		"apiHasBackup": regionDb.Backup != nil,
	})

	if regionDb.Backup == nil {
		tflog.Debug(ctx, "Read: Skipping remote backup (nil in API)", map[string]interface{}{"region": region})
		return
	}

	stateRemoteBackup := stateOverrideRegion["remote_backup"]
	if stateRemoteBackup == nil {
		tflog.Debug(ctx, "Read: Skipping remote backup (nil in state)", map[string]interface{}{"region": region})
		return
	}

	stateRemoteBackupList := stateRemoteBackup.([]interface{})
	tflog.Debug(ctx, "Read: Remote backup state list", map[string]interface{}{
		"region":    region,
		"listCount": len(stateRemoteBackupList),
	})

	if len(stateRemoteBackupList) > 0 {
		regionDbConfig["remote_backup"] = pro.FlattenBackupPlan(regionDb.Backup, stateRemoteBackupList, "")
		tflog.Debug(ctx, "Read: Added remote_backup to region config", map[string]interface{}{"region": region})
	}
}

// addEnableDefaultUserIfNeeded applies hybrid GetRawConfig/GetRawState logic
// to determine if enable_default_user should be in state.
func addEnableDefaultUserIfNeeded(
	ctx context.Context,
	regionDbConfig map[string]interface{},
	d *schema.ResourceData,
	db *databases.ActiveActiveDatabase,
	region string,
	regionDb *databases.CrdbDatabase,
) {
	if regionDb.Security.EnableDefaultUser == nil || db.GlobalEnableDefaultUser == nil {
		return
	}

	regionEnableDefaultUser := redis.BoolValue(regionDb.Security.EnableDefaultUser)
	globalEnableDefaultUser := redis.BoolValue(db.GlobalEnableDefaultUser)

	decision := decideEnableDefaultUserInclusion(d, region, regionEnableDefaultUser, globalEnableDefaultUser)

	if decision.shouldInclude {
		regionDbConfig["enable_default_user"] = regionEnableDefaultUser
	}

	tflog.Debug(ctx, "Read: enable_default_user decision", map[string]interface{}{
		"region":                region,
		"getRawConfigAvailable": !d.GetRawConfig().IsNull(),
		"shouldInclude":         decision.shouldInclude,
		"value":                 regionEnableDefaultUser,
		"reason":                decision.reason,
	})
}

// logRegionConfigBuilt logs the final region config for debugging.
func logRegionConfigBuilt(ctx context.Context, region string, regionDbConfig map[string]interface{}) {
	tflog.Debug(ctx, "Read: Completed region config", map[string]interface{}{
		"region":                           region,
		"hasEnableDefaultUser":             regionDbConfig["enable_default_user"] != nil,
		"enableDefaultUserValue":           regionDbConfig["enable_default_user"],
		"hasOverrideGlobalSourceIps":       regionDbConfig["override_global_source_ips"] != nil,
		"hasOverrideGlobalDataPersistence": regionDbConfig["override_global_data_persistence"] != nil,
		"hasOverrideGlobalPassword":        regionDbConfig["override_global_password"] != nil,
		"hasOverrideGlobalAlert":           regionDbConfig["override_global_alert"] != nil,
		"hasRemoteBackup":                  regionDbConfig["remote_backup"] != nil,
	})
}

// buildRegionConfigFromAPIAndState orchestrates building region config from API and state.
// Each override field is handled by a dedicated helper function for clarity and maintainability.
func buildRegionConfigFromAPIAndState(ctx context.Context, d *schema.ResourceData, db *databases.ActiveActiveDatabase, region string, regionDb *databases.CrdbDatabase, stateOverrideRegion map[string]interface{}) map[string]interface{} {
	tflog.Debug(ctx, "Read: Starting buildRegionConfigFromAPIAndState", map[string]interface{}{
		"region": region,
	})

	regionDbConfig := map[string]interface{}{
		"name": region,
	}

	// Handle each override field using dedicated helper functions
	addSourceIPsIfOverridden(ctx, regionDbConfig, d, regionDb, region)
	addDataPersistenceIfOverridden(ctx, regionDbConfig, db, regionDb, region)
	addPasswordIfOverridden(ctx, regionDbConfig, db, regionDb, region)
	addAlertsIfOverridden(ctx, regionDbConfig, d, regionDb, region)
	addRemoteBackupIfConfigured(ctx, regionDbConfig, regionDb, stateOverrideRegion, region)
	addEnableDefaultUserIfNeeded(ctx, regionDbConfig, d, db, region, regionDb)

	logRegionConfigBuilt(ctx, region, regionDbConfig)

	return regionDbConfig
}

// stringSlicesEqual compares two string slices for equality (order-insensitive).
// This is used for comparing source IP lists where order doesn't matter.
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	// Make copies to avoid modifying the original slices
	aCopy := make([]string, len(a))
	bCopy := make([]string, len(b))
	copy(aCopy, a)
	copy(bCopy, b)

	// Sort both copies
	sort.Strings(aCopy)
	sort.Strings(bCopy)

	// Compare sorted slices
	for i := range aCopy {
		if aCopy[i] != bCopy[i] {
			return false
		}
	}
	return true
}
