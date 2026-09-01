package utils_test

import (
	"context"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/pricing"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
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

func TestSubscriptionNameFilter(t *testing.T) {
	f := utils.SubscriptionNameFilter("alpha")

	assert.True(t, f(&subscriptions.Subscription{Name: redis.String("alpha")}), "should match the given name")
	assert.False(t, f(&subscriptions.Subscription{Name: redis.String("beta")}), "should not match a different name")
}

// --- PricingListFromAPI ------------------------------------------------------

func decodePricing(ctx context.Context, t *testing.T, prices []*pricing.Pricing) []utils.PricingModel {
	t.Helper()
	list, diags := utils.PricingListFromAPI(ctx, prices)
	require.False(t, diags.HasError())
	var got []utils.PricingModel
	require.False(t, list.ElementsAs(ctx, &got, false).HasError())
	return got
}

func TestPricingListFromAPI(t *testing.T) {
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
		list, diags := utils.PricingListFromAPI(ctx, nil)
		require.False(t, diags.HasError())
		assert.False(t, list.IsNull())
		assert.Empty(t, list.Elements())
	})
}

// --- ResourceTagsMapFromAPI -------------------------------------------------

func TestResourceTagsMapFromAPI(t *testing.T) {
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
