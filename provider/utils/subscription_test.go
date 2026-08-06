package utils_test

import (
	"context"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/maintenance"
	"github.com/RedisLabs/rediscloud-go-api/service/pricing"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func strPtrs(vals ...string) []*string {
	out := make([]*string, len(vals))
	for i, v := range vals {
		out[i] = redis.String(v)
	}
	return out
}

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

// --- FlattenMaintenance ------------------------------------------------------

type testWindow struct {
	StartHour       types.Int64 `tfsdk:"start_hour"`
	DurationInHours types.Int64 `tfsdk:"duration_in_hours"`
	Days            types.List  `tfsdk:"days"`
}

type testMaintenance struct {
	Mode   types.String `tfsdk:"mode"`
	Window types.List   `tfsdk:"window"`
}

func TestFlattenMaintenance(t *testing.T) {
	ctx := context.Background()

	t.Run("nil maintenance yields a null list", func(t *testing.T) {
		list, diags := utils.FlattenMaintenance(ctx, nil)
		require.False(t, diags.HasError())
		assert.True(t, list.IsNull())
	})

	t.Run("manual mode maps windows and days", func(t *testing.T) {
		m := &maintenance.Maintenance{
			Mode: redis.String("manual"),
			Windows: []*maintenance.Window{
				{StartHour: redis.Int(22), DurationInHours: redis.Int(8), Days: strPtrs("Monday", "Thursday")},
				{StartHour: redis.Int(12), DurationInHours: redis.Int(6), Days: strPtrs("Friday", "Saturday", "Sunday")},
			},
		}

		list, diags := utils.FlattenMaintenance(ctx, m)
		require.False(t, diags.HasError())
		require.False(t, list.IsNull())

		var entries []testMaintenance
		require.False(t, list.ElementsAs(ctx, &entries, false).HasError())
		require.Len(t, entries, 1)
		assert.Equal(t, "manual", entries[0].Mode.ValueString())

		var windows []testWindow
		require.False(t, entries[0].Window.ElementsAs(ctx, &windows, false).HasError())
		require.Len(t, windows, 2)

		assert.EqualValues(t, 22, windows[0].StartHour.ValueInt64())
		assert.EqualValues(t, 8, windows[0].DurationInHours.ValueInt64())
		var days0 []string
		require.False(t, windows[0].Days.ElementsAs(ctx, &days0, false).HasError())
		assert.Equal(t, []string{"Monday", "Thursday"}, days0)

		assert.EqualValues(t, 12, windows[1].StartHour.ValueInt64())
		var days1 []string
		require.False(t, windows[1].Days.ElementsAs(ctx, &days1, false).HasError())
		assert.Equal(t, []string{"Friday", "Saturday", "Sunday"}, days1)
	})

	t.Run("automatic mode has an empty but non-null window list", func(t *testing.T) {
		m := &maintenance.Maintenance{Mode: redis.String("automatic")}

		list, diags := utils.FlattenMaintenance(ctx, m)
		require.False(t, diags.HasError())

		var entries []testMaintenance
		require.False(t, list.ElementsAs(ctx, &entries, false).HasError())
		require.Len(t, entries, 1)
		assert.Equal(t, "automatic", entries[0].Mode.ValueString())
		assert.False(t, entries[0].Window.IsNull())
		assert.Empty(t, entries[0].Window.Elements())
	})
}

// --- FlattenPricing ----------------------------------------------------------

type testPricing struct {
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

func decodePricing(ctx context.Context, t *testing.T, prices []*pricing.Pricing) []testPricing {
	t.Helper()
	list, diags := utils.FlattenPricing(ctx, prices)
	require.False(t, diags.HasError())
	var got []testPricing
	require.False(t, list.ElementsAs(ctx, &got, false).HasError())
	return got
}

func TestFlattenPricing(t *testing.T) {
	ctx := context.Background()

	t.Run("sorts deterministically regardless of input order", func(t *testing.T) {
		p1 := &pricing.Pricing{Region: redis.String("us-east-1"), Type: redis.String("MinimumPrice")}
		p2 := &pricing.Pricing{Region: redis.String("us-east-2"), Type: redis.String("MinimumPrice")}

		forward := decodePricing(ctx, t, []*pricing.Pricing{p1, p2})
		reversed := decodePricing(ctx, t, []*pricing.Pricing{p2, p1})

		require.Len(t, forward, 2)
		require.Len(t, reversed, 2)
		// Both input orders converge to the same region ordering — this is the
		// regression guard for the perpetual-plan-diff bug.
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
				// DatabaseName, TypeDetails, Quantity intentionally left nil
			},
		}

		got := decodePricing(ctx, t, prices)
		require.Len(t, got, 1)

		assert.Equal(t, "Shards", got[0].Type.ValueString())
		assert.Equal(t, "USD", got[0].PriceCurrency.ValueString())
		assert.InEpsilon(t, 0.5, got[0].PricePerUnit.ValueFloat64(), 1e-9)

		// nil *string -> null
		assert.True(t, got[0].DatabaseName.IsNull())
		assert.True(t, got[0].TypeDetails.IsNull())
		// nil *int -> known 0, not null
		assert.False(t, got[0].Quantity.IsNull())
		assert.EqualValues(t, 0, got[0].Quantity.ValueInt64())
	})

	t.Run("empty input yields an empty but non-null list", func(t *testing.T) {
		list, diags := utils.FlattenPricing(ctx, nil)
		require.False(t, diags.HasError())
		assert.False(t, list.IsNull())
		assert.Empty(t, list.Elements())
	})
}

// --- FlattenResourceTags -----------------------------------------------------

func TestFlattenResourceTags(t *testing.T) {
	ctx := context.Background()

	t.Run("maps key/value pairs", func(t *testing.T) {
		tags := []*subscriptions.ResourceTag{
			{Key: redis.String("environment"), Value: redis.String("test")},
			{Key: redis.String("owner"), Value: redis.String("team-a")},
		}
		m, diags := utils.FlattenResourceTags(ctx, tags)
		require.False(t, diags.HasError())
		require.False(t, m.IsNull())

		var got map[string]string
		require.False(t, m.ElementsAs(ctx, &got, false).HasError())
		assert.Equal(t, map[string]string{"environment": "test", "owner": "team-a"}, got)
	})

	t.Run("empty input yields an empty but non-null map", func(t *testing.T) {
		m, diags := utils.FlattenResourceTags(ctx, nil)
		require.False(t, diags.HasError())
		assert.False(t, m.IsNull())
		assert.Empty(t, m.Elements())
	})
}
