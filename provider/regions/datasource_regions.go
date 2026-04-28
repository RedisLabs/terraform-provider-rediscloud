package regions

import (
	"context"
	"fmt"

	"github.com/RedisLabs/rediscloud-go-api/service/cloud_accounts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &regionsDataSource{}
	_ datasource.DataSourceWithConfigure = &regionsDataSource{}
)

// regionsDataSource is the data source implementation.
type regionsDataSource struct {
	client *client.ApiClient
}

// NewRegionsDataSource returns a new data source instance.
func NewRegionsDataSource() datasource.DataSource {
	return &regionsDataSource{}
}

// Metadata returns the data source type name.
func (d *regionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_regions"
}

// Configure adds the provider configured client to the data source.
func (d *regionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *regionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Regions data source allows access to a list of supported cloud provider regions. These regions can be used with the subscription resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier for this data source",
				Computed:    true,
			},
			"provider_name": schema.StringAttribute{
				Description: "The name of the cloud provider to filter returned regions, (accepted values are `AWS` or `GCP`).",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(cloud_accounts.ProviderValues()...),
				},
			},
		},
		// regions is a SetNestedBlock (not SetNestedAttribute) because the SDKv2
		// schema used TypeSet with Elem: &schema.Resource{}, which represents a
		// block in protocol v5. Using SetNestedAttribute would cause a runtime panic
		// in the muxed provider.
		Blocks: map[string]schema.Block{
			"regions": schema.SetNestedBlock{
				Description: "A list of regions from either a single or multiple cloud providers",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"region_id": schema.Int64Attribute{
							Description: "The unique identifier of the region",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The identifier assigned by the cloud provider, (for example `eu-west-1` for `AWS`)",
							Computed:    true,
						},
						"provider_name": schema.StringAttribute{
							Description: "The identifier of the owning cloud provider, (either `AWS` or `GCP`)",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}
