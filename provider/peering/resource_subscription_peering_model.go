package peering

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// SubscriptionPeeringResourceModel describes the resource data model.
type SubscriptionPeeringResourceModel struct {
	ID                  types.String   `tfsdk:"id"`
	SubscriptionID      types.String   `tfsdk:"subscription_id"`
	ProviderName        types.String   `tfsdk:"provider_name"`
	Region              types.String   `tfsdk:"region"`
	AWSAccountID        types.String   `tfsdk:"aws_account_id"`
	VPCID               types.String   `tfsdk:"vpc_id"`
	VPCCidr             types.String   `tfsdk:"vpc_cidr"`
	VPCCidrs            types.Set      `tfsdk:"vpc_cidrs"`
	GCPProjectID        types.String   `tfsdk:"gcp_project_id"`
	GCPNetworkName      types.String   `tfsdk:"gcp_network_name"`
	Status              types.String   `tfsdk:"status"`
	AWSPeeringID        types.String   `tfsdk:"aws_peering_id"`
	GCPRedisProjectID   types.String   `tfsdk:"gcp_redis_project_id"`
	GCPRedisNetworkName types.String   `tfsdk:"gcp_redis_network_name"`
	GCPPeeringID        types.String   `tfsdk:"gcp_peering_id"`
	Timeouts            timeouts.Value `tfsdk:"timeouts"`
}

func (m *SubscriptionPeeringResourceModel) nullUnknowns() {
	for _, v := range []*types.String{
		&m.Region, &m.AWSAccountID, &m.VPCID, &m.VPCCidr, &m.GCPProjectID, &m.GCPNetworkName,
		&m.Status, &m.AWSPeeringID, &m.GCPRedisProjectID, &m.GCPRedisNetworkName, &m.GCPPeeringID,
	} {
		nullIfUnknown(v)
	}
	emptySetIfUnset(&m.VPCCidrs)
}
