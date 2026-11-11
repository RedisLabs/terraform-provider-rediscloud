package privatelink

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ActiveActivePrivateLinkResourceModel maps the rediscloud_active_active_private_link resource schema.
type ActiveActivePrivateLinkResourceModel struct {
	ID                       types.String      `tfsdk:"id"`
	SubscriptionID           types.String      `tfsdk:"subscription_id"`
	RegionID                 types.Int64       `tfsdk:"region_id"`
	ShareName                types.String      `tfsdk:"share_name"`
	Principal                PrincipalSetValue `tfsdk:"principal"`
	ResourceConfigurationID  types.String      `tfsdk:"resource_configuration_id"`
	ResourceConfigurationArn types.String      `tfsdk:"resource_configuration_arn"`
	ShareArn                 types.String      `tfsdk:"share_arn"`
	Connections              types.Set         `tfsdk:"connections"`
	Databases                types.Set         `tfsdk:"databases"`
	Timeouts                 timeouts.Value    `tfsdk:"timeouts"`
}
