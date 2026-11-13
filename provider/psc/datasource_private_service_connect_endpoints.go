package psc

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

var (
	_ datasource.DataSource              = &privateServiceConnectEndpointsDataSource{}
	_ datasource.DataSourceWithConfigure = &privateServiceConnectEndpointsDataSource{}
)

type privateServiceConnectEndpointsDataSource struct {
	client *client.ApiClient
}

// NewPrivateServiceConnectEndpointsDataSource returns the rediscloud_private_service_connect_endpoints data source.
func NewPrivateServiceConnectEndpointsDataSource() datasource.DataSource {
	return &privateServiceConnectEndpointsDataSource{}
}

var endpointElemType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"private_service_connect_endpoint_id": types.Int64Type,
	"gcp_project_id":                      types.StringType,
	"gcp_vpc_name":                        types.StringType,
	"gcp_vpc_subnet_name":                 types.StringType,
	"endpoint_connection_name":            types.StringType,
	"status":                              types.StringType,
	"service_attachments":                 types.ListType{ElemType: serviceAttachmentElemType},
}}

func (d *privateServiceConnectEndpointsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_private_service_connect_endpoints"
}

func (d *privateServiceConnectEndpointsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Private Service Connect Endpoints data source allows access to an available endpoints on a Private Service Connect Service within your Redis Enterprise Cloud Account.",
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
				Required:    true,
			},
			"endpoints": schema.ListAttribute{
				Computed:    true,
				ElementType: endpointElemType,
			},
		},
	}
}

func (d *privateServiceConnectEndpointsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
