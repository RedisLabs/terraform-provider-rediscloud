package utils_test

import (
	"context"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/pricing"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

// --- FilterSubscriptions -----------------------------------------------------

func TestFilterSubscriptions(t *testing.T) {
	newSub := func(name, deploymentType string) *subscriptions.Subscription {
		return &subscriptions.Subscription{
			Name:           redis.String(name),
			DeploymentType: redis.String(deploymentType),
		}
	}

	aaAlpha := newSub("alpha", subscriptions.SubscriptionDeploymentTypeActiveActive)
	singleBeta := newSub("beta", subscriptions.SubscriptionDeploymentTypeSingleRegion)
	singleAlpha := newSub("alpha", subscriptions.SubscriptionDeploymentTypeSingleRegion)
	subs := []*subscriptions.Subscription{aaAlpha, singleBeta, singleAlpha}

	// Use the real constructors so these compose-and-apply cases exercise the actual
	// filter predicates, not stand-ins.
	isActiveActive := utils.ActiveActiveSubscriptionFilter()
	namedAlpha := utils.SubscriptionNameFilter("alpha")

	t.Run("no filters returns everything", func(t *testing.T) {
		got := utils.FilterSubscriptions(subs, nil)
		assert.Equal(t, subs, got)
	})

	t.Run("single filter narrows the set", func(t *testing.T) {
		got := utils.FilterSubscriptions(subs, []utils.SubscriptionFilter{isActiveActive})
		assert.Equal(t, []*subscriptions.Subscription{aaAlpha}, got)
	})

	t.Run("filters are ANDed together", func(t *testing.T) {
		got := utils.FilterSubscriptions(subs, []utils.SubscriptionFilter{isActiveActive, namedAlpha})
		assert.Equal(t, []*subscriptions.Subscription{aaAlpha}, got)
	})

	t.Run("a name filter keeps every match", func(t *testing.T) {
		got := utils.FilterSubscriptions(subs, []utils.SubscriptionFilter{namedAlpha})
		assert.Equal(t, []*subscriptions.Subscription{aaAlpha, singleAlpha}, got)
	})

	t.Run("no matches returns empty", func(t *testing.T) {
		got := utils.FilterSubscriptions(subs, []utils.SubscriptionFilter{
			func(*subscriptions.Subscription) bool { return false },
		})
		assert.Empty(t, got)
	})
}

// --- Filter constructors -----------------------------------------------------

func TestActiveActiveSubscriptionFilter(t *testing.T) {
	f := utils.ActiveActiveSubscriptionFilter()
	aa := &subscriptions.Subscription{DeploymentType: redis.String(subscriptions.SubscriptionDeploymentTypeActiveActive)}
	single := &subscriptions.Subscription{DeploymentType: redis.String(subscriptions.SubscriptionDeploymentTypeSingleRegion)}

	assert.True(t, f(aa), "should match an active-active subscription")
	assert.False(t, f(single), "should not match a single-region subscription")
}

func TestProSubscriptionFilter(t *testing.T) {
	f := utils.ProSubscriptionFilter()
	single := &subscriptions.Subscription{DeploymentType: redis.String(subscriptions.SubscriptionDeploymentTypeSingleRegion)}
	aa := &subscriptions.Subscription{DeploymentType: redis.String(subscriptions.SubscriptionDeploymentTypeActiveActive)}
	other := &subscriptions.Subscription{DeploymentType: redis.String("other")}

	assert.True(t, f(single), "should match a single-region subscription")
	assert.False(t, f(aa), "should not match an active-active subscription")
	assert.False(t, f(other), "should not match an unset/other deployment type")
}

func TestSubscriptionNameFilter(t *testing.T) {
	f := utils.SubscriptionNameFilter("alpha")

	assert.True(t, f(&subscriptions.Subscription{Name: redis.String("alpha")}), "should match the given name")
	assert.False(t, f(&subscriptions.Subscription{Name: redis.String("beta")}), "should not match a different name")
}

// --- PricingFromAPI ----------------------------------------------------------

func decodePricing(ctx context.Context, t *testing.T, prices []*pricing.Pricing) []utils.PricingModel {
	t.Helper()
	list, diags := utils.PricingFromAPI(ctx, prices)
	require.False(t, diags.HasError())
	var got []utils.PricingModel
	require.False(t, list.ElementsAs(ctx, &got, false).HasError())
	return got
}

func TestPricingFromAPI(t *testing.T) {
	ctx := context.Background()

	t.Run("sorts deterministically regardless of input order", func(t *testing.T) {
		p1 := &pricing.Pricing{Region: redis.String("us-east-1"), Type: redis.String("MinimumPrice")}
		p2 := &pricing.Pricing{Region: redis.String("us-east-2"), Type: redis.String("MinimumPrice")}

		forward := decodePricing(ctx, t, []*pricing.Pricing{p1, p2})
		reversed := decodePricing(ctx, t, []*pricing.Pricing{p2, p1})

		require.Len(t, forward, 2)
		require.Len(t, reversed, 2)
		assert.Equal(t, "us-east-1", forward[0].Region.ValueString())
		assert.Equal(t, "us-east-2", forward[1].Region.ValueString())
		assert.Equal(t, "us-east-1", reversed[0].Region.ValueString())
		assert.Equal(t, "us-east-2", reversed[1].Region.ValueString())
	})

	t.Run("maps fields with null-for-nil strings and zero-for-nil ints", func(t *testing.T) {
		prices := []*pricing.Pricing{
			{
				Type:                redis.String("Shards"),
				QuantityMeasurement: redis.String("shards"),
				PricePerUnit:        redis.Float64(0.5),
				PriceCurrency:       redis.String("USD"),
				PricePeriod:         redis.String("hour"),
				Region:              redis.String("us-east-1"),
			},
		}

		got := decodePricing(ctx, t, prices)
		require.Len(t, got, 1)

		assert.Equal(t, "Shards", got[0].Type.ValueString())
		assert.Equal(t, "USD", got[0].PriceCurrency.ValueString())
		assert.Equal(t, 0.5, got[0].PricePerUnit.ValueFloat64())
		assert.True(t, got[0].DatabaseName.IsNull())
		assert.True(t, got[0].TypeDetails.IsNull())
		assert.False(t, got[0].Quantity.IsNull())
		assert.EqualValues(t, 0, got[0].Quantity.ValueInt64())
	})

	t.Run("empty input yields an empty but non-null list", func(t *testing.T) {
		list, diags := utils.PricingFromAPI(ctx, nil)
		require.False(t, diags.HasError())
		assert.False(t, list.IsNull())
		assert.Empty(t, list.Elements())
	})
}

// --- ResourceTagsFromAPI ----------------------------------------------------

func TestResourceTagsFromAPI(t *testing.T) {
	ctx := context.Background()

	t.Run("maps key/value pairs", func(t *testing.T) {
		tags := []*subscriptions.ResourceTag{
			{Key: redis.String("environment"), Value: redis.String("test")},
			{Key: redis.String("owner"), Value: redis.String("team-a")},
		}
		m, diags := utils.ResourceTagsFromAPI(ctx, tags)
		require.False(t, diags.HasError())
		require.False(t, m.IsNull())

		var got map[string]string
		require.False(t, m.ElementsAs(ctx, &got, false).HasError())
		assert.Equal(t, map[string]string{"environment": "test", "owner": "team-a"}, got)
	})

	t.Run("empty input yields an empty but non-null map", func(t *testing.T) {
		m, diags := utils.ResourceTagsFromAPI(ctx, nil)
		require.False(t, diags.HasError())
		assert.False(t, m.IsNull())
		assert.Empty(t, m.Elements())
	})
}

type testNetwork struct {
	NetworkingSubnetID       types.String `tfsdk:"networking_subnet_id"`
	NetworkingDeploymentCIDR types.String `tfsdk:"networking_deployment_cidr"`
	NetworkingVpcID          types.String `tfsdk:"networking_vpc_id"`
}

type testRegion struct {
	Region                     types.String `tfsdk:"region"`
	MultipleAvailabilityZones  types.Bool   `tfsdk:"multiple_availability_zones"`
	PreferredAvailabilityZones types.List   `tfsdk:"preferred_availability_zones"`
	NetworkingVpcID            types.String `tfsdk:"networking_vpc_id"`
	Networks                   types.List   `tfsdk:"networks"`
}

type testCloudProvider struct {
	Provider       types.String `tfsdk:"provider"`
	CloudAccountID types.String `tfsdk:"cloud_account_id"`
	AwsAccountID   types.String `tfsdk:"aws_account_id"`
	ResourceTags   types.Map    `tfsdk:"resource_tags"`
	Region         types.Set    `tfsdk:"region"`
}

func TestCloudProvidersFromAPI(t *testing.T) {
	ctx := context.Background()

	t.Run("maps the full nested cloud_provider structure", func(t *testing.T) {
		cd := &subscriptions.CloudDetail{
			Provider:       redis.String("AWS"),
			CloudAccountID: redis.Int(123),
			AWSAccountID:   redis.String("123456789012"),
			ResourceTags: []*subscriptions.ResourceTag{
				{Key: redis.String("environment"), Value: redis.String("production")},
				{Key: redis.String("team"), Value: redis.String("platform")},
			},
			Regions: []*subscriptions.Region{
				{
					Region:                     redis.String("eu-west-1"),
					MultipleAvailabilityZones:  redis.Bool(false),
					PreferredAvailabilityZones: []*string{redis.String("euw1-az1"), redis.String("euw1-az2")},
					Networking: []*subscriptions.Networking{
						{
							SubnetID:       redis.String("subnet-abc"),
							DeploymentCIDR: redis.String("10.0.0.0/24"),
							VPCId:          redis.String("vpc-xyz"),
						},
					},
				},
			},
		}

		list, diags := utils.CloudProvidersFromAPI(ctx, []*subscriptions.CloudDetail{cd})
		require.False(t, diags.HasError())
		require.False(t, list.IsNull())

		var cps []testCloudProvider
		require.False(t, list.ElementsAs(ctx, &cps, false).HasError())
		require.Len(t, cps, 1)

		assert.Equal(t, "AWS", cps[0].Provider.ValueString())
		// cloud_account_id is always the stringified int, never null.
		assert.Equal(t, "123", cps[0].CloudAccountID.ValueString())
		assert.Equal(t, "123456789012", cps[0].AwsAccountID.ValueString())

		var tags map[string]string
		require.False(t, cps[0].ResourceTags.ElementsAs(ctx, &tags, false).HasError())
		assert.Equal(t, map[string]string{"environment": "production", "team": "platform"}, tags)

		var regions []testRegion
		require.False(t, cps[0].Region.ElementsAs(ctx, &regions, false).HasError())
		require.Len(t, regions, 1)
		assert.Equal(t, "eu-west-1", regions[0].Region.ValueString())
		assert.False(t, regions[0].MultipleAvailabilityZones.ValueBool())
		assert.False(t, regions[0].NetworkingVpcID.IsNull())
		assert.Empty(t, regions[0].NetworkingVpcID.ValueString())

		var azs []string
		require.False(t, regions[0].PreferredAvailabilityZones.ElementsAs(ctx, &azs, false).HasError())
		assert.Equal(t, []string{"euw1-az1", "euw1-az2"}, azs)

		var networks []testNetwork
		require.False(t, regions[0].Networks.ElementsAs(ctx, &networks, false).HasError())
		require.Len(t, networks, 1)
		assert.Equal(t, "subnet-abc", networks[0].NetworkingSubnetID.ValueString())
		assert.Equal(t, "10.0.0.0/24", networks[0].NetworkingDeploymentCIDR.ValueString())
		assert.Equal(t, "vpc-xyz", networks[0].NetworkingVpcID.ValueString())
	})

	t.Run("nil optionals use known zero values", func(t *testing.T) {
		cd := &subscriptions.CloudDetail{
			Provider: redis.String("GCP"),
		}

		list, diags := utils.CloudProvidersFromAPI(ctx, []*subscriptions.CloudDetail{cd})
		require.False(t, diags.HasError())

		var cps []testCloudProvider
		require.False(t, list.ElementsAs(ctx, &cps, false).HasError())
		require.Len(t, cps, 1)

		assert.Equal(t, "GCP", cps[0].Provider.ValueString())
		assert.Equal(t, "0", cps[0].CloudAccountID.ValueString())
		assert.False(t, cps[0].AwsAccountID.IsNull())
		assert.Empty(t, cps[0].AwsAccountID.ValueString())
		assert.False(t, cps[0].ResourceTags.IsNull())
		assert.Empty(t, cps[0].ResourceTags.Elements())
		assert.False(t, cps[0].Region.IsNull())
		assert.Empty(t, cps[0].Region.Elements())
	})

	t.Run("nil region collections become known empty values", func(t *testing.T) {
		cd := &subscriptions.CloudDetail{
			Regions: []*subscriptions.Region{
				{Region: redis.String("eu-west-1")},
			},
		}

		list, diags := utils.CloudProvidersFromAPI(ctx, []*subscriptions.CloudDetail{cd})
		require.False(t, diags.HasError())

		var cloudProviders []testCloudProvider
		require.False(t, list.ElementsAs(ctx, &cloudProviders, false).HasError())
		require.Len(t, cloudProviders, 1)

		var regions []testRegion
		require.False(t, cloudProviders[0].Region.ElementsAs(ctx, &regions, false).HasError())
		require.Len(t, regions, 1)
		assert.False(t, regions[0].PreferredAvailabilityZones.IsNull())
		assert.Empty(t, regions[0].PreferredAvailabilityZones.Elements())
		assert.False(t, regions[0].Networks.IsNull())
		assert.Empty(t, regions[0].Networks.Elements())
	})

	t.Run("empty input yields an empty but non-null list", func(t *testing.T) {
		list, diags := utils.CloudProvidersFromAPI(ctx, nil)
		require.False(t, diags.HasError())
		assert.False(t, list.IsNull())
		assert.Empty(t, list.Elements())
	})
}
