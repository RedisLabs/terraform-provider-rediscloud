package psc

import "github.com/hashicorp/terraform-plugin-framework/types"

// PrivateServiceConnectModel maps the rediscloud_private_service_connect data source schema.
type PrivateServiceConnectModel struct {
	ID                             types.String `tfsdk:"id"`
	SubscriptionID                 types.String `tfsdk:"subscription_id"`
	PrivateServiceConnectServiceID types.Int64  `tfsdk:"private_service_connect_service_id"`
	ConnectionHostName             types.String `tfsdk:"connection_host_name"`
	ServiceAttachmentName          types.String `tfsdk:"service_attachment_name"`
	Status                         types.String `tfsdk:"status"`
}
