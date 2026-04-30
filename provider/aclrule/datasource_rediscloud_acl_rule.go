package aclrule

import (
	"context"
	"fmt"


	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &aclRuleDataSource{}
	_ datasource.DataSourceWithConfigure = &aclRuleDataSource{}
)

// aclRuleDataSource is the data source implementation.
type aclRuleDataSource struct {
	client *client.ApiClient
}

// NewAclRuleDataSource returns a new data source instance.
func NewAclRuleDataSource() datasource.DataSource {
	return &aclRuleDataSource{}
}

func (d *aclRuleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_acl_rule"
}

func (d *aclRuleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The ACL Rule (known also as RedisRule) allows fine-grained permissions to be assigned to a subset of ACL Users",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the ACL rule",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "A meaningful name to identify the rule",
				Required:    true,
			},
			"rule": schema.StringAttribute{
				Description: "The Rule itself, must comply with Redis' ACL syntax",
				Computed:    true,
			},
		},
	}
}

func (d *aclRuleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

