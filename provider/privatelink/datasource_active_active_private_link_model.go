package privatelink

import "github.com/hashicorp/terraform-plugin-framework/types"

// ActiveActivePrivateLinkDataSourceModel describes the Active-Active subscription PrivateLink data source.
type ActiveActivePrivateLinkDataSourceModel struct {
	ID                       types.String `tfsdk:"id"`
	SubscriptionID           types.String `tfsdk:"subscription_id"`
	RegionID                 types.Int64  `tfsdk:"region_id"`
	Principals               types.List   `tfsdk:"principals"`
	ResourceConfigurationID  types.String `tfsdk:"resource_configuration_id"`
	ResourceConfigurationArn types.String `tfsdk:"resource_configuration_arn"`
	ShareArn                 types.String `tfsdk:"share_arn"`
	ShareName                types.String `tfsdk:"share_name"`
	Connections              types.Set    `tfsdk:"connections"`
	Databases                types.Set    `tfsdk:"databases"`
}
