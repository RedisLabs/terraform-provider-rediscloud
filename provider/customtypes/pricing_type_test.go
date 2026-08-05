package customtypes_test

import (
	"context"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/pricing"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
)

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
	list, diags := customtypes.NewPricingList(ctx, prices)
	require.False(t, diags.HasError())
	var got []testPricing
	require.False(t, list.ElementsAs(ctx, &got, false).HasError())
	return got
}

func TestNewPricingList(t *testing.T) {
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

	t.Run("maps fields with empty-for-nil strings and zero-for-nil numbers", func(t *testing.T) {
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

		// nil *string -> "" (never null), matching the SDKv2 flatten for backward compat.
		assert.False(t, got[0].DatabaseName.IsNull())
		assert.Equal(t, "", got[0].DatabaseName.ValueString())
		assert.Equal(t, "", got[0].TypeDetails.ValueString())
		// nil *int -> known 0.
		assert.EqualValues(t, 0, got[0].Quantity.ValueInt64())
	})

	t.Run("empty input yields an empty but non-null list", func(t *testing.T) {
		list, diags := customtypes.NewPricingList(ctx, nil)
		require.False(t, diags.HasError())
		assert.False(t, list.IsNull())
		assert.Empty(t, list.Elements())
	})
}

// --- PricingType / PricingValue basics --------------------------------------

func TestPricingTypeEqual(t *testing.T) {
	assert.True(t, customtypes.NewPricingType().Equal(customtypes.NewPricingType()))
	// A bare object type of any shape is not equal to the custom type.
	assert.False(t, customtypes.NewPricingType().Equal(types.ObjectType{}))
}

func TestPricingValueTerraformRoundTrip(t *testing.T) {
	ctx := context.Background()

	orig, diags := customtypes.NewPricingValue(ctx, &pricing.Pricing{
		Region: redis.String("us-east-1"),
		Type:   redis.String("MinimumPrice"),
	})
	require.False(t, diags.HasError())

	// ToTerraformValue -> ValueFromTerraform round-trips back to a PricingValue.
	tv, err := orig.ToTerraformValue(ctx)
	require.NoError(t, err)

	back, err := customtypes.NewPricingType().ValueFromTerraform(ctx, tv)
	require.NoError(t, err)

	got, ok := back.(customtypes.PricingValue)
	require.True(t, ok, "ValueFromTerraform should yield a customtypes.PricingValue, got %T", back)
	assert.True(t, orig.Equal(got))
}

func TestPricingValueAs(t *testing.T) {
	ctx := context.Background()

	v, diags := customtypes.NewPricingValue(ctx, &pricing.Pricing{
		DatabaseName: redis.String("db-1"),
		Region:       redis.String("us-east-1"),
		Quantity:     redis.Int(3),
	})
	require.False(t, diags.HasError())

	// Typed, compile-checked field access instead of a string-keyed attribute map.
	m, diags := v.As(ctx)
	require.False(t, diags.HasError())
	assert.Equal(t, "db-1", m.DatabaseName.ValueString())
	assert.Equal(t, "us-east-1", m.Region.ValueString())
	assert.EqualValues(t, 3, m.Quantity.ValueInt64())
}
