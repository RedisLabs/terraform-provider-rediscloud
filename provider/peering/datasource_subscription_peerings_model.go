package peering

import (
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
)

// SubscriptionPeeringsDataSourceModel describes the data source data model.
type SubscriptionPeeringsDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	SubscriptionID types.String `tfsdk:"subscription_id"`
	Status         types.String `tfsdk:"status"`
	Peerings       types.Set    `tfsdk:"peerings"`
}

// PeeringModel describes one element of the peerings set.
type PeeringModel struct {
	//TODO(TF3.0) make peering_id a string
	PeeringID           types.Int64  `tfsdk:"peering_id"`
	ProviderName        types.String `tfsdk:"provider_name"`
	AWSAccountID        types.String `tfsdk:"aws_account_id"`
	VPCID               types.String `tfsdk:"vpc_id"`
	VPCCidr             types.String `tfsdk:"vpc_cidr"`
	GCPProjectID        types.String `tfsdk:"gcp_project_id"`
	GCPNetworkName      types.String `tfsdk:"gcp_network_name"`
	Status              types.String `tfsdk:"status"`
	AWSPeeringID        types.String `tfsdk:"aws_peering_id"`
	Region              types.String `tfsdk:"region"`
	GCPRedisProjectID   types.String `tfsdk:"gcp_redis_project_id"`
	GCPRedisNetworkName types.String `tfsdk:"gcp_redis_network_name"`
	GCPPeeringID        types.String `tfsdk:"gcp_peering_id"`
}

var peeringElemType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(PeeringModel{})}
