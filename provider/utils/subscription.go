package utils

import (
	"context"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/maintenance"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// CmkEnabledString is the value the API uses for PersistentStorageEncryptionType when
// customer-managed key encryption is enabled on a subscription.
const CmkEnabledString = "customer-managed-key"

// SubscriptionFilter reports whether a subscription should be kept when narrowing the
// result of Subscription.List.
type SubscriptionFilter func(sub *subscriptions.Subscription) bool

// FilterSubscriptions returns the subscriptions that pass every filter. It is shared
// between the active-active and pro subscription data sources, which both read from
// the single Subscription.List endpoint and then narrow the result by deployment type.
func FilterSubscriptions(subs []*subscriptions.Subscription, filters []SubscriptionFilter) []*subscriptions.Subscription {
	var filtered []*subscriptions.Subscription
	for _, sub := range subs {
		if filterSubscription(sub, filters) {
			filtered = append(filtered, sub)
		}
	}
	return filtered
}

func filterSubscription(sub *subscriptions.Subscription, filters []SubscriptionFilter) bool {
	for _, f := range filters {
		if !f(sub) {
			return false
		}
	}
	return true
}

// ActiveActiveSubscriptionFilter matches active-active subscriptions. The active-active
// and pro data sources share the Subscription.List endpoint and split its results on
// deployment type — this is the active-active half.
func ActiveActiveSubscriptionFilter() SubscriptionFilter {
	return func(sub *subscriptions.Subscription) bool {
		return redis.StringValue(sub.DeploymentType) == subscriptions.SubscriptionDeploymentTypeActiveActive
	}
}

// SubscriptionNameFilter matches subscriptions with the given name.
func SubscriptionNameFilter(name string) SubscriptionFilter {
	return func(sub *subscriptions.Subscription) bool {
		return redis.StringValue(sub.Name) == name
	}
}

// maintenanceWindowAttrTypes describes a single window within a maintenance_windows block.
var maintenanceWindowAttrTypes = map[string]attr.Type{
	"start_hour":        types.Int64Type,
	"duration_in_hours": types.Int64Type,
	"days":              types.ListType{ElemType: types.StringType},
}

// maintenanceAttrTypes describes a single maintenance_windows block.
var maintenanceAttrTypes = map[string]attr.Type{
	"mode":   types.StringType,
	"window": types.ListType{ElemType: types.ObjectType{AttrTypes: maintenanceWindowAttrTypes}},
}

// FlattenMaintenance converts a maintenance API response into the single-element
// maintenance_windows list expected by the subscription data sources.
func FlattenMaintenance(ctx context.Context, m *maintenance.Maintenance) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	maintenanceType := types.ObjectType{AttrTypes: maintenanceAttrTypes}
	if m == nil {
		return types.ListNull(maintenanceType), diags
	}

	windowType := types.ObjectType{AttrTypes: maintenanceWindowAttrTypes}
	// A non-null but possibly empty list: automatic-mode maintenance has no windows.
	windowElems := make([]attr.Value, 0, len(m.Windows))
	for _, w := range m.Windows {
		days, d := types.ListValueFrom(ctx, types.StringType, w.Days)
		diags.Append(d...)
		if diags.HasError() {
			return types.ListNull(maintenanceType), diags
		}

		window, d := types.ObjectValue(maintenanceWindowAttrTypes, map[string]attr.Value{
			"start_hour":        types.Int64Value(int64(redis.IntValue(w.StartHour))),
			"duration_in_hours": types.Int64Value(int64(redis.IntValue(w.DurationInHours))),
			"days":              days,
		})
		diags.Append(d...)
		if diags.HasError() {
			return types.ListNull(maintenanceType), diags
		}
		windowElems = append(windowElems, window)
	}

	windows, d := types.ListValue(windowType, windowElems)
	diags.Append(d...)
	if diags.HasError() {
		return types.ListNull(maintenanceType), diags
	}

	entry, d := types.ObjectValue(maintenanceAttrTypes, map[string]attr.Value{
		"mode":   types.StringValue(redis.StringValue(m.Mode)),
		"window": windows,
	})
	diags.Append(d...)
	if diags.HasError() {
		return types.ListNull(maintenanceType), diags
	}

	list, d := types.ListValue(maintenanceType, []attr.Value{entry})
	diags.Append(d...)
	return list, diags
}

// FlattenResourceTags converts the API's key/value tag slice into a types.Map for
// Terraform state. It always yields a known (possibly empty) map, never null, so a
// "tags cleared" state records an empty map rather than a missing attribute.
func FlattenResourceTags(ctx context.Context, tags []*subscriptions.ResourceTag) (types.Map, diag.Diagnostics) {
	result := make(map[string]string, len(tags))
	for _, t := range tags {
		result[redis.StringValue(t.Key)] = redis.StringValue(t.Value)
	}
	return types.MapValueFrom(ctx, types.StringType, result)
}
