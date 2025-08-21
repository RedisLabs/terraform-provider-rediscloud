package pro

import (
	"context"
	"fmt"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"strconv"
	"strings"
	"time"
)

func flattenSubscriptionAllowlist(ctx context.Context, subId int, api *utils.ApiClient) ([]map[string]interface{}, error) {
	allowlist, err := api.Client.Subscription.GetCIDRAllowlist(ctx, subId)
	if err != nil {
		return nil, err
	}

	if !isNil(allowlist.Errors) {
		return nil, fmt.Errorf("unable to read allowlist for subscription %d: %v", subId, allowlist.Errors)
	}

	var cidrs []string
	for _, cidr := range allowlist.CIDRIPs {
		cidrs = append(cidrs, redis.StringValue(cidr))
	}
	var sgs []string
	for _, sg := range allowlist.SecurityGroupIDs {
		sgs = append(sgs, redis.StringValue(sg))
	}

	tfs := map[string]interface{}{}

	if len(cidrs) != 0 {
		tfs["cidrs"] = cidrs
	}
	if len(sgs) != 0 {
		tfs["security_group_ids"] = sgs
	}
	if len(tfs) == 0 {
		return nil, nil
	}

	return []map[string]interface{}{tfs}, nil
}

func isNil(i interface{}) bool {
	if i == nil {
		return true
	}

	if l, ok := i.([]interface{}); ok {
		if len(l) == 0 {
			return true
		}
	}

	if m, ok := i.(map[string]interface{}); ok {
		if len(m) == 0 {
			return true
		}
	}

	return false
}

func flattenCloudDetails(cloudDetails []*subscriptions.CloudDetail, isResource bool) []map[string]interface{} {
	var cdl []map[string]interface{}

	for _, currentCloudDetail := range cloudDetails {

		var regions []interface{}
		for _, currentRegion := range currentCloudDetail.Regions {

			regionMapString := map[string]interface{}{
				"region":                       currentRegion.Region,
				"multiple_availability_zones":  currentRegion.MultipleAvailabilityZones,
				"preferred_availability_zones": currentRegion.PreferredAvailabilityZones,
				"networks":                     flattenNetworks(currentRegion.Networking),
			}

			if isResource {
				if len(currentRegion.Networking) > 0 && !redis.BoolValue(currentRegion.MultipleAvailabilityZones) {
					regionMapString["networking_deployment_cidr"] = currentRegion.Networking[0].DeploymentCIDR
				} else {
					regionMapString["networking_deployment_cidr"] = ""
				}
			}

			regions = append(regions, regionMapString)
		}

		cdlMapString := map[string]interface{}{
			"provider":         currentCloudDetail.Provider,
			"cloud_account_id": strconv.Itoa(redis.IntValue(currentCloudDetail.CloudAccountID)),
			"region":           regions,
		}
		cdl = append(cdl, cdlMapString)
	}

	return cdl
}

func flattenNetworks(networks []*subscriptions.Networking) []map[string]interface{} {
	var cdl []map[string]interface{}

	for _, currentNetwork := range networks {

		networkMapString := map[string]interface{}{
			"networking_deployment_cidr": currentNetwork.DeploymentCIDR,
			"networking_vpc_id":          currentNetwork.VPCId,
			"networking_subnet_id":       currentNetwork.SubnetID,
		}

		cdl = append(cdl, networkMapString)
	}

	return cdl
}

func skipDiffIfIntervalIs12And12HourTimeDiff(k, oldValue, newValue string, d *schema.ResourceData) bool {
	// If interval is set to every 12 hours and the `time_utc` is in the afternoon,
	// then the API will return the _morning_ time when queried.
	// `interval` is assumed to be an attribute within the same block as the attribute being diffed.

	parts := strings.Split(k, ".")
	parts[len(parts)-1] = "interval"

	var interval = d.Get(strings.Join(parts, "."))

	if interval != databases.BackupIntervalEvery12Hours {
		return false
	}

	oldTime, err := time.Parse("15:04", oldValue)
	if err != nil {
		return false
	}
	newTime, err := time.Parse("15:04", newValue)
	if err != nil {
		return false
	}

	return oldTime.Minute() == newTime.Minute() && oldTime.Add(12*time.Hour).Hour() == newTime.Hour()
}

func customizeDiff() schema.CustomizeDiffFunc {
	return func(ctx context.Context, diff *schema.ResourceDiff, meta interface{}) error {
		if err := validateQueryPerformanceFactor()(ctx, diff, meta); err != nil {
			return err
		}
		if err := utils.RemoteBackupIntervalSetCorrectly("remote_backup")(ctx, diff, meta); err != nil {
			return err
		}
		return nil
	}
}

func validateQueryPerformanceFactor() schema.CustomizeDiffFunc {
	return func(ctx context.Context, diff *schema.ResourceDiff, meta interface{}) error {
		// Check if "query_performance_factor" is set
		qpf, qpfExists := diff.GetOk("query_performance_factor")

		// Ensure "modules" is explicitly defined in the HCL
		_, modulesExists := diff.GetOkExists("modules")

		if qpfExists && qpf.(string) != "" {
			if !modulesExists {
				return fmt.Errorf(`"query_performance_factor" requires the "modules" key to be explicitly defined in HCL`)
			}

			// Retrieve modules as a slice of interfaces
			rawModules := diff.Get("modules").(*schema.Set).List()

			// Convert modules to []map[string]interface{}
			var modules []map[string]interface{}
			for _, rawModule := range rawModules {
				if moduleMap, ok := rawModule.(map[string]interface{}); ok {
					modules = append(modules, moduleMap)
				}
			}

			// Check if "RediSearch" exists
			if !containsDBModule(modules, "RediSearch") {
				return fmt.Errorf(`"query_performance_factor" requires the "modules" list to contain "RediSearch"`)
			}
		}
		return nil
	}
}

// Helper function to check if a module exists
func containsDBModule(modules []map[string]interface{}, moduleName string) bool {
	for _, module := range modules {
		if name, ok := module["name"].(string); ok && name == moduleName {
			return true
		}
	}
	return false
}
