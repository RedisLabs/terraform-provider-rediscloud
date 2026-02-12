package cloudaccount

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
	_ datasource.DataSource              = &cloudAccountDataSource{}
	_ datasource.DataSourceWithConfigure = &cloudAccountDataSource{}
)

// cloudAccountDataSource is the data source implementation.
type cloudAccountDataSource struct {
	client *client.ApiClient
}

// NewCloudAccountDataSource returns a new data source instance.
func NewCloudAccountDataSource() datasource.DataSource {
	return &cloudAccountDataSource{}
}

// Metadata returns the data source type name.
func (d *cloudAccountDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloud_account"
}

// Configure adds the provider configured client to the data source.
func (d *cloudAccountDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *cloudAccountDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Cloud Account data source allows access to the ID of a Cloud Account configuration.  This ID can be used when creating Subscription resources.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The ID of the cloud account",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "A meaningful name to identify the cloud account",
				Optional:    true,
				Computed:    true,
			},
			"access_key_id": schema.StringAttribute{
				Description: "The access key ID associated with the cloud account",
				Computed:    true,
			},
			"exclude_internal_account": schema.BoolAttribute{
				Description: "Whether to exclude the Redis Labs internal cloud account.",
				Optional:    true,
				Computed:    true,
			},
			"provider_type": schema.StringAttribute{
				Description: "The cloud provider of the cloud account, (either `AWS` or `GCP`)",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(cloud_accounts.ProviderValues()...),
				},
			},
		},
	}
}
