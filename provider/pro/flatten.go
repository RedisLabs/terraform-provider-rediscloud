package pro

import (
	"strconv"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/RedisLabs/rediscloud-go-api/service/maintenance"
	"github.com/RedisLabs/rediscloud-go-api/service/pricing"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

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

// FlattenPricing maps pricing state for SDKv2 subscription resources.
// Keep this map-based helper until its remaining callers migrate to the Framework.
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

// FlattenMaintenance maps maintenance state for SDKv2 subscription resources.
// Keep this map-based helper until its remaining callers migrate to the Framework.
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
