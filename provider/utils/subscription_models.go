package utils

import (
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
)

// PricingModel should be used to encode or decode subscription pricing entries as Terraform values.
type PricingModel struct {
	DatabaseName        types.String  `tfsdk:"database_name"`
	Type                types.String  `tfsdk:"type"`
	TypeDetails         types.String  `tfsdk:"type_details"`
	Quantity            types.Int64   `tfsdk:"quantity"`
	QuantityMeasurement types.String  `tfsdk:"quantity_measurement"`
	PricePerUnit        types.Float64 `tfsdk:"price_per_unit"`
	PriceCurrency       types.String  `tfsdk:"price_currency"`
	PricePeriod         types.String  `tfsdk:"price_period"`
	Region              types.String  `tfsdk:"region"`
}

var pricingAttrTypes = customtypes.AttrTypesOf(PricingModel{})

// CloudNetworkModel should be used to encode or decode subscription networks as Terraform values.
type CloudNetworkModel struct {
	NetworkingSubnetID       types.String `tfsdk:"networking_subnet_id"`
	NetworkingDeploymentCIDR types.String `tfsdk:"networking_deployment_cidr"`
	NetworkingVPCID          types.String `tfsdk:"networking_vpc_id"`
}

var cloudNetworkAttrTypes = customtypes.AttrTypesOf(CloudNetworkModel{})

// CloudRegionModel should be used to encode or decode subscription cloud regions as Terraform values.
type CloudRegionModel struct {
	Region                     types.String `tfsdk:"region"`
	MultipleAvailabilityZones  types.Bool   `tfsdk:"multiple_availability_zones"`
	PreferredAvailabilityZones types.List   `tfsdk:"preferred_availability_zones"`
	NetworkingVPCID            types.String `tfsdk:"networking_vpc_id"`
	Networks                   types.List   `tfsdk:"networks"`
}

// cloudRegionAttrTypes supplies element types for collections whose zero values do not
// contain enough type information.
var cloudRegionAttrTypes = customtypes.AttrTypesOf(CloudRegionModel{
	PreferredAvailabilityZones: types.ListNull(types.StringType),
	Networks:                   types.ListNull(types.ObjectType{AttrTypes: cloudNetworkAttrTypes}),
})

// CloudProviderModel should be used to encode or decode subscription cloud providers as Terraform values.
type CloudProviderModel struct {
	Provider       types.String `tfsdk:"provider"`
	CloudAccountID types.String `tfsdk:"cloud_account_id"`
	AWSAccountID   types.String `tfsdk:"aws_account_id"`
	ResourceTags   types.Map    `tfsdk:"resource_tags"`
	Region         types.Set    `tfsdk:"region"`
}

// cloudProviderAttrTypes supplies element types for collections whose zero values do not
// contain enough type information.
var cloudProviderAttrTypes = customtypes.AttrTypesOf(CloudProviderModel{
	ResourceTags: types.MapNull(types.StringType),
	Region:       types.SetNull(types.ObjectType{AttrTypes: cloudRegionAttrTypes}),
})
