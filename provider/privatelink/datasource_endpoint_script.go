package privatelink

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

var (
	_ datasource.DataSource              = &endpointScriptDataSource{}
	_ datasource.DataSourceWithConfigure = &endpointScriptDataSource{}
)

type endpointScriptDataSource struct {
	client *client.ApiClient
}

// NewEndpointScriptDataSource returns the Pro subscription PrivateLink endpoint script data source.
func NewEndpointScriptDataSource() datasource.DataSource {
	return &endpointScriptDataSource{}
}

func (d *endpointScriptDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_private_link_endpoint_script"
}

func (d *endpointScriptDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The PrivateLink Endpoint Script data source allows users to request an endpoint script for a pro subscription PrivateLink",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the Pro subscription",
				Computed:    true,
			},
			"subscription_id": schema.StringAttribute{
				Description: "The ID of a Pro subscription",
				Required:    true,
			},
			"endpoint_script": schema.StringAttribute{
				Description: "The endpoint script for the PrivateLink",
				Computed:    true,
			},
		},
	}
}

func (d *endpointScriptDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
