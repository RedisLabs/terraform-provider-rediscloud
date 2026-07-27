package utils

import (
	"context"
	"fmt"
	"sort"
	"strconv"

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

// ProSubscriptionFilter matches pro subscriptions. The active-active
// and pro data sources share the Subscription.List endpoint and split its results on
// deployment type — this is the pro half.
func ProSubscriptionFilter() SubscriptionFilter {
	return func(sub *subscriptions.Subscription) bool {
		return redis.StringValue(sub.DeploymentType) == subscriptions.SubscriptionDeploymentTypeSingleRegion
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

// PricingFromAPI converts a pricing API response into the pricing list expected by the
// subscription data sources. Entries are sorted by a composite key so the ordered list
// is stable across reads. Pricing.List does not guarantee a consistent order, which
// would otherwise churn the list and produce a perpetual plan diff.
func PricingFromAPI(ctx context.Context, prices []*pricing.Pricing) (types.List, diag.Diagnostics) {
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

type cloudNetworkModel struct {
	NetworkingSubnetID       types.String `tfsdk:"networking_subnet_id"`
	NetworkingDeploymentCIDR types.String `tfsdk:"networking_deployment_cidr"`
	NetworkingVPCID          types.String `tfsdk:"networking_vpc_id"`
}

var cloudNetworkAttrTypes = customtypes.AttrTypesOf(cloudNetworkModel{})

type cloudRegionModel struct {
	Region                     types.String `tfsdk:"region"`
	MultipleAvailabilityZones  types.Bool   `tfsdk:"multiple_availability_zones"`
	PreferredAvailabilityZones types.List   `tfsdk:"preferred_availability_zones"`
	NetworkingVPCID            types.String `tfsdk:"networking_vpc_id"`
	Networks                   types.List   `tfsdk:"networks"`
}

// cloudRegionAttrTypes seeds the element types that zero-value lists cannot
// report.
var cloudRegionAttrTypes = customtypes.AttrTypesOf(cloudRegionModel{
	PreferredAvailabilityZones: types.ListNull(types.StringType),
	Networks:                   types.ListNull(types.ObjectType{AttrTypes: cloudNetworkAttrTypes}),
})

type cloudProviderModel struct {
	Provider       types.String `tfsdk:"provider"`
	CloudAccountID types.String `tfsdk:"cloud_account_id"`
	AWSAccountID   types.String `tfsdk:"aws_account_id"`
	ResourceTags   types.Map    `tfsdk:"resource_tags"`
	Region         types.Set    `tfsdk:"region"`
}

// cloudProviderAttrTypes seeds the element types that zero-value maps and sets
// cannot report.
var cloudProviderAttrTypes = customtypes.AttrTypesOf(cloudProviderModel{
	ResourceTags: types.MapNull(types.StringType),
	Region:       types.SetNull(types.ObjectType{AttrTypes: cloudRegionAttrTypes}),
})

// CloudProvidersFromAPI maps cloud details to the computed data-source shape.
// The region-level VPC identifier remains a known empty string because this shape
// exposes networking identifiers in the nested networks block.
func CloudProvidersFromAPI(ctx context.Context, cloudDetails []*subscriptions.CloudDetail) (types.List, diag.Diagnostics) {
	cloudProviderType := types.ObjectType{AttrTypes: cloudProviderAttrTypes}
	models := make([]cloudProviderModel, 0, len(cloudDetails))

	for _, cloudDetail := range cloudDetails {
		regions, diags := cloudRegionsFromAPI(ctx, cloudDetail.Regions)
		if diags.HasError() {
			return types.ListNull(cloudProviderType), diags
		}

		resourceTags, diags := ResourceTagsFromAPI(ctx, cloudDetail.ResourceTags)
		if diags.HasError() {
			return types.ListNull(cloudProviderType), diags
		}

		models = append(models, cloudProviderModel{
			Provider:       types.StringValue(redis.StringValue(cloudDetail.Provider)),
			CloudAccountID: types.StringValue(strconv.Itoa(redis.IntValue(cloudDetail.CloudAccountID))),
			AWSAccountID:   types.StringValue(redis.StringValue(cloudDetail.AWSAccountID)),
			ResourceTags:   resourceTags,
			Region:         regions,
		})
	}

	return types.ListValueFrom(ctx, cloudProviderType, models)
}

func cloudRegionsFromAPI(ctx context.Context, regions []*subscriptions.Region) (types.Set, diag.Diagnostics) {
	regionType := types.ObjectType{AttrTypes: cloudRegionAttrTypes}
	models := make([]cloudRegionModel, 0, len(regions))

	for _, region := range regions {
		networks, diags := cloudNetworksFromAPI(ctx, region.Networking)
		if diags.HasError() {
			return types.SetNull(regionType), diags
		}

		availabilityZones := make([]string, 0, len(region.PreferredAvailabilityZones))
		for _, availabilityZone := range region.PreferredAvailabilityZones {
			availabilityZones = append(availabilityZones, redis.StringValue(availabilityZone))
		}

		preferredAZs, diags := types.ListValueFrom(ctx, types.StringType, availabilityZones)
		if diags.HasError() {
			return types.SetNull(regionType), diags
		}

		models = append(models, cloudRegionModel{
			Region:                     types.StringValue(redis.StringValue(region.Region)),
			MultipleAvailabilityZones:  types.BoolValue(redis.BoolValue(region.MultipleAvailabilityZones)),
			PreferredAvailabilityZones: preferredAZs,
			NetworkingVPCID:            types.StringValue(""),
			Networks:                   networks,
		})
	}

	return types.SetValueFrom(ctx, regionType, models)
}

func cloudNetworksFromAPI(ctx context.Context, networks []*subscriptions.Networking) (types.List, diag.Diagnostics) {
	networkType := types.ObjectType{AttrTypes: cloudNetworkAttrTypes}
	models := make([]cloudNetworkModel, 0, len(networks))

	for _, network := range networks {
		models = append(models, cloudNetworkModel{
			NetworkingSubnetID:       types.StringValue(redis.StringValue(network.SubnetID)),
			NetworkingDeploymentCIDR: types.StringValue(redis.StringValue(network.DeploymentCIDR)),
			NetworkingVPCID:          types.StringValue(redis.StringValue(network.VPCId)),
		})
	}

	return types.ListValueFrom(ctx, networkType, models)
}
