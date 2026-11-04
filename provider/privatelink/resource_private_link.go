package privatelink

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

var (
	_ resource.Resource                = &privateLinkResource{}
	_ resource.ResourceWithConfigure   = &privateLinkResource{}
	_ resource.ResourceWithImportState = &privateLinkResource{}
)

type privateLinkResource struct {
	client *client.ApiClient
}

// NewPrivateLinkResource returns the rediscloud_private_link resource.
func NewPrivateLinkResource() resource.Resource {
	return &privateLinkResource{}
}

func (r *privateLinkResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_private_link"
}

func (r *privateLinkResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *privateLinkResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Private Link to a pro Subscription in your Redis Enterprise Cloud Account.",
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
			"share_name": schema.StringAttribute{
				Description: "Name of this PrivateLink share",
				Required:    true,
			},
			"resource_configuration_id":  computedStableString("ID of the resource configuration to attach to this PrivateLink"),
			"resource_configuration_arn": computedStableString("ARN of the resource configuration attached to this PrivateLink"),
			"share_arn":                  computedStableString("ARN of the share attached to this Private Link"),
			"connections": schema.SetAttribute{
				Description: "Connections attached to this PrivateLink",
				Computed:    true,
				ElementType: connectionElemType,
			},
			"databases": schema.SetAttribute{
				Description: "The databases attached to this PrivateLink",
				Computed:    true,
				ElementType: databaseElemType,
			},
		},
		Blocks: map[string]schema.Block{
			"principal": principalBlock(),
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Read:   true,
				Update: true,
				Delete: true,
			}),
		},
	}
}

func computedStableString(description string) schema.StringAttribute {
	return schema.StringAttribute{
		Description: description,
		Computed:    true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}
}

func principalBlock() schema.SetNestedBlock {
	return schema.SetNestedBlock{
		Description: "List of principals attached to this PrivateLink",
		CustomType:  NewPrincipalSetType(),
		Validators: []validator.Set{
			setvalidator.IsRequired(),
			setvalidator.SizeAtLeast(1),
		},
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"principal": schema.StringAttribute{
					Required: true,
				},
				"principal_type": schema.StringAttribute{
					Required: true,
					Validators: []validator.String{
						stringvalidator.RegexMatches(
							regexp.MustCompile("^(aws_account|organization|organization_unit|iam_role|iam_user|service_principal)$"),
							"must be one of 'aws_account', 'organization', 'organization_unit', 'iam_role', 'iam_user', 'service_principal'",
						),
					},
				},
				"principal_alias": schema.StringAttribute{
					Optional: true,
				},
			},
		},
	}
}

func (r *privateLinkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, pathID, req, resp)
}
