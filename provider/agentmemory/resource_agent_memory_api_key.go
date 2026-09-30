package agentmemory

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

var (
	_ resource.Resource                = &agentMemoryAPIKeyResource{}
	_ resource.ResourceWithConfigure   = &agentMemoryAPIKeyResource{}
	_ resource.ResourceWithImportState = &agentMemoryAPIKeyResource{}
)

type agentMemoryAPIKeyResource struct {
	client *client.ApiClient
}

func NewAgentMemoryAPIKeyResource() resource.Resource {
	return &agentMemoryAPIKeyResource{}
}

func (r *agentMemoryAPIKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_agent_memory_api_key"
}

func (r *agentMemoryAPIKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *agentMemoryAPIKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	requiresReplace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Creates a data-plane API key for a Redis Agent Memory service.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The API key ID.",
				Computed:    true,
			},
			"store_id": schema.StringAttribute{
				Description: "The Agent Memory store ID.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
					stringvalidator.RegexMatches(regexp.MustCompile(`^[A-Za-z0-9-]+$`), "must contain only letters, numbers, and hyphens"),
				},
				PlanModifiers: requiresReplace,
			},
			"name": schema.StringAttribute{
				Description: "The name of the API key.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 100),
					stringvalidator.RegexMatches(regexp.MustCompile(`^\S+$`), "must not contain whitespace"),
				},
				PlanModifiers: requiresReplace,
			},
			"api_key": schema.StringAttribute{
				Description: "The generated data-plane API key. The API returns this secret only during creation.",
				Computed:    true,
				Sensitive:   true,
			},
			"obfuscated_token": schema.StringAttribute{
				Description: "The obfuscated API key returned by the Admin API.",
				Computed:    true,
			},
			"created_at": schema.Int64Attribute{
				Description: "The API key creation timestamp.",
				Computed:    true,
			},
		},
	}
}

// Import uses <store-id>/<api-key-id>; the secret API key cannot be recovered.
func (r *agentMemoryAPIKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	storeID, apiKeyID, ok := splitAPIKeyImportID(req.ID)
	if !ok {
		resp.Diagnostics.AddError("Invalid import ID", "Expected an import ID in the form <store-id>/<api-key-id>.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("store_id"), storeID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), apiKeyID)...)
}

func splitAPIKeyImportID(id string) (string, string, bool) {
	for i := 1; i < len(id)-1; i++ {
		if id[i] == '/' {
			return id[:i], id[i+1:], true
		}
	}
	return "", "", false
}
