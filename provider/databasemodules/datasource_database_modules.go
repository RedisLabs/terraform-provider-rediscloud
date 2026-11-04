package databasemodules

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

var (
	_ datasource.DataSource              = &databaseModulesDataSource{}
	_ datasource.DataSourceWithConfigure = &databaseModulesDataSource{}
)

type databaseModulesDataSource struct {
	client *client.ApiClient
}

// NewDatabaseModulesDataSource provides the Framework implementation for rediscloud_database_modules.
func NewDatabaseModulesDataSource() datasource.DataSource {
	return &databaseModulesDataSource{}
}

func (d *databaseModulesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_modules"
}

func (d *databaseModulesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *databaseModulesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Database modules data source allows access to the details of existing database modules within your Redis Enterprise Cloud account.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Identifier for this data source",
				Computed:    true,
			},
			// A computed set attribute, not a block: SDKv2 exposed this computed-only
			// collection as an attribute, and blocks plan as empty on deferred reads.
			"modules": schema.SetAttribute{
				Description: "A list of database modules",
				Computed:    true,
				ElementType: ModuleElemType,
			},
		},
	}
}
