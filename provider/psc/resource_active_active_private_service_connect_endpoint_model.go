package psc

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ActiveActivePrivateServiceConnectEndpointResourceModel maps the rediscloud_active_active_private_service_connect_endpoint resource schema.
type ActiveActivePrivateServiceConnectEndpointResourceModel struct {
	ID                              types.String   `tfsdk:"id"`
	SubscriptionID                  types.String   `tfsdk:"subscription_id"`
	RegionID                        types.Int64    `tfsdk:"region_id"`
	PrivateServiceConnectServiceID  types.Int64    `tfsdk:"private_service_connect_service_id"`
	PrivateServiceConnectEndpointID types.Int64    `tfsdk:"private_service_connect_endpoint_id"`
	GCPProjectID                    types.String   `tfsdk:"gcp_project_id"`
	GCPVPCName                      types.String   `tfsdk:"gcp_vpc_name"`
	GCPVPCSubnetName                types.String   `tfsdk:"gcp_vpc_subnet_name"`
	EndpointConnectionName          types.String   `tfsdk:"endpoint_connection_name"`
	ServiceAttachments              types.List     `tfsdk:"service_attachments"`
	Timeouts                        timeouts.Value `tfsdk:"timeouts"`
}
