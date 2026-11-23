package essentialssubscription

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

var (
	_ resource.Resource                = &essentialsSubscriptionResource{}
	_ resource.ResourceWithConfigure   = &essentialsSubscriptionResource{}
	_ resource.ResourceWithImportState = &essentialsSubscriptionResource{}
)

type essentialsSubscriptionResource struct {
	client *client.ApiClient
}

// NewEssentialsSubscriptionResource returns a new resource instance.
func NewEssentialsSubscriptionResource() resource.Resource {
	return &essentialsSubscriptionResource{}
}

// Metadata returns the resource type name.
func (r *essentialsSubscriptionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_essentials_subscription"
}

// Configure adds the provider configured client to the resource.
func (r *essentialsSubscriptionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *essentialsSubscriptionResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Essentials Subscription within your Redis Enterprise Cloud Account.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the subscription",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "A meaningful name to identify the subscription",
				Required:    true,
			},
			"status": schema.StringAttribute{
				Description: "The status of this subscription",
				Computed:    true,
			},
			//TODO(TF3.0) refactor plan_id to a StringAttribute
			"plan_id": schema.Int64Attribute{
				Description: "The identifier of the plan to template the subscription",
				Required:    true,
			},
			// SDKv2 defaulted this to credit-card and suppressed every diff after creation.
			// Computed + UseStateForUnknown keeps the prior value when the attribute is removed from config.
			"payment_method": schema.StringAttribute{
				Description: "Payment method for the requested subscription. If credit-card is specified, the payment method id must be defined. This information is only used when creating a new subscription and any changes will be ignored after this.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile("^(credit-card|marketplace)$"), "must be 'credit-card' or 'marketplace'"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			//TODO(TF3.0) refactor payment_method_id to a nullable StringAttribute
			"payment_method_id": schema.Int64Attribute{
				Description: "The identifier of the method which will be charged for this subscription. Not required for free plans",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"creation_date": schema.StringAttribute{
				Description: "The date/time this subscription was created",
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
