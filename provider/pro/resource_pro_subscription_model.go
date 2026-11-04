package pro

import (
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

// ProSubscriptionResourceModel describes the rediscloud_subscription resource data model.
type ProSubscriptionResourceModel struct {
	ID                                    types.String                     `tfsdk:"id"`
	Name                                  types.String                     `tfsdk:"name"`
	PaymentMethod                         types.String                     `tfsdk:"payment_method"`
	PaymentMethodID                       types.String                     `tfsdk:"payment_method_id"`
	MemoryStorage                         types.String                     `tfsdk:"memory_storage"`
	Allowlist                             types.List                       `tfsdk:"allowlist"`
	CloudProvider                         types.List                       `tfsdk:"cloud_provider"`
	CreationPlan                          types.List                       `tfsdk:"creation_plan"`
	RedisVersion                          types.String                     `tfsdk:"redis_version"`
	MaintenanceWindows                    customtypes.MaintenanceListValue `tfsdk:"maintenance_windows"`
	Pricing                               types.List                       `tfsdk:"pricing"`
	CustomerManagedKeyEnabled             types.Bool                       `tfsdk:"customer_managed_key_enabled"`
	CustomerManagedKeyDeletionGracePeriod types.String                     `tfsdk:"customer_managed_key_deletion_grace_period"`
	CustomerManagedKey                    types.List                       `tfsdk:"customer_managed_key"`
	CustomerManagedKeyRedisServiceAccount types.String                     `tfsdk:"customer_managed_key_redis_service_account"`
	CustomerManagedKeyAwsRoleArn          types.String                     `tfsdk:"customer_managed_key_aws_role_arn"`
	PublicEndpointAccess                  types.Bool                       `tfsdk:"public_endpoint_access"`
	PrometheusEndpoint                    types.String                     `tfsdk:"prometheus_endpoint"`
	Timeouts                              timeouts.Value                   `tfsdk:"timeouts"`
}

// SubscriptionAllowlistModel describes the allowlist block.
type SubscriptionAllowlistModel struct {
	Cidrs            types.Set `tfsdk:"cidrs"`
	SecurityGroupIDs types.Set `tfsdk:"security_group_ids"`
}

// SubscriptionRegionModel describes a cloud_provider.region block.
type SubscriptionRegionModel struct {
	Region                     types.String `tfsdk:"region"`
	MultipleAvailabilityZones  types.Bool   `tfsdk:"multiple_availability_zones"`
	PreferredAvailabilityZones types.List   `tfsdk:"preferred_availability_zones"`
	NetworkingDeploymentCIDR   types.String `tfsdk:"networking_deployment_cidr"`
	NetworkingVPCID            types.String `tfsdk:"networking_vpc_id"`
	Networks                   types.List   `tfsdk:"networks"`
}

// SubscriptionCloudProviderModel describes the cloud_provider block.
type SubscriptionCloudProviderModel struct {
	Provider       types.String `tfsdk:"provider"`
	CloudAccountID types.String `tfsdk:"cloud_account_id"`
	AWSAccountID   types.String `tfsdk:"aws_account_id"`
	Region         types.Set    `tfsdk:"region"`
	ResourceTags   types.Map    `tfsdk:"resource_tags"`
}

// CreationPlanModel describes the creation_plan block.
type CreationPlanModel struct {
	MemoryLimitInGB            types.Float64 `tfsdk:"memory_limit_in_gb"`
	DatasetSizeInGB            types.Float64 `tfsdk:"dataset_size_in_gb"`
	QueryPerformanceFactor     types.String  `tfsdk:"query_performance_factor"`
	ThroughputMeasurementBy    types.String  `tfsdk:"throughput_measurement_by"`
	ThroughputMeasurementValue types.Int64   `tfsdk:"throughput_measurement_value"`
	AverageItemSizeInBytes     types.Int64   `tfsdk:"average_item_size_in_bytes"`
	RamPercentage              types.Int64   `tfsdk:"ram_percentage"`
	Quantity                   types.Int64   `tfsdk:"quantity"`
	SupportOSSClusterAPI       types.Bool    `tfsdk:"support_oss_cluster_api"`
	Replication                types.Bool    `tfsdk:"replication"`
	Modules                    types.List    `tfsdk:"modules"`
}

// CustomerManagedKeyModel describes a customer_managed_key block.
type CustomerManagedKeyModel struct {
	ResourceName types.String `tfsdk:"resource_name"`
}

var subscriptionAllowlistElemType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(SubscriptionAllowlistModel{
	Cidrs:            types.SetNull(types.StringType),
	SecurityGroupIDs: types.SetNull(types.StringType),
})}

var subscriptionNetworkElemType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(utils.CloudNetworkModel{})}

var subscriptionRegionElemType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(SubscriptionRegionModel{
	PreferredAvailabilityZones: types.ListNull(types.StringType),
	Networks:                   types.ListNull(subscriptionNetworkElemType),
})}

var subscriptionCloudProviderElemType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(SubscriptionCloudProviderModel{
	Region:       types.SetNull(subscriptionRegionElemType),
	ResourceTags: types.MapNull(types.StringType),
})}

var creationPlanElemType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(CreationPlanModel{
	Modules: types.ListNull(types.StringType),
})}

var customerManagedKeyElemType = types.ObjectType{AttrTypes: customtypes.AttrTypesOf(CustomerManagedKeyModel{})}
