package utils

import (
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/RedisLabs/rediscloud-go-api/service/maintenance"
	"github.com/RedisLabs/rediscloud-go-api/service/psc"
	"github.com/RedisLabs/rediscloud-go-api/service/transit_gateway/attachments"
)

func FlattenCidrs(cidrs []*attachments.Cidr) []string {
	cidrStrings := make([]string, 0)
	for _, cidr := range cidrs {
		cidrStrings = append(cidrStrings, redis.StringValue(cidr.CidrAddress))
	}
	return cidrStrings
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

func FlattenBackupPlan(backup *databases.Backup, existing []interface{}, periodicBackupPath string) []map[string]interface{} {
	if backup == nil || !redis.BoolValue(backup.Enabled) || periodicBackupPath != "" {
		return nil
	}

	storageType := ""
	if len(existing) == 1 {
		d := existing[0].(map[string]interface{})
		storageType = d["storage_type"].(string)
	}

	return []map[string]interface{}{
		{
			"interval":     redis.StringValue(backup.Interval),
			"time_utc":     redis.StringValue(backup.TimeUTC),
			"storage_type": storageType,
			"storage_path": redis.StringValue(backup.Destination),
		},
	}
}

func FlattenPrivateServiceConnectEndpoints(endpoints []*psc.PrivateServiceConnectEndpoint,
	serviceAttachments map[int][]psc.TerraformGCPServiceAttachment) []map[string]interface{} {

	var rl []map[string]interface{}
	for _, endpoint := range endpoints {

		endpointMapString := map[string]interface{}{
			"private_service_connect_endpoint_id": redis.IntValue(endpoint.ID),
			"gcp_project_id":                      redis.StringValue(endpoint.GCPProjectID),
			"gcp_vpc_name":                        redis.StringValue(endpoint.GCPVPCName),
			"gcp_vpc_subnet_name":                 redis.StringValue(endpoint.GCPVPCSubnetName),
			"endpoint_connection_name":            redis.StringValue(endpoint.EndpointConnectionName),
			"status":                              redis.StringValue(endpoint.Status),
			"service_attachments":                 FlattenPrivateServiceConnectEndpointServiceAttachments(serviceAttachments[redis.IntValue(endpoint.ID)]),
		}

		rl = append(rl, endpointMapString)
	}

	return rl
}

func FlattenPrivateServiceConnectEndpointServiceAttachments(serviceAttachments []psc.TerraformGCPServiceAttachment) []map[string]interface{} {
	var rl []map[string]interface{}
	for _, serviceAttachment := range serviceAttachments {

		serviceAttachmentMapString := map[string]interface{}{
			"name":                 serviceAttachment.Name,
			"dns_record":           serviceAttachment.DNSRecord,
			"ip_address_name":      serviceAttachment.IPAddressName,
			"forwarding_rule_name": serviceAttachment.ForwardingRuleName,
		}

		rl = append(rl, serviceAttachmentMapString)
	}

	return rl
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
