package privatelink

import "github.com/hashicorp/terraform-plugin-framework/types"

// PrivateLinkDataSourceModel describes the Pro subscription PrivateLink data source.
type PrivateLinkDataSourceModel struct {
	ID                       types.String      `tfsdk:"id"`
	SubscriptionID           types.String      `tfsdk:"subscription_id"`
	ShareName                types.String      `tfsdk:"share_name"`
	Principals               PrincipalSetValue `tfsdk:"principals"`
	ResourceConfigurationID  types.String      `tfsdk:"resource_configuration_id"`
	ResourceConfigurationArn types.String      `tfsdk:"resource_configuration_arn"`
	ShareArn                 types.String      `tfsdk:"share_arn"`
	Connections              types.Set         `tfsdk:"connections"`
	Databases                types.Set         `tfsdk:"databases"`
}
