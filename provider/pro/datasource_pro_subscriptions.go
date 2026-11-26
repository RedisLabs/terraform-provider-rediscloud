package pro

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
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

// Schema defines the schema for the data source. Each subscriptions element omits
// maintenance_windows and pricing because populating them would require an extra API
// call per subscription. Everything exposed here comes from Subscription.List.
func (d *proSubscriptionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	subscriptionType := types.ObjectType{AttrTypes: customtypes.AttrTypesOf(ProSubscriptionModel{
		CloudProvider: types.ListNull(utils.CloudProviderElemType),
	})}

	resp.Schema = schema.Schema{
		Description: "The Pro Subscriptions data source allows access to a list of all pro subscriptions within your Redis Enterprise Cloud account.",
		Attributes: map[string]schema.Attribute{
			// A computed list attribute remains unknown when Terraform defers the read.
			"subscriptions": schema.ListAttribute{
				Description: "A list of the pro subscriptions in the account",
				Computed:    true,
				ElementType: subscriptionType,
			},
		},
		Blocks: map[string]schema.Block{
			"filters": schema.SingleNestedBlock{
				Description: "Optional filters that narrow the returned subscriptions. Omit to return every pro subscription.",
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Description: "If specified, only subscriptions with this exact name are returned.",
						Optional:    true,
					},
				},
			},
		},
	}
}
