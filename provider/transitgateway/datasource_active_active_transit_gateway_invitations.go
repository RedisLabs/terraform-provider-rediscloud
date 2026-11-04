package transitgateway

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

var (
	_ datasource.DataSource              = &activeActiveTransitGatewayInvitationsDataSource{}
	_ datasource.DataSourceWithConfigure = &activeActiveTransitGatewayInvitationsDataSource{}
)

type activeActiveTransitGatewayInvitationsDataSource struct {
	client *client.ApiClient
}

// NewActiveActiveTransitGatewayInvitationsDataSource provides the Framework implementation for rediscloud_active_active_transit_gateway_invitations.
func NewActiveActiveTransitGatewayInvitationsDataSource() datasource.DataSource {
	return &activeActiveTransitGatewayInvitationsDataSource{}
}

func (d *activeActiveTransitGatewayInvitationsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_active_active_transit_gateway_invitations"
}

func (d *activeActiveTransitGatewayInvitationsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *activeActiveTransitGatewayInvitationsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists AWS Transit Gateway resource share invitations for a specific region in an Active-Active subscription. Invitations are created when you share an AWS Transit Gateway with Redis Cloud via AWS Resource Manager.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The identifier of the data source, in the form <subscription_id>/<region_id>",
				Computed:    true,
			},
			"subscription_id": schema.StringAttribute{
				Description: "The ID of the Active-Active subscription",
				Required:    true,
			},
			"region_id": schema.Int64Attribute{
				Description: "The region ID",
				Required:    true,
			},
			"wait_for_invitations_timeout": schema.Int64Attribute{
				Description: "When set, retry fetching invitations until at least one is found or the specified timeout (in seconds) is reached. " +
					"Useful when creating AWS RAM share and Redis Cloud resources in the same Terraform run.",
				Optional: true,
			},
			"invitations": schema.ListAttribute{
				Description: "List of Transit Gateway invitations",
				Computed:    true,
				ElementType: TransitGatewayInvitationElemType,
			},
		},
	}
}
