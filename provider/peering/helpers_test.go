package peering

import (
	"context"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePeeringID(t *testing.T) {
	sub, id, err := parsePeeringID("123/456")
	require.NoError(t, err)
	assert.Equal(t, 123, sub)
	assert.Equal(t, 456, id)

	for _, invalid := range []string{"", "123", "123/456/789", "abc/456", "123/abc"} {
		_, _, err := parsePeeringID(invalid)
		assert.Error(t, err, invalid)
	}
}

func TestFindVPCPeering(t *testing.T) {
	peerings := []*subscriptions.VPCPeering{{ID: redis.Int(1)}, {ID: redis.Int(2)}}
	assert.Same(t, peerings[1], findVPCPeering(2, peerings))
	assert.Nil(t, findVPCPeering(3, peerings))
}

func TestVPCCidrsFromAPI(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name      string
		vpcCidr   *string
		vpcCidrs  []*subscriptions.CIDR
		wantCidr  string
		wantCidrs []string
	}{
		{
			name:      "single cidr only",
			vpcCidr:   redis.String("10.0.0.0/24"),
			wantCidr:  "10.0.0.0/24",
			wantCidrs: []string{"10.0.0.0/24"},
		},
		{
			name:      "cidrs only fall back to the first for vpc_cidr",
			vpcCidrs:  []*subscriptions.CIDR{{VPCCidr: redis.String("10.0.1.0/24")}, {VPCCidr: redis.String("10.0.2.0/24")}},
			wantCidr:  "10.0.1.0/24",
			wantCidrs: []string{"10.0.1.0/24", "10.0.2.0/24"},
		},
		{
			name:      "both set keep vpc_cidr",
			vpcCidr:   redis.String("10.0.0.0/24"),
			vpcCidrs:  []*subscriptions.CIDR{{VPCCidr: redis.String("10.0.1.0/24")}},
			wantCidr:  "10.0.0.0/24",
			wantCidrs: []string{"10.0.1.0/24"},
		},
		{
			name:      "none set yields empty strings as SDKv2 did",
			wantCidr:  "",
			wantCidrs: []string{""},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cidr, cidrs, diags := vpcCidrsFromAPI(ctx, tc.vpcCidr, tc.vpcCidrs)
			require.False(t, diags.HasError())
			assert.Equal(t, types.StringValue(tc.wantCidr), cidr)

			var got []string
			require.False(t, cidrs.ElementsAs(ctx, &got, false).HasError())
			assert.ElementsMatch(t, tc.wantCidrs, got)
		})
	}
}

func TestVPCCidrsFromPlan(t *testing.T) {
	ctx := context.Background()

	got, diags := vpcCidrsFromPlan(ctx, types.SetUnknown(types.StringType))
	require.False(t, diags.HasError())
	assert.Nil(t, got)

	got, diags = vpcCidrsFromPlan(ctx, types.SetNull(types.StringType))
	require.False(t, diags.HasError())
	assert.Nil(t, got)

	set, _ := types.SetValueFrom(ctx, types.StringType, []string{"10.0.0.0/24"})
	got, diags = vpcCidrsFromPlan(ctx, set)
	require.False(t, diags.HasError())
	require.Len(t, got, 1)
	assert.Equal(t, "10.0.0.0/24", redis.StringValue(got[0]))
}

func TestIsSet(t *testing.T) {
	assert.False(t, isSet(types.StringNull()))
	assert.False(t, isSet(types.StringUnknown()))
	assert.False(t, isSet(types.StringValue("")))
	assert.True(t, isSet(types.StringValue("x")))
}

func TestCidrValidator(t *testing.T) {
	ctx := context.Background()
	for value, wantErr := range map[string]bool{
		"10.0.0.0/24": false,
		"10.0.0.0":    true,
		"not-a-cidr":  true,
	} {
		resp := &validator.StringResponse{}
		cidrValidator{}.ValidateString(ctx, validator.StringRequest{Path: path.Root("vpc_cidr"), ConfigValue: types.StringValue(value)}, resp)
		assert.Equal(t, wantErr, resp.Diagnostics.HasError(), value)
	}

	resp := &validator.StringResponse{}
	cidrValidator{}.ValidateString(ctx, validator.StringRequest{Path: path.Root("vpc_cidr"), ConfigValue: types.StringUnknown()}, resp)
	assert.False(t, resp.Diagnostics.HasError())
}

func TestSubscriptionPeeringNullUnknowns(t *testing.T) {
	m := SubscriptionPeeringResourceModel{
		Region:       types.StringValue("eu-west-1"),
		GCPProjectID: types.StringUnknown(),
		VPCCidrs:     types.SetUnknown(types.StringType),
		Status:       types.StringUnknown(),
	}
	m.nullUnknowns()
	assert.Equal(t, types.StringValue("eu-west-1"), m.Region)
	assert.True(t, m.GCPProjectID.IsNull())
	assert.True(t, m.Status.IsNull())
	assert.Equal(t, types.SetValueMust(types.StringType, []attr.Value{}), m.VPCCidrs)
}

func TestEmptySetIfUnset(t *testing.T) {
	empty := types.SetValueMust(types.StringType, []attr.Value{})
	for _, v := range []types.Set{types.SetNull(types.StringType), types.SetUnknown(types.StringType)} {
		emptySetIfUnset(&v)
		assert.Equal(t, empty, v)
	}

	v := types.SetValueMust(types.StringType, []attr.Value{types.StringValue("10.0.0.0/16")})
	emptySetIfUnset(&v)
	assert.Len(t, v.Elements(), 1)
}

func TestFindActiveActiveVPCPeering(t *testing.T) {
	regions := []*subscriptions.ActiveActiveVpcRegion{
		{SourceRegion: redis.String("eu-west-1"), VPCPeerings: []*subscriptions.ActiveActiveVPCPeering{{ID: redis.Int(1)}}},
		{SourceRegion: redis.String("us-east-1"), VPCPeerings: []*subscriptions.ActiveActiveVPCPeering{{ID: redis.Int(2)}, {ID: redis.Int(3)}}},
	}

	peering, sourceRegion := findActiveActiveVPCPeering(3, regions)
	require.NotNil(t, peering)
	assert.Equal(t, 3, redis.IntValue(peering.ID))
	assert.Equal(t, "us-east-1", redis.StringValue(sourceRegion))

	peering, sourceRegion = findActiveActiveVPCPeering(4, regions)
	assert.Nil(t, peering)
	assert.Nil(t, sourceRegion)
}

func TestActiveActiveSubscriptionPeeringNullUnknowns(t *testing.T) {
	m := ActiveActiveSubscriptionPeeringResourceModel{
		SourceRegion:      types.StringValue("europe-west2"),
		DestinationRegion: types.StringUnknown(),
		VPCCidrs:          types.SetUnknown(types.StringType),
	}
	m.nullUnknowns()
	assert.Equal(t, types.StringValue("europe-west2"), m.SourceRegion)
	assert.True(t, m.DestinationRegion.IsNull())
	assert.Equal(t, types.SetValueMust(types.StringType, []attr.Value{}), m.VPCCidrs)
}
