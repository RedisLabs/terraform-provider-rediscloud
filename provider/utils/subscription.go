package utils

import (
	"context"
	"fmt"
	"sort"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/maintenance"
	"github.com/RedisLabs/rediscloud-go-api/service/pricing"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// CmkEnabledString is the value the API uses for PersistentStorageEncryptionType when
// customer-managed key encryption is enabled on a subscription.
const CmkEnabledString = "customer-managed-key"

// FilterSubscriptions returns the subscriptions that pass every filter. It is shared
// between the active-active and pro subscription data sources, which both read from
// the single Subscription.List endpoint and then narrow the result by deployment type.
func FilterSubscriptions(subs []*subscriptions.Subscription, filters []func(sub *subscriptions.Subscription) bool) []*subscriptions.Subscription {
	var filtered []*subscriptions.Subscription
	for _, sub := range subs {
		if filterSubscription(sub, filters) {
			filtered = append(filtered, sub)
		}
	}
	return filtered
}

func filterSubscription(sub *subscriptions.Subscription, filters []func(sub *subscriptions.Subscription) bool) bool {
	for _, f := range filters {
		if !f(sub) {
			return false
		}
	}
	return true
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

// pricingAttrTypes describes a single pricing block.
var pricingAttrTypes = map[string]attr.Type{
	"database_name":        types.StringType,
	"type":                 types.StringType,
	"type_details":         types.StringType,
	"quantity":             types.Int64Type,
	"quantity_measurement": types.StringType,
	"price_per_unit":       types.Float64Type,
	"price_currency":       types.StringType,
	"price_period":         types.StringType,
	"region":               types.StringType,
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
		"mode":   types.StringPointerValue(m.Mode),
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

// FlattenPricing converts a pricing API response into the pricing list expected by the
// subscription data sources. Entries are sorted by a composite key so the ordered list
// is stable across reads: the Pricing.List API does not guarantee a consistent order,
// which would otherwise churn the list and produce a perpetual plan diff.
func FlattenPricing(ctx context.Context, prices []*pricing.Pricing) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	pricingType := types.ObjectType{AttrTypes: pricingAttrTypes}

	sorted := make([]*pricing.Pricing, len(prices))
	copy(sorted, prices)
	sort.SliceStable(sorted, func(i, j int) bool {
		return pricingSortKey(sorted[i]) < pricingSortKey(sorted[j])
	})

	elems := make([]attr.Value, 0, len(sorted))
	for _, p := range sorted {
		entry, d := types.ObjectValue(pricingAttrTypes, map[string]attr.Value{
			"database_name":        types.StringPointerValue(p.DatabaseName),
			"type":                 types.StringPointerValue(p.Type),
			"type_details":         types.StringPointerValue(p.TypeDetails),
			"quantity":             types.Int64Value(int64(redis.IntValue(p.Quantity))),
			"quantity_measurement": types.StringPointerValue(p.QuantityMeasurement),
			"price_per_unit":       types.Float64PointerValue(p.PricePerUnit),
			"price_currency":       types.StringPointerValue(p.PriceCurrency),
			"price_period":         types.StringPointerValue(p.PricePeriod),
			"region":               types.StringPointerValue(p.Region),
		})
		diags.Append(d...)
		if diags.HasError() {
			return types.ListNull(pricingType), diags
		}
		elems = append(elems, entry)
	}

	list, d := types.ListValue(pricingType, elems)
	diags.Append(d...)
	return list, diags
}

// pricingSortKey builds a deterministic ordering key for a pricing entry, combining every
// field so the sort is total even when entries differ only in a single value.
func pricingSortKey(p *pricing.Pricing) string {
	return fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%d|%f",
		redis.StringValue(p.Region),
		redis.StringValue(p.Type),
		redis.StringValue(p.TypeDetails),
		redis.StringValue(p.DatabaseName),
		redis.StringValue(p.QuantityMeasurement),
		redis.StringValue(p.PricePeriod),
		redis.StringValue(p.PriceCurrency),
		redis.IntValue(p.Quantity),
		redis.Float64Value(p.PricePerUnit),
	)
}
