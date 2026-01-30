package paymentmethod

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &paymentMethodDataSource{}
	_ datasource.DataSourceWithConfigure = &paymentMethodDataSource{}
)

// paymentMethodDataSource is the data source implementation.
type paymentMethodDataSource struct {
	client *client.ApiClient
}

// NewDataSource returns a new data source instance.
func NewDataSource() datasource.DataSource {
	return &paymentMethodDataSource{}
}

// Metadata returns the data source type name.
func (d *paymentMethodDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_payment_method"
}

// Configure adds the provider configured client to the data source.
func (d *paymentMethodDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *paymentMethodDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Payment Method data source allows access to the ID of a Payment Method configured against your Redis Enterprise Cloud account. This ID can be used when creating Subscription resources.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the payment method",
				Computed:    true,
			},
			"card_type": schema.StringAttribute{
				Description: "Type of card that the payment method should be, such as `Visa`",
				Optional:    true,
				Computed:    true,
			},
			"exclude_expired": schema.BoolAttribute{
				Description: "Whether to exclude any expired cards or not. Default is `true`.",
				Optional:    true,
			},
			"last_four_numbers": schema.StringAttribute{
				Description: "Last four numbers of the card of the payment method",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^\d{4}$`),
						"must contain last four numbers of the card of the payment method",
					),
				},
			},
		},
	}
}
