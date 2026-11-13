package psc

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

var (
	_ datasource.DataSource              = &privateServiceConnectDataSource{}
	_ datasource.DataSourceWithConfigure = &privateServiceConnectDataSource{}
)

type privateServiceConnectDataSource struct {
	client *client.ApiClient
}

// NewPrivateServiceConnectDataSource returns the rediscloud_private_service_connect data source.
func NewPrivateServiceConnectDataSource() datasource.DataSource {
	return &privateServiceConnectDataSource{}
}

func (d *privateServiceConnectDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_private_service_connect"
}

func (d *privateServiceConnectDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Private Service Connect data source allows access to an available Private Service Connect Service within your Redis Enterprise Cloud Account.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"subscription_id": schema.StringAttribute{
				Description: "The ID of a Pro subscription",
				Required:    true,
			},
			//TODO(TF3.0) consider exposing private_service_connect_service_id as a string
			"private_service_connect_service_id": schema.Int64Attribute{
				Description: "The ID of the Private Service Connect Service relative to the associated subscription",
				Computed:    true,
			},
			"connection_host_name": schema.StringAttribute{
				Description: "The connection host name",
				Computed:    true,
			},
			"service_attachment_name": schema.StringAttribute{
				Description: "The service attachment name",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "The Private Service Connect status",
				Computed:    true,
			},
		},
	}
}

func (d *privateServiceConnectDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
