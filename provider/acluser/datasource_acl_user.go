package acluser

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &aclUserDataSource{}
	_ datasource.DataSourceWithConfigure = &aclUserDataSource{}
)

// aclUserDataSource is the data source implementation.
type aclUserDataSource struct {
	client *client.ApiClient
}

// NewAclUserDataSource returns a new data source instance.
func NewAclUserDataSource() datasource.DataSource {
	return &aclUserDataSource{}
}

// Metadata returns the data source type name.
func (d *aclUserDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_acl_user"
}

// Configure adds the provider configured client to the data source.
func (d *aclUserDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *aclUserDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The ACL User is an authenticated entity whose permissions are described by an associated Role",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the ACL user",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "A meaningful name to identify the user",
				Required:    true,
			},
			"role": schema.StringAttribute{
				Description: "The Role which this User has",
				Computed:    true,
			},
		},
	}
}
