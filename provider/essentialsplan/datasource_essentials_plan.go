package essentialsplan

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &essentialsPlanDataSource{}
	_ datasource.DataSourceWithConfigure = &essentialsPlanDataSource{}
)

// essentialsPlanDataSource is the data source implementation.
type essentialsPlanDataSource struct {
	client *client.ApiClient
}

// NewEssentialsPlanDataSource creates a new data source instance.
func NewEssentialsPlanDataSource() datasource.DataSource {
	return &essentialsPlanDataSource{}
}

// Metadata returns the data source type name.
func (d *essentialsPlanDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_essentials_plan"
}

// Configure adds the provider configured client to the data source.
func (d *essentialsPlanDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.ApiClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.ApiClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

// Schema defines the schema for the data source.
func (d *essentialsPlanDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Essentials Plan data source allows access to the templates for Essentials Subscriptions.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The plan's unique identifier",
				Computed:    true,
				Optional:    true,
			},
			"name": schema.StringAttribute{
				Description: "A convenient name for the plan. Not guaranteed to be unique, especially across provider/region",
				Computed:    true,
				Optional:    true,
			},
			"size": schema.Float64Attribute{
				Description: "The capacity of databases created in this plan",
				Computed:    true,
				Optional:    true,
			},
			"size_measurement_unit": schema.StringAttribute{
				Description: "The units of 'size', usually 'MB' or 'GB'",
				Computed:    true,
				Optional:    true,
			},
			"subscription_id": schema.Int64Attribute{
				Description: "Filter plans by what is available for a given subscription",
				Optional:    true,
			},
			"cloud_provider": schema.StringAttribute{
				Description: "The cloud provider: 'AWS', 'GCP' or 'Azure'",
				Computed:    true,
				Optional:    true,
			},
			"region": schema.StringAttribute{
				Description: "The region to place databases in, format and availability dependent on cloud_provider",
				Computed:    true,
				Optional:    true,
			},
			"region_id": schema.Int64Attribute{
				Description: "An internal, unique-across-cloud-providers id for database region",
				Computed:    true,
			},
			"price": schema.Int64Attribute{
				Description: "The plan's cost",
				Computed:    true,
			},
			"price_currency": schema.StringAttribute{
				Description: "The currency in which the plan's price is denominated, e.g. 'USD'",
				Computed:    true,
			},
			"price_period": schema.StringAttribute{
				Description: "The billing period that the price applies to, usually 'Month'",
				Computed:    true,
			},
			"maximum_databases": schema.Int64Attribute{
				Description: "The maximum number of databases that can be created under this plan",
				Computed:    true,
			},
			"maximum_throughput": schema.Int64Attribute{
				Description: "The maximum throughput supported by databases in this plan",
				Computed:    true,
			},
			"maximum_bandwidth_in_gb": schema.Int64Attribute{
				Description: "The maximum network bandwidth, in gigabytes (GB), supported by databases in this plan",
				Computed:    true,
			},
			"availability": schema.StringAttribute{
				Description: "'No replication', 'Single-zone' or 'Multi-zone'",
				Computed:    true,
				Optional:    true,
			},
			"connections": schema.StringAttribute{
				Description: "The maximum number of concurrent client connections supported by databases in this plan",
				Computed:    true,
			},
			"cidr_allow_rules": schema.Int64Attribute{
				Description: "The maximum number of CIDR allow-list rules that can be configured for databases in this plan",
				Computed:    true,
			},
			"support_data_persistence": schema.BoolAttribute{
				Description: "Whether databases in this plan support data persistence",
				Computed:    true,
				Optional:    true,
			},
			"support_instant_and_daily_backups": schema.BoolAttribute{
				Description: "Whether databases in this plan support instant and daily backups",
				Computed:    true,
			},
			"support_replication": schema.BoolAttribute{
				Description: "Whether databases in this plan support replication",
				Computed:    true,
				Optional:    true,
			},
			"support_clustering": schema.BoolAttribute{
				Description: "Whether databases in this plan support clustering",
				Computed:    true,
			},
			"supported_alerts": schema.ListAttribute{
				Description: "List of the type of alerts supported by databases in this plan",
				Computed:    true,
				ElementType: types.StringType,
			},
			"customer_support": schema.StringAttribute{
				Description: "Level of customer support available e.g. 'Basic', 'Standard'",
				Computed:    true,
			},
		},
	}
}
