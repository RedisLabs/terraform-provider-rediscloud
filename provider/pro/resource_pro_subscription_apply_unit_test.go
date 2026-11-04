package pro_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/pro"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func TestProSubscriptionCreateStoresAPINetworks(t *testing.T) {
	ctx := context.Background()
	s := proSubscriptionSchema(t)
	prior := proSubscriptionPlanState(t, s)
	zones := tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, "eu-west-1a")})
	config := setValues(t, configFromState(t, s, prior), isRegionAttr("preferred_availability_zones"), zones)
	null := tftypes.NewValue(s.Type().TerraformType(ctx), nil)

	planned, _ := planProSubscription(t, s, null, config, config)

	fake := testhelpers.NewFakeAPI()
	fake.Handle("GET /subscriptions/123", func(*http.Request) (any, error) {
		return map[string]any{
			"id": 123, "name": "sub", "status": "active", "paymentMethodType": "credit-card", "memoryStorage": "ram",
			"cloudDetails": []any{map[string]any{
				"provider": "AWS", "cloudAccountId": 1, "awsAccountId": "123456789012",
				"regions": []any{map[string]any{
					"region": "eu-west-1", "multipleAvailabilityZones": false, "preferredAvailabilityZones": []string{"eu-west-1a"},
					"networking": []any{map[string]any{"deploymentCIDR": "10.0.0.0/24", "vpcId": "vpc-api", "subnetId": "subnet-api"}},
				}},
			}},
		}, nil
	})
	fake.Handle("GET /subscriptions/123/maintenance-windows", func(*http.Request) (any, error) {
		return map[string]any{"mode": "automatic"}, nil
	})
	fake.Handle("GET /subscriptions/123/pricing", func(*http.Request) (any, error) {
		return map[string]any{"pricing": []any{}}, nil
	})

	state := tfsdk.State{Schema: s, Raw: planned}
	var model pro.ProSubscriptionResourceModel
	require.False(t, state.Get(ctx, &model).HasError())
	model.ID = types.StringValue("123")
	diags := pro.ReadProSubscriptionAfterApply(ctx, testhelpers.NewAPIClient(t, fake), 123, &model, &state)
	require.False(t, diags.HasError(), "%v", diags)

	var stored pro.ProSubscriptionResourceModel
	require.False(t, state.Get(ctx, &stored).HasError())
	var providers []pro.SubscriptionCloudProviderModel
	require.False(t, stored.CloudProvider.ElementsAs(ctx, &providers, false).HasError())
	var regions []pro.SubscriptionRegionModel
	require.False(t, providers[0].Region.ElementsAs(ctx, &regions, false).HasError())
	require.Len(t, regions, 1)
	var networks []utils.CloudNetworkModel
	require.False(t, regions[0].Networks.ElementsAs(ctx, &networks, false).HasError())
	require.Len(t, networks, 1, "networks must come from the API, got %s", regions[0].Networks)
	assert.Equal(t, "subnet-api", networks[0].NetworkingSubnetID.ValueString())
	assert.Equal(t, "vpc-api", regions[0].NetworkingVPCID.ValueString())
}
