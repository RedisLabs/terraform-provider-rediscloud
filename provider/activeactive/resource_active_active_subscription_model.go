package activeactive

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
)

// ActiveActiveSubscriptionResourceModel is the Terraform model for the
// rediscloud_active_active_subscription resource.
type ActiveActiveSubscriptionResourceModel struct {
	ID                                    types.String                     `tfsdk:"id"`
	Name                                  types.String                     `tfsdk:"name"`
	PaymentMethod                         types.String                     `tfsdk:"payment_method"`
	PaymentMethodID                       types.String                     `tfsdk:"payment_method_id"`
	CloudProvider                         types.String                     `tfsdk:"cloud_provider"`
	AwsAccountID                          types.String                     `tfsdk:"aws_account_id"`
	CreationPlan                          CreationPlanListValue            `tfsdk:"creation_plan"`
	RedisVersion                          types.String                     `tfsdk:"redis_version"`
	MaintenanceWindows                    customtypes.MaintenanceListValue `tfsdk:"maintenance_windows"`
	Pricing                               types.List                       `tfsdk:"pricing"`
	CustomerManagedKeyEnabled             types.Bool                       `tfsdk:"customer_managed_key_enabled"`
	CustomerManagedKeyDeletionGracePeriod types.String                     `tfsdk:"customer_managed_key_deletion_grace_period"`
	CustomerManagedKey                    CustomerManagedKeyListValue      `tfsdk:"customer_managed_key"`
	CustomerManagedKeyRedisServiceAccount types.String                     `tfsdk:"customer_managed_key_redis_service_account"`
	CustomerManagedKeyAwsRoleArn          types.String                     `tfsdk:"customer_managed_key_aws_role_arn"`
	PublicEndpointAccess                  types.Bool                       `tfsdk:"public_endpoint_access"`
	ResourceTags                          types.Map                        `tfsdk:"resource_tags"`
	Timeouts                              timeouts.Value                   `tfsdk:"timeouts"`
}
