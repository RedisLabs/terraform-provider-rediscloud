package psc

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

var (
	_ resource.Resource                = &activeActivePrivateServiceConnectEndpointResource{}
	_ resource.ResourceWithConfigure   = &activeActivePrivateServiceConnectEndpointResource{}
	_ resource.ResourceWithImportState = &activeActivePrivateServiceConnectEndpointResource{}
)

type activeActivePrivateServiceConnectEndpointResource struct {
	client *client.ApiClient
}

// NewActiveActivePrivateServiceConnectEndpointResource returns the rediscloud_active_active_private_service_connect_endpoint resource.
func NewActiveActivePrivateServiceConnectEndpointResource() resource.Resource {
	return &activeActivePrivateServiceConnectEndpointResource{}
}

func (r *activeActivePrivateServiceConnectEndpointResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_active_active_private_service_connect_endpoint"
}

func (r *activeActivePrivateServiceConnectEndpointResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *activeActivePrivateServiceConnectEndpointResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attributes := endpointResourceAttributes("The ID of the Pro subscription to attach")
	//TODO(TF3.0) consider exposing region_id as a string
	attributes["region_id"] = schema.Int64Attribute{
		Description: "The ID of the GCP region",
		Required:    true,
		PlanModifiers: []planmodifier.Int64{
			int64planmodifier.RequiresReplace(),
		},
	}

	resp.Schema = schema.Schema{
		Description: "Manages a Private Service Connect Endpoint to an  Active-Active Subscription in your Redis Enterprise Cloud Account.",
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

func (r *activeActivePrivateServiceConnectEndpointResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
