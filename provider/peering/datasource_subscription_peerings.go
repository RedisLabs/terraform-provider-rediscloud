package peering

import (
	"context"
	"fmt"
	"regexp"

	"github.com/RedisLabs/rediscloud-go-api/service/subscriptions"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

var (
	_ datasource.DataSource              = &subscriptionPeeringsDataSource{}
	_ datasource.DataSourceWithConfigure = &subscriptionPeeringsDataSource{}
)

type subscriptionPeeringsDataSource struct {
	client *client.ApiClient
}

// NewSubscriptionPeeringsDataSource returns a new data source instance.
func NewSubscriptionPeeringsDataSource() datasource.DataSource {
	return &subscriptionPeeringsDataSource{}
}

func (d *subscriptionPeeringsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_subscription_peerings"
}

func (d *subscriptionPeeringsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *subscriptionPeeringsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Subscription Peerings data source allows access to a list of VPC peerings configured on the subscription.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier for this data source: the `status` filter, or `ALL` when no filter is set",
				Optional:    true,
				Computed:    true,
			},
			"subscription_id": schema.StringAttribute{
				Description: "A valid subscription predefined in the current account",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^\d+$`), "must be a number"),
				},
			},
			"status": schema.StringAttribute{
				Description: "Current status of the account - `initiating-request`, `pending-acceptance`, `active`, `inactive` or `failed`",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						subscriptions.VPCPeeringStatusInitiatingRequest,
						subscriptions.VPCPeeringStatusActive,
						subscriptions.VPCPeeringStatusInactive,
						subscriptions.VPCPeeringStatusPendingAcceptance,
						subscriptions.VPCPeeringStatusFailed,
					),
				},
			},
			// SDKv2 served this computed-only TypeSet of resources as an attribute, not a block.
			"peerings": schema.SetAttribute{
				Description: "A list of VPC peerings from either a single or multiple cloud providers",
				Computed:    true,
				ElementType: peeringElemType,
			},
		},
	}
}
