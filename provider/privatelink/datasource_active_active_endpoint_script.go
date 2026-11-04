package privatelink

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

var (
	_ datasource.DataSource              = &activeActiveEndpointScriptDataSource{}
	_ datasource.DataSourceWithConfigure = &activeActiveEndpointScriptDataSource{}
)

type activeActiveEndpointScriptDataSource struct {
	client *client.ApiClient
}

// NewActiveActiveEndpointScriptDataSource returns the Active-Active PrivateLink endpoint script data source.
func NewActiveActiveEndpointScriptDataSource() datasource.DataSource {
	return &activeActiveEndpointScriptDataSource{}
}

func (d *activeActiveEndpointScriptDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_active_active_private_link_endpoint_script"
}

func (d *activeActiveEndpointScriptDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Active Active PrivateLink Endpoint Script data source allows users to request an endpoint script for an Active Active Subscription PrivateLink",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the Active Active subscription",
				Computed:    true,
			},
			"subscription_id": schema.StringAttribute{
				Description: "The ID of an Active Active subscription",
				Required:    true,
			},
			"region_id": schema.Int64Attribute{
				Description: "The ID of an Active Active subscription region",
				Required:    true,
			},
			"endpoint_script": schema.StringAttribute{
				Description: "The endpoint script for the PrivateLink",
				Computed:    true,
			},
		},
	}
}

func (d *activeActiveEndpointScriptDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
