package datapersistence

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &dataPersistenceDataSource{}
	_ datasource.DataSourceWithConfigure = &dataPersistenceDataSource{}
)

// dataPersistenceDataSource is the data source implementation.
type dataPersistenceDataSource struct {
	client *client.ApiClient
}

// NewDataPersistenceDataSource returns a new data source instance.
func NewDataPersistenceDataSource() datasource.DataSource {
	return &dataPersistenceDataSource{}
}

// Metadata returns the data source type name.
func (d *dataPersistenceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_data_persistence"
}

// Configure adds the provider configured client to the data source.
func (d *dataPersistenceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *dataPersistenceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The data persistence data source allows access to a list of supported data persistence options. Each option represents the rate at which a database will persist its data to storage.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier for this data source.",
				Computed:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"data_persistence": schema.SetNestedBlock{
				Description: "A list of data persistence options that can be applied to subscription databases.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "The identifier of the data persistence option.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "A meaningful description of the data persistence option.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}
