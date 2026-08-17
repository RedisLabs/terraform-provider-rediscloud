package langcache

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

var (
	_ resource.Resource                = &langCacheAPIKeyResource{}
	_ resource.ResourceWithConfigure   = &langCacheAPIKeyResource{}
	_ resource.ResourceWithImportState = &langCacheAPIKeyResource{}
)

type langCacheAPIKeyResource struct {
	client *client.ApiClient
}

func NewLangCacheAPIKeyResource() resource.Resource {
	return &langCacheAPIKeyResource{}
}

func (r *langCacheAPIKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_langcache_api_key"
}

func (r *langCacheAPIKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *langCacheAPIKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	requiresReplace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Creates a data-plane API key for a Redis LangCache.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The API key ID.",
				Computed:    true,
			},
			"cache_id": schema.StringAttribute{
				Description:   "The LangCache ID.",
				Required:      true,
				PlanModifiers: requiresReplace,
			},
			"name": schema.StringAttribute{
				Description:   "The name of the API key.",
				Required:      true,
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

// Import uses <cache-id>/<api-key-id>; the secret API key cannot be recovered.
func (r *langCacheAPIKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	cacheID, apiKeyID, ok := splitAPIKeyImportID(req.ID)
	if !ok {
		resp.Diagnostics.AddError("Invalid import ID", "Expected an import ID in the form <cache-id>/<api-key-id>.")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cache_id"), cacheID)...)
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
