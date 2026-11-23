package peering

import (
	"context"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPeeringsFromAPI(t *testing.T) {
	ctx := context.Background()

	t.Run("nil yields a known empty set", func(t *testing.T) {
		set, diags := PeeringsFromAPI(ctx, nil)
		require.False(t, diags.HasError())
		assert.False(t, set.IsNull())
		assert.False(t, set.IsUnknown())
		assert.Empty(t, set.Elements())
	})

	t.Run("missing fields map to empty strings and zero", func(t *testing.T) {
		set, diags := PeeringsFromAPI(ctx, []*subscriptions.VPCPeering{
			{ID: redis.Int(1), Status: redis.String("active"), VPCId: redis.String("vpc-1"), AWSAccountID: redis.String("123"), Region: redis.String("eu-west-1")},
			{Status: redis.String("pending-acceptance"), GCPProjectUID: redis.String("proj"), NetworkName: redis.String("net")},
		})
		require.False(t, diags.HasError())

		var models []PeeringModel
		require.False(t, set.ElementsAs(ctx, &models, false).HasError())
		require.Len(t, models, 2)

		byProvider := map[string]PeeringModel{}
		for _, m := range models {
			byProvider[m.ProviderName.ValueString()] = m
		}

		aws := byProvider["AWS"]
		assert.Equal(t, int64(1), aws.PeeringID.ValueInt64())
		assert.Equal(t, "vpc-1", aws.VPCID.ValueString())
		assert.False(t, aws.GCPProjectID.IsNull())
		assert.Equal(t, "", aws.GCPProjectID.ValueString())

		gcp := byProvider["GCP"]
		assert.Equal(t, int64(0), gcp.PeeringID.ValueInt64())
		assert.Equal(t, "net", gcp.GCPNetworkName.ValueString())
		assert.False(t, gcp.VPCID.IsNull())
		assert.Equal(t, "", gcp.VPCID.ValueString())
	})
}

func TestProviderNameFromVPCPeering(t *testing.T) {
	assert.Equal(t, "AWS", providerNameFromVPCPeering(&subscriptions.VPCPeering{VPCId: redis.String("vpc-1")}))
	assert.Equal(t, "GCP", providerNameFromVPCPeering(&subscriptions.VPCPeering{GCPProjectUID: redis.String("p")}))
	assert.Equal(t, "Unknown", providerNameFromVPCPeering(&subscriptions.VPCPeering{VPCId: redis.String("")}))
}

func TestFilterVPCPeeringsByStatus(t *testing.T) {
	peerings := []*subscriptions.VPCPeering{
		{ID: redis.Int(1), Status: redis.String("active")},
		{ID: redis.Int(2), Status: redis.String("failed")},
		{ID: redis.Int(3)},
	}
	filtered := filterVPCPeeringsByStatus(peerings, "active")
	require.Len(t, filtered, 1)
	assert.Equal(t, 1, redis.IntValue(filtered[0].ID))
	assert.Empty(t, filterVPCPeeringsByStatus(peerings, "inactive"))
}
