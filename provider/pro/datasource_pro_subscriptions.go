package pro

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &proSubscriptionsDataSource{}
	_ datasource.DataSourceWithConfigure = &proSubscriptionsDataSource{}
)

// proSubscriptionsDataSource is the data source implementation. It lists every pro
// subscription in the account, unlike the singular proSubscriptionDataSource which
// resolves to exactly one.
type proSubscriptionsDataSource struct {
	client *client.ApiClient
}

// NewProSubscriptionsDataSource creates a new data source instance.
func NewProSubscriptionsDataSource() datasource.DataSource {
	return &proSubscriptionsDataSource{}
}

// Metadata returns the data source type name. The public name is
// "rediscloud_subscriptions" (all pro/flexible subscriptions).
func (d *proSubscriptionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_subscriptions"
}

// Configure adds the provider configured client to the data source.
func (d *proSubscriptionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.ApiClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.ApiClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = c
}

// Schema defines the schema for the data source. Each subscriptions element reuses the
// singular data source's shared attributes and cloud_provider block, but only those —
// maintenance_windows and pricing are omitted because populating them would require an
// extra API call per subscription. Everything exposed here comes from Subscription.List.
func (d *proSubscriptionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Pro Subscriptions data source allows access to a list of all pro subscriptions within your Redis Enterprise Cloud account.",
		// The provider's protocol-v5 mux cannot serve nested attributes, so the
		// object-shaped filters and subscriptions schemas must use blocks.
		Blocks: map[string]schema.Block{
			// filters groups the (optional) query filters so their purpose is explicit and
			// new filters can be added alongside name without a schema-breaking change.
			"filters": schema.SingleNestedBlock{
				Description: "Optional filters that narrow the returned subscriptions. Omit to return every pro subscription.",
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Description: "If specified, only subscriptions with this exact name are returned.",
						Optional:    true,
					},
				},
			},
			"subscriptions": schema.ListNestedBlock{
				Description: "A list of the pro subscriptions in the account",
				NestedObject: schema.NestedBlockObject{
					Attributes: subscriptionDataSourceAttributes(),
					Blocks: map[string]schema.Block{
						"cloud_provider": cloudProviderBlock(),
					},
				},
			},
		},
	}
}
