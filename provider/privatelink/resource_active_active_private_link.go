package privatelink

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

var (
	_ resource.Resource                = &activeActivePrivateLinkResource{}
	_ resource.ResourceWithConfigure   = &activeActivePrivateLinkResource{}
	_ resource.ResourceWithImportState = &activeActivePrivateLinkResource{}
)

type activeActivePrivateLinkResource struct {
	client *client.ApiClient
}

// NewActiveActivePrivateLinkResource returns the rediscloud_active_active_private_link resource.
func NewActiveActivePrivateLinkResource() resource.Resource {
	return &activeActivePrivateLinkResource{}
}

func (r *activeActivePrivateLinkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_active_active_private_link"
}

func (r *activeActivePrivateLinkResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *activeActivePrivateLinkResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Private Link to an Active Active Subscription in your Redis Enterprise Cloud Account.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"subscription_id": schema.StringAttribute{
				Description: "The ID of the Pro subscription to attach",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"region_id": schema.Int64Attribute{
				Description: "The RedisCloud ID of the active active subscription region",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"share_name": schema.StringAttribute{
				Description: "Name of this PrivateLink share",
				Required:    true,
			},
			"resource_configuration_id":  computedStableString("ID of the resource configuration to attach to this PrivateLink"),
			"resource_configuration_arn": computedStableString("ARN of the resource configuration attached to this PrivateLink"),
			"share_arn":                  computedStableString("ARN of the share to attach to this Private Link"),
			"connections": schema.SetAttribute{
				Description: "Connections attached to this PrivateLink",
				Computed:    true,
				ElementType: connectionElemType,
			},
			"databases": schema.SetAttribute{
				Computed:    true,
				ElementType: databaseElemType,
			},
		},
		Blocks: map[string]schema.Block{
			"principal": principalBlock("Must be an allowed Principal Type. ('aws_account', 'organization', 'organization_unit', 'iam_role', 'iam_user', 'service_principal')'"),
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Read:   true,
				Update: true,
				Delete: true,
			}),
		},
	}
}

func (r *activeActivePrivateLinkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, pathID, req, resp)
}
