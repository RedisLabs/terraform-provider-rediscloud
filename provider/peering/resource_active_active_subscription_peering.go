package peering

import (
	"context"
	"fmt"
	"regexp"

	"github.com/RedisLabs/rediscloud-go-api/service/cloud_accounts"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

var (
	_ resource.Resource                = &activeActiveSubscriptionPeeringResource{}
	_ resource.ResourceWithConfigure   = &activeActiveSubscriptionPeeringResource{}
	_ resource.ResourceWithImportState = &activeActiveSubscriptionPeeringResource{}
	_ resource.ResourceWithModifyPlan  = &activeActiveSubscriptionPeeringResource{}
)

type activeActiveSubscriptionPeeringResource struct {
	client *client.ApiClient
}

// NewActiveActiveSubscriptionPeeringResource returns a new resource instance.
func NewActiveActiveSubscriptionPeeringResource() resource.Resource {
	return &activeActiveSubscriptionPeeringResource{}
}

func (r *activeActiveSubscriptionPeeringResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_active_active_subscription_peering"
}

func (r *activeActiveSubscriptionPeeringResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *activeActiveSubscriptionPeeringResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates a VPC peering for an existing Redis Enterprise Cloud Active-Active Subscription, allowing access to your subscription databases as if they were on the same network.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the peering, as `<subscription_id>/<peering_id>`",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"subscription_id": schema.StringAttribute{
				Description: "A valid subscription predefined in the current account",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^\d+$`), "must be a number"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"provider_name": schema.StringAttribute{
				Description: "The cloud provider to use with the vpc peering, (either `AWS` or `GCP`)",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(providerAWS),
				Validators: []validator.String{
					stringvalidator.OneOf(cloud_accounts.ProviderValues()...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"source_region":      forceNewOptionalComputedString("AWS or GCP Region that the VPC to be peered lives in"),
			"destination_region": forceNewOptionalComputedString("AWS Region that the VPC to be peered lives in"),
			"aws_account_id":     forceNewOptionalComputedString("AWS account id that the VPC to be peered lives in"),
			"vpc_id":             forceNewOptionalComputedString("Identifier of the VPC to be peered"),
			"vpc_cidr": schema.StringAttribute{
				Description: "CIDR range of the VPC to be peered",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					cidrValidator{},
					stringvalidator.ConflictsWith(path.MatchRoot("vpc_cidrs")),
				},
				PlanModifiers: forceNewStringPlanModifiers(),
			},
			"vpc_cidrs": vpcCidrsAttribute(),
			// Unlike the other configurable attributes these never forced
			// replacement in SDKv2, so a change fails in Update instead.
			"gcp_project_id":         updatableOptionalComputedString("GCP project ID that the VPC to be peered lives in"),
			"gcp_network_name":       updatableOptionalComputedString("The name of the network to be peered"),
			"status":                 computedString("Current status of the account - `initiating-request`, `pending-acceptance`, `active`, `inactive` or `failed`"),
			"aws_peering_id":         computedString("Identifier of the AWS cloud peering"),
			"gcp_redis_project_id":   computedString("Identifier of the Redis Enterprise Cloud GCP project to be peered"),
			"gcp_redis_network_name": computedString("The name of the Redis Enterprise Cloud network to be peered"),
			"gcp_peering_id":         computedString("Identifier of the cloud peering"),
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Read:   true,
				Delete: true,
			}),
		},
	}
}

func updatableOptionalComputedString(description string) schema.StringAttribute {
	return schema.StringAttribute{
		Description: description,
		Optional:    true,
		Computed:    true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}
}
