package langcache

import (
	"context"
	"fmt"

	langcacheapi "github.com/RedisLabs/rediscloud-go-api/service/langcache"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

var (
	_ resource.Resource                   = &langCacheResource{}
	_ resource.ResourceWithConfigure      = &langCacheResource{}
	_ resource.ResourceWithImportState    = &langCacheResource{}
	_ resource.ResourceWithValidateConfig = &langCacheResource{}
)

type langCacheResource struct {
	client *client.ApiClient
}

func NewLangCacheResource() resource.Resource {
	return &langCacheResource{}
}

func (r *langCacheResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_langcache"
}

func (r *langCacheResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *langCacheResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	requiresReplace := []planmodifier.String{stringplanmodifier.RequiresReplace()}

	resp.Schema = schema.Schema{
		Description: "Creates a Redis LangCache backed by an existing Redis Cloud database.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The LangCache ID.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the LangCache.",
				Required:    true,
			},
			"database_id": schema.StringAttribute{
				Description:   "The ID of the Redis Cloud database that backs the LangCache.",
				Required:      true,
				PlanModifiers: requiresReplace,
			},
			"embedding_model_provider": schema.StringAttribute{
				Description: "The embedding provider ID.",
				Required:    true,
			},
			"embedding_model_name": schema.StringAttribute{
				Description: "The embedding model ID.",
				Required:    true,
			},
			"embedding_model_api_key": schema.StringAttribute{
				Description: "API key for embedding providers that require one.",
				Optional:    true,
				Sensitive:   true,
			},
			"custom_model_base_url": schema.StringAttribute{
				Description: "Base URL for a custom embedding model. Must be set together with custom_model_dimensions.",
				Optional:    true,
			},
			"custom_model_dimensions": schema.Int32Attribute{
				Description: "Vector dimensions for a custom embedding model. Must be set together with custom_model_base_url.",
				Optional:    true,
			},
			"default_search_threshold": schema.Float64Attribute{
				Description: "Default distance threshold used when searching cache entries.",
				Required:    true,
			},
			"default_ttl_millis": schema.Int64Attribute{
				Description: "Default entry TTL in milliseconds. Zero means entries are not stored; -1 stores entries indefinitely.",
				Required:    true,
			},
			"attributes": schema.SetAttribute{
				Description: "Custom attributes available for filtering cache entries.",
				Required:    true,
				ElementType: types.StringType,
			},
			"search_strategies": schema.ListAttribute{
				Description: "Search strategies in priority order. Supported values are exact and semantic.",
				Optional:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(stringvalidator.OneOf(
						langcacheapi.SearchStrategyExact,
						langcacheapi.SearchStrategySemantic,
					)),
				},
			},
			"flush_on_destroy": schema.BoolAttribute{
				Description: "Whether to flush cache entries when destroying the LangCache. Defaults to false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"endpoint": schema.StringAttribute{
				Description: "The API endpoint for the LangCache.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "The current LangCache provisioning status.",
				Computed:    true,
			},
		},
	}
}

func (r *langCacheResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config LangCacheResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	baseURLSet := !config.CustomModelBaseURL.IsNull() && !config.CustomModelBaseURL.IsUnknown()
	dimensionsSet := !config.CustomModelDimensions.IsNull() && !config.CustomModelDimensions.IsUnknown()
	if baseURLSet != dimensionsSet {
		resp.Diagnostics.AddError(
			"Incomplete custom embedding model configuration",
			"custom_model_base_url and custom_model_dimensions must either both be configured or both be omitted.",
		)
	}
}

func (r *langCacheResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
