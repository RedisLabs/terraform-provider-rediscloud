package activeactive

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ActiveActiveSubscriptionDataSourceModel describes the data source data model.
type ActiveActiveSubscriptionDataSourceModel struct {
	ID                                    types.String `tfsdk:"id"`
	Name                                  types.String `tfsdk:"name"`
	PaymentMethod                         types.String `tfsdk:"payment_method"`
	PaymentMethodID                       types.String `tfsdk:"payment_method_id"`
	NumberOfDatabases                     types.Int64  `tfsdk:"number_of_databases"`
	Status                                types.String `tfsdk:"status"`
	CustomerManagedKeyEnabled             types.Bool   `tfsdk:"customer_managed_key_enabled"`
	CustomerManagedKeyDeletionGracePeriod types.String `tfsdk:"customer_managed_key_deletion_grace_period"`
	CustomerManagedKeyRedisServiceAccount types.String `tfsdk:"customer_managed_key_redis_service_account"`
	CustomerManagedKeyAwsRoleArn          types.String `tfsdk:"customer_managed_key_aws_role_arn"`
	PublicEndpointAccess                  types.Bool   `tfsdk:"public_endpoint_access"`
	CloudProvider                         types.String `tfsdk:"cloud_provider"`
	AwsAccountID                          types.String `tfsdk:"aws_account_id"`
	MaintenanceWindows                    types.List   `tfsdk:"maintenance_windows"`
	Pricing                               types.List   `tfsdk:"pricing"`
	ResourceTags                          types.Map    `tfsdk:"resource_tags"`
}
