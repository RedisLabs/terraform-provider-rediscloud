package psc

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

var (
	_ resource.Resource                = &privateServiceConnectEndpointResource{}
	_ resource.ResourceWithConfigure   = &privateServiceConnectEndpointResource{}
	_ resource.ResourceWithImportState = &privateServiceConnectEndpointResource{}
)

type privateServiceConnectEndpointResource struct {
	client *client.ApiClient
}

// NewPrivateServiceConnectEndpointResource returns the rediscloud_private_service_connect_endpoint resource.
func NewPrivateServiceConnectEndpointResource() resource.Resource {
	return &privateServiceConnectEndpointResource{}
}

func (r *privateServiceConnectEndpointResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_private_service_connect_endpoint"
}

func (r *privateServiceConnectEndpointResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *privateServiceConnectEndpointResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := endpointResourceAttributes("The ID of the Pro subscription to attach")
	resp.Schema = schema.Schema{
		Description: "Manages a Private Service Connect Endpoint to a Pro Subscription in your Redis Enterprise Cloud Account.",
		Attributes:  attributes,
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Read:   true,
				Delete: true,
			}),
		},
	}
}

// endpointResourceAttributes returns the attributes shared by the Pro and
// Active-Active Private Service Connect endpoint resources.
func endpointResourceAttributes(subscriptionDescription string) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed: true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"subscription_id": requiredReplaceString(subscriptionDescription),
		//TODO(TF3.0) consider exposing private_service_connect_service_id as a string
		"private_service_connect_service_id": schema.Int64Attribute{
			Description: "The ID of the Private Service Connect",
			Required:    true,
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.RequiresReplace(),
			},
		},
		//TODO(TF3.0) consider exposing private_service_connect_endpoint_id as a string
		"private_service_connect_endpoint_id": schema.Int64Attribute{
			Description: "The ID of the Private Service Connect Endpoint",
			Computed:    true,
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
		"gcp_project_id":           requiredReplaceString("The Google Cloud Project ID"),
		"gcp_vpc_name":             requiredReplaceString("The GCP VPC Network name"),
		"gcp_vpc_subnet_name":      requiredReplaceString("The GCP Subnet name"),
		"endpoint_connection_name": requiredReplaceString("The endpoint connection name prefix. This prefix that will be used to create the Private Service Connect endpoint in your Google Cloud account"),
		"service_attachments": schema.ListAttribute{
			Description: "The service attachments that were created for the Private Service Connect endpoint",
			Computed:    true,
			ElementType: serviceAttachmentElemType,
			PlanModifiers: []planmodifier.List{
				listplanmodifier.UseStateForUnknown(),
			},
		},
	}
}

func requiredReplaceString(description string) schema.StringAttribute {
	return schema.StringAttribute{
		Description: description,
		Required:    true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.RequiresReplace(),
		},
	}
}

func (r *privateServiceConnectEndpointResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
