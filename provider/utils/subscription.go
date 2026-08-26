package utils

import (
	"context"
	"fmt"
	"sort"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/pricing"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
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

// PricingModel represents one pricing entry returned by the subscription data sources.
type PricingModel struct {
	DatabaseName        types.String  `tfsdk:"database_name"`
	Type                types.String  `tfsdk:"type"`
	TypeDetails         types.String  `tfsdk:"type_details"`
	Quantity            types.Int64   `tfsdk:"quantity"`
	QuantityMeasurement types.String  `tfsdk:"quantity_measurement"`
	PricePerUnit        types.Float64 `tfsdk:"price_per_unit"`
	PriceCurrency       types.String  `tfsdk:"price_currency"`
	PricePeriod         types.String  `tfsdk:"price_period"`
	Region              types.String  `tfsdk:"region"`
}

var pricingAttrTypes = customtypes.AttrTypesOf(PricingModel{})

// PricingListFromAPI converts a pricing API response into the pricing list expected by the
// subscription data sources. Entries are sorted by a composite key so the ordered list
// is stable across reads. Pricing.List does not guarantee a consistent order, which
// would otherwise churn the list and produce a perpetual plan diff.
func PricingListFromAPI(ctx context.Context, prices []*pricing.Pricing) (types.List, diag.Diagnostics) {
	pricingType := types.ObjectType{AttrTypes: pricingAttrTypes}

	sorted := make([]*pricing.Pricing, len(prices))
	copy(sorted, prices)
	sort.SliceStable(sorted, func(i, j int) bool {
		return pricingSortKey(sorted[i]) < pricingSortKey(sorted[j])
	})

	elems := make([]PricingModel, 0, len(sorted))
	for _, p := range sorted {
		elems = append(elems, PricingModel{
			DatabaseName:        types.StringPointerValue(p.DatabaseName),
			Type:                types.StringPointerValue(p.Type),
			TypeDetails:         types.StringPointerValue(p.TypeDetails),
			Quantity:            types.Int64Value(int64(redis.IntValue(p.Quantity))),
			QuantityMeasurement: types.StringPointerValue(p.QuantityMeasurement),
			PricePerUnit:        types.Float64PointerValue(p.PricePerUnit),
			PriceCurrency:       types.StringPointerValue(p.PriceCurrency),
			PricePeriod:         types.StringPointerValue(p.PricePeriod),
			Region:              types.StringPointerValue(p.Region),
		})
	}

	return types.ListValueFrom(ctx, pricingType, elems)
}

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

// ResourceTagsFromAPI converts the API's key/value tag slice into a types.Map for
// Terraform state. It always yields a known (possibly empty) map, never null, so a
// "tags cleared" state records an empty map rather than a missing attribute.
func ResourceTagsFromAPI(ctx context.Context, tags []*subscriptions.ResourceTag) (types.Map, diag.Diagnostics) {
	result := make(map[string]string, len(tags))
	for _, t := range tags {
		result[redis.StringValue(t.Key)] = redis.StringValue(t.Value)
	}
	return types.MapValueFrom(ctx, types.StringType, result)
}
