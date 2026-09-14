package pro

import (
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
)

// ProSubscriptionDataSourceModel describes the data source data model.
type ProSubscriptionDataSourceModel struct {
	ID                                    types.String                     `tfsdk:"id"`
	Name                                  types.String                     `tfsdk:"name"`
	PaymentMethod                         types.String                     `tfsdk:"payment_method"`
	PaymentMethodID                       types.String                     `tfsdk:"payment_method_id"`
	MemoryStorage                         types.String                     `tfsdk:"memory_storage"`
	NumberOfDatabases                     types.Int64                      `tfsdk:"number_of_databases"`
	Status                                types.String                     `tfsdk:"status"`
	CustomerManagedKeyEnabled             types.Bool                       `tfsdk:"customer_managed_key_enabled"`
	CustomerManagedKeyDeletionGracePeriod types.String                     `tfsdk:"customer_managed_key_deletion_grace_period"`
	CustomerManagedKeyRedisServiceAccount types.String                     `tfsdk:"customer_managed_key_redis_service_account"`
	CustomerManagedKeyAwsRoleArn          types.String                     `tfsdk:"customer_managed_key_aws_role_arn"`
	PublicEndpointAccess                  types.Bool                       `tfsdk:"public_endpoint_access"`
	PrometheusEndpoint                    types.String                     `tfsdk:"prometheus_endpoint"`
	CloudProvider                         types.List                       `tfsdk:"cloud_provider"`
	MaintenanceWindows                    customtypes.MaintenanceListValue `tfsdk:"maintenance_windows"`
	Pricing                               types.List                       `tfsdk:"pricing"`
}
