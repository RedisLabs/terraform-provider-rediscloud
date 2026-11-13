package psc

import "github.com/hashicorp/terraform-plugin-framework/types"

// PrivateServiceConnectEndpointsModel maps the rediscloud_private_service_connect_endpoints data source schema.
type PrivateServiceConnectEndpointsModel struct {
	ID                             types.String `tfsdk:"id"`
	SubscriptionID                 types.String `tfsdk:"subscription_id"`
	PrivateServiceConnectServiceID types.Int64  `tfsdk:"private_service_connect_service_id"`
	Endpoints                      types.List   `tfsdk:"endpoints"`
}

// EndpointModel represents one endpoint of a Private Service Connect service.
type EndpointModel struct {
	PrivateServiceConnectEndpointID types.Int64  `tfsdk:"private_service_connect_endpoint_id"`
	GCPProjectID                    types.String `tfsdk:"gcp_project_id"`
	GCPVPCName                      types.String `tfsdk:"gcp_vpc_name"`
	GCPVPCSubnetName                types.String `tfsdk:"gcp_vpc_subnet_name"`
	EndpointConnectionName          types.String `tfsdk:"endpoint_connection_name"`
	Status                          types.String `tfsdk:"status"`
	ServiceAttachments              types.List   `tfsdk:"service_attachments"`
}
