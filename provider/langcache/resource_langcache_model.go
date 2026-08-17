package langcache

import "github.com/hashicorp/terraform-plugin-framework/types"

// LangCacheResourceModel describes a LangCache resource.
type LangCacheResourceModel struct {
	ID                     types.String  `tfsdk:"id"`
	Name                   types.String  `tfsdk:"name"`
	DatabaseID             types.String  `tfsdk:"database_id"`
	EmbeddingModelProvider types.String  `tfsdk:"embedding_model_provider"`
	EmbeddingModelName     types.String  `tfsdk:"embedding_model_name"`
	EmbeddingModelAPIKey   types.String  `tfsdk:"embedding_model_api_key"`
	CustomModelBaseURL     types.String  `tfsdk:"custom_model_base_url"`
	CustomModelDimensions  types.Int32   `tfsdk:"custom_model_dimensions"`
	DefaultSearchThreshold types.Float64 `tfsdk:"default_search_threshold"`
	DefaultTTLMillis       types.Int64   `tfsdk:"default_ttl_millis"`
	Attributes             types.Set     `tfsdk:"attributes"`
	SearchStrategies       types.List    `tfsdk:"search_strategies"`
	FlushOnDestroy         types.Bool    `tfsdk:"flush_on_destroy"`
	Endpoint               types.String  `tfsdk:"endpoint"`
	Status                 types.String  `tfsdk:"status"`
}
