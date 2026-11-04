package privatelink

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

var (
	_ datasource.DataSource              = &privateLinkDataSource{}
	_ datasource.DataSourceWithConfigure = &privateLinkDataSource{}
)

type privateLinkDataSource struct {
	client *client.ApiClient
}

// NewPrivateLinkDataSource returns the Pro subscription PrivateLink data source.
func NewPrivateLinkDataSource() datasource.DataSource {
	return &privateLinkDataSource{}
}

func (d *privateLinkDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_private_link"
}

func (d *privateLinkDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Private Link data source allows access to an available Pro Subscription Private Link within your Redis Enterprise Cloud Account.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the Pro subscription",
				Computed:    true,
			},
			"subscription_id": schema.StringAttribute{
				Description: "The ID of a Pro subscription",
				Required:    true,
			},
			"share_name": schema.StringAttribute{
				Description: "ARN of the share to attach to this PrivateLink",
				Computed:    true,
			},
			"principals": schema.SetAttribute{
				Description: "List of principals attached to this PrivateLink",
				Computed:    true,
				CustomType:  NewPrincipalSetType(),
				ElementType: principalElemType,
			},
			"resource_configuration_id": schema.StringAttribute{
				Description: "ID of the resource configuration to attach to this PrivateLink",
				Computed:    true,
			},
			"resource_configuration_arn": schema.StringAttribute{
				Description: "ARN of the resource configuration attached to this PrivateLink",
				Computed:    true,
			},
			"share_arn": schema.StringAttribute{
				Description: "ARN of the share to attach to this Private Link",
				Computed:    true,
			},
			"connections": schema.SetAttribute{
				Description: "Connections attached to this PrivateLink",
				Computed:    true,
				ElementType: connectionElemType,
			},
			"databases": schema.SetAttribute{
				Description: "The databases attached to this PrivateLink",
				Computed:    true,
				ElementType: databaseElemType,
			},
		},
	}
}

func (d *privateLinkDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
