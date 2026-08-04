package cloudaccount

import (
	"context"
	"fmt"

	"github.com/RedisLabs/rediscloud-go-api/service/cloud_accounts"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &cloudAccountResource{}
	_ resource.ResourceWithConfigure   = &cloudAccountResource{}
	_ resource.ResourceWithImportState = &cloudAccountResource{}
)

// cloudAccountResource is the resource implementation.
type cloudAccountResource struct {
	client *client.ApiClient
}

// NewCloudAccountResource returns a new resource instance.
func NewCloudAccountResource() resource.Resource {
	return &cloudAccountResource{}
}

// Metadata returns the resource type name.
func (r *cloudAccountResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloud_account"
}

// Configure adds the provider configured client to the resource.
func (r *cloudAccountResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.ApiClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.ApiClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = c
}

// Schema defines the schema for the resource.
func (r *cloudAccountResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates a Cloud Account resource representing the access credentials to a cloud provider account, (`AWS` or `GCP`). Your Redis Enterprise Cloud account uses these credentials to provision databases within your infrastructure.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the cloud account",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"access_key_id": schema.StringAttribute{
				Description: "Cloud provider access key",
				Required:    true,
			},
			"access_secret_key": schema.StringAttribute{
				Description: "Cloud provider secret key",
				Required:    true,
				Sensitive:   true,
			},
			"console_password": schema.StringAttribute{
				Description: "Cloud provider management console password",
				Required:    true,
				Sensitive:   true,
			},
			"console_username": schema.StringAttribute{
				Description: "Cloud provider management console username",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "Display name of the account",
				Required:    true,
			},
			"provider_type": schema.StringAttribute{
				Description: "Cloud provider type - either `AWS` or `GCP`",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(cloud_accounts.ProviderValues()...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"sign_in_login_url": schema.StringAttribute{
				Description: "Cloud provider management console login URL",
				Required:    true,
			},
			"status": schema.StringAttribute{
				Description: "The current status of the account - `draft`, `pending` or `active`",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Read:   true,
				Update: true,
				Delete: true,
			}),
		},
	}
}
