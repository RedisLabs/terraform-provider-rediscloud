package aclrole

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

var (
	_ datasource.DataSource              = &aclRoleDataSource{}
	_ datasource.DataSourceWithConfigure = &aclRoleDataSource{}
)

type aclRoleDataSource struct {
	client *client.ApiClient
}

// NewAclRoleDataSource returns a new data source instance.
func NewAclRoleDataSource() datasource.DataSource {
	return &aclRoleDataSource{}
}

func (d *aclRoleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_acl_role"
}

func (d *aclRoleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *aclRoleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The ACL Role grants a number of permissions to databases",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the ACL role",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "A meaningful name to identify the user",
				Required:    true,
			},
			// A computed attribute rather than a block, as SDKv2 exposed it: a deferred read
			// plans blocks as empty sets instead of unknown.
			"rule": schema.SetAttribute{
				Description: "This Role's permissions and the databases to which they apply",
				Computed:    true,
				CustomType:  NewRuleSetType(),
				ElementType: NewRuleSetType().ElementType(),
			},
		},
	}
}
