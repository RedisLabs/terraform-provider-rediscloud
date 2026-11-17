package transitgateway

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

var (
	_ datasource.DataSource              = &transitGatewayDataSource{}
	_ datasource.DataSourceWithConfigure = &transitGatewayDataSource{}
)

type transitGatewayDataSource struct {
	client *client.ApiClient
}

// NewTransitGatewayDataSource provides the Framework implementation for rediscloud_transit_gateway.
func NewTransitGatewayDataSource() datasource.DataSource {
	return &transitGatewayDataSource{}
}

func (d *transitGatewayDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_transit_gateway"
}

func (d *transitGatewayDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *transitGatewayDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Transit Gateway data source allows access to an available Transit Gateway within your Redis Enterprise Cloud Account.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The identifier of the data source, in the form <subscription_id>/<tgw_id>",
				Computed:    true,
			},
			"subscription_id": schema.StringAttribute{
				Description: "The id of a Pro/Flexible subscription",
				Required:    true,
			},
			"tgw_id": schema.Int64Attribute{
				Description: "The id of the Transit Gateway relative to the associated subscription",
				Optional:    true,
				Computed:    true,
			},
			"aws_tgw_uid": schema.StringAttribute{
				Description: "The id of the Transit Gateway as known to AWS",
				Optional:    true,
				Computed:    true,
			},
			"wait_for_tgw_timeout": schema.Int64Attribute{
				Description: "When set, retry fetching until a Transit Gateway matching the filters is found or the specified timeout (in seconds) is reached. " +
					"Useful when accepting a TGW invitation and querying the TGW in the same Terraform run.",
				Optional: true,
			},
			"attachment_uid": schema.StringAttribute{
				Description: "A unique identifier for the Subscription/Transit Gateway attachment, if any",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "The status of the Transit Gateway",
				Computed:    true,
			},
			"attachment_status": schema.StringAttribute{
				Description: "The status of the Subscription/Transit Gateway attachment, if any",
				Computed:    true,
			},
			"aws_account_id": schema.StringAttribute{
				Description: "The Transit Gateway's AWS account id",
				Computed:    true,
			},
			"cidrs": schema.ListAttribute{
				Description: "A list of consumer Cidr blocks, if an attachment exists",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}
