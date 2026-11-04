package transitgateway

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

var (
	_ resource.Resource                = &activeActiveTransitGatewayAttachmentResource{}
	_ resource.ResourceWithConfigure   = &activeActiveTransitGatewayAttachmentResource{}
	_ resource.ResourceWithImportState = &activeActiveTransitGatewayAttachmentResource{}
)

type activeActiveTransitGatewayAttachmentResource struct {
	client *client.ApiClient
}

// NewActiveActiveTransitGatewayAttachmentResource provides the Framework implementation for rediscloud_active_active_transit_gateway_attachment.
func NewActiveActiveTransitGatewayAttachmentResource() resource.Resource {
	return &activeActiveTransitGatewayAttachmentResource{}
}

func (r *activeActiveTransitGatewayAttachmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_active_active_transit_gateway_attachment"
}

func (r *activeActiveTransitGatewayAttachmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *activeActiveTransitGatewayAttachmentResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Transit Gateway Attachment to an Active Active Subscription in your Redis Enterprise Cloud Account.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The identifier of the attachment, in the form <subscription_id>/<region_id>/<tgw_id>",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"subscription_id": schema.StringAttribute{
				Description: "The ID of the Active-Active subscription to attach",
				Required:    true,
			},
			"region_id": schema.StringAttribute{
				Description: "The id of the AWS region",
				Required:    true,
			},
			"tgw_id": schema.Int64Attribute{
				Description: "The id of the Transit Gateway to attach to",
				Required:    true,
			},
			"aws_tgw_uid":       transitGatewayAttachmentStableAttribute("The id of the Transit Gateway as known to AWS"),
			"attachment_uid":    transitGatewayAttachmentStableAttribute("A unique identifier for the Subscription/Transit Gateway attachment, if established"),
			"status":            schema.StringAttribute{Description: "The status of the Transit Gateway", Computed: true},
			"attachment_status": schema.StringAttribute{Description: "The status of the Subscription/Transit Gateway attachment, if established", Computed: true},
			"aws_account_id":    transitGatewayAttachmentStableAttribute("The Transit Gateway's AWS account id"),
			"cidrs":             transitGatewayAttachmentCidrsAttribute("A list of consumer CIDR blocks. It is recommended to use the rediscloud_active_active_transit_gateway_route resource instead for managing CIDRs."),
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

func (r *activeActiveTransitGatewayAttachmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
