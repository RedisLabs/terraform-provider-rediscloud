package pro

import (
	"context"
	"fmt"
	"strconv"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/RedisLabs/rediscloud-go-api/service/maintenance"
	"github.com/RedisLabs/rediscloud-go-api/service/pricing"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func flattenSubscriptionAllowlist(ctx context.Context, subId int, api *client.ApiClient) ([]map[string]interface{}, error) {
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

func FlattenCloudDetails(cloudDetails []*subscriptions.CloudDetail, isResource bool) []map[string]interface{} {
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

		if currentCloudDetail.AWSAccountID != nil {
			cdlMapString["aws_account_id"] = redis.StringValue(currentCloudDetail.AWSAccountID)
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

func FlattenAlerts(alerts []*databases.Alert) []map[string]interface{} {
	var tfs = make([]map[string]interface{}, 0)

	for _, alert := range alerts {
		tf := map[string]interface{}{
			"name":  redis.StringValue(alert.Name),
			"value": redis.IntValue(alert.Value),
		}
		tfs = append(tfs, tf)
	}

	return tfs
}

func FlattenModules(modules []*databases.Module) []map[string]interface{} {

	var tfs = make([]map[string]interface{}, 0)
	for _, module := range modules {

		tf := map[string]interface{}{
			"name": redis.StringValue(module.Name),
		}
		tfs = append(tfs, tf)
	}

	return tfs
}

func FlattenRegexRules(rules []*databases.RegexRule) []string {
	ret := make([]string, len(rules))
	for _, rule := range rules {
		ret[rule.Ordinal] = rule.Pattern
	}

	if len(ret) == 2 && ret[0] == ".*\\{(?<tag>.*)\\}.*" && ret[1] == "(?<tag>.*)" {
		// This is the default regex rules - https://docs.redislabs.com/latest/rc/concepts/clustering/#custom-hashing-policy
		return []string{}
	}

	return ret
}

func ReadPaymentMethodID(d *schema.ResourceData) (*int, error) {
	pmID := d.Get("payment_method_id").(string)
	if pmID != "" {
		pmID, err := strconv.Atoi(pmID)
		if err != nil {
			return nil, err
		}
		return redis.Int(pmID), nil
	}
	return nil, nil
}

func FlattenPricing(pricing []*pricing.Pricing) []map[string]interface{} {
	var tfs = make([]map[string]interface{}, 0)
	for _, p := range pricing {

		tf := map[string]interface{}{
			"database_name":        p.DatabaseName,
			"type":                 p.Type,
			"type_details":         p.TypeDetails,
			"quantity":             p.Quantity,
			"quantity_measurement": p.QuantityMeasurement,
			"price_per_unit":       p.PricePerUnit,
			"price_currency":       p.PriceCurrency,
			"price_period":         p.PricePeriod,
			"region":               p.Region,
		}
		tfs = append(tfs, tf)
	}

	return tfs
}

func FlattenMaintenance(m *maintenance.Maintenance) []map[string]interface{} {
	var windows []map[string]interface{}
	for _, w := range m.Windows {
		tfw := map[string]interface{}{
			"start_hour":        w.StartHour,
			"duration_in_hours": w.DurationInHours,
			"days":              w.Days,
		}
		windows = append(windows, tfw)
	}

	tf := map[string]interface{}{
		"mode":   m.Mode,
		"window": windows,
	}

	return []map[string]interface{}{tf}
}
