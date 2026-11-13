package peering

import (
	"context"
	"strconv"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (d *subscriptionPeeringsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config SubscriptionPeeringsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	subId, err := strconv.Atoi(config.SubscriptionID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid subscription_id", err.Error())
		return
	}

	vpcPeerings, err := d.client.Client.Subscription.ListVPCPeering(ctx, subId)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list subscription peerings", err.Error())
		return
	}

	id := "ALL"
	if status := config.Status.ValueString(); status != "" {
		vpcPeerings = filterVPCPeeringsByStatus(vpcPeerings, status)
		id = status
	}

	peerings, diags := PeeringsFromAPI(ctx, vpcPeerings)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	config.ID = types.StringValue(id)
	config.Peerings = peerings

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

func filterVPCPeeringsByStatus(vpcPeerings []*subscriptions.VPCPeering, status string) []*subscriptions.VPCPeering {
	var filtered []*subscriptions.VPCPeering
	for _, vpcPeering := range vpcPeerings {
		if redis.StringValue(vpcPeering.Status) == status {
			filtered = append(filtered, vpcPeering)
		}
	}
	return filtered
}

// PeeringsFromAPI maps API VPC peerings to the data source peerings set.
func PeeringsFromAPI(ctx context.Context, vpcPeerings []*subscriptions.VPCPeering) (types.Set, diag.Diagnostics) {
	elems := make([]PeeringModel, 0, len(vpcPeerings))
	for _, p := range vpcPeerings {
		//TODO(TF3.0) make the peering string fields nullable
		elems = append(elems, PeeringModel{
			PeeringID:           types.Int64Value(int64(redis.IntValue(p.ID))),
			ProviderName:        types.StringValue(providerNameFromVPCPeering(p)),
			AWSAccountID:        types.StringValue(redis.StringValue(p.AWSAccountID)),
			VPCID:               types.StringValue(redis.StringValue(p.VPCId)),
			VPCCidr:             types.StringValue(redis.StringValue(p.VPCCidr)),
			GCPProjectID:        types.StringValue(redis.StringValue(p.GCPProjectUID)),
			GCPNetworkName:      types.StringValue(redis.StringValue(p.NetworkName)),
			Status:              types.StringValue(redis.StringValue(p.Status)),
			AWSPeeringID:        types.StringValue(redis.StringValue(p.AWSPeeringID)),
			Region:              types.StringValue(redis.StringValue(p.Region)),
			GCPRedisProjectID:   types.StringValue(redis.StringValue(p.RedisProjectUID)),
			GCPRedisNetworkName: types.StringValue(redis.StringValue(p.RedisNetworkName)),
			GCPPeeringID:        types.StringValue(redis.StringValue(p.CloudPeeringID)),
		})
	}
	return types.SetValueFrom(ctx, peeringElemType, elems)
}

func providerNameFromVPCPeering(vpcPeering *subscriptions.VPCPeering) string {
	if redis.StringValue(vpcPeering.VPCId) != "" {
		return "AWS"
	}
	if redis.StringValue(vpcPeering.GCPProjectUID) != "" {
		return "GCP"
	}
	return "Unknown"
}
