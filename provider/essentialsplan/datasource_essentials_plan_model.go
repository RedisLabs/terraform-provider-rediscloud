package essentialsplan

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// EssentialsPlanModel describes the data source data model.
type EssentialsPlanModel struct {
	ID                            types.Int64   `tfsdk:"id"`
	Name                          types.String  `tfsdk:"name"`
	Size                          types.Float64 `tfsdk:"size"`
	SizeMeasurementUnit           types.String  `tfsdk:"size_measurement_unit"`
	SubscriptionID                types.Int64   `tfsdk:"subscription_id"`
	CloudProvider                 types.String  `tfsdk:"cloud_provider"`
	Region                        types.String  `tfsdk:"region"`
	RegionID                      types.Int64   `tfsdk:"region_id"`
	Price                         types.Int64   `tfsdk:"price"`
	PriceCurrency                 types.String  `tfsdk:"price_currency"`
	PricePeriod                   types.String  `tfsdk:"price_period"`
	MaximumDatabases              types.Int64   `tfsdk:"maximum_databases"`
	MaximumThroughput             types.Int64   `tfsdk:"maximum_throughput"`
	MaximumBandwidthInGB          types.Int64   `tfsdk:"maximum_bandwidth_in_gb"`
	Availability                  types.String  `tfsdk:"availability"`
	Connections                   types.String  `tfsdk:"connections"`
	CidrAllowRules                types.Int64   `tfsdk:"cidr_allow_rules"`
	SupportDataPersistence        types.Bool    `tfsdk:"support_data_persistence"`
	SupportInstantAndDailyBackups types.Bool    `tfsdk:"support_instant_and_daily_backups"`
	SupportReplication            types.Bool    `tfsdk:"support_replication"`
	SupportClustering             types.Bool    `tfsdk:"support_clustering"`
	SupportedAlerts               types.List    `tfsdk:"supported_alerts"`
	CustomerSupport               types.String  `tfsdk:"customer_support"`
}
