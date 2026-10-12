package essentialssubscription

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &essentialsSubscriptionDataSource{}
	_ datasource.DataSourceWithConfigure = &essentialsSubscriptionDataSource{}
)

// essentialsSubscriptionDataSource is the data source implementation.
type essentialsSubscriptionDataSource struct {
	client *client.ApiClient
}

// NewEssentialsSubscriptionDataSource creates a new data source instance.
func NewEssentialsSubscriptionDataSource() datasource.DataSource {
	return &essentialsSubscriptionDataSource{}
}

// Metadata returns the data source type name.
func (d *essentialsSubscriptionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_essentials_subscription"
}

// Configure adds the provider configured client to the data source.
func (d *essentialsSubscriptionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *essentialsSubscriptionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Watches an Essentials Subscription within your Redis Enterprise Cloud Account.",
		Attributes: map[string]schema.Attribute{
			//TODO(TF3.0) refactor the id to a StringAttribute
			"id": schema.Int64Attribute{
				Description: "The subscription's id",
				Computed:    true,
				Optional:    true,
			},
			"name": schema.StringAttribute{
				Description: "A meaningful name to identify the subscription",
				Computed:    true,
				Optional:    true,
			},
			"status": schema.StringAttribute{
				Description: "The status of this subscription",
				Computed:    true,
			},
			"plan_id": schema.Int64Attribute{
				Description: "The identifier of the plan to template the subscription",
				Computed:    true,
			},
			"payment_method_id": schema.Int64Attribute{
				Description: "The identifier of the method which will be charged for this subscription. Not required for free plans",
				Computed:    true,
			},
			"creation_date": schema.StringAttribute{
				Description: "The date/time this subscription was created",
				Computed:    true,
			},
		},
	}
}
