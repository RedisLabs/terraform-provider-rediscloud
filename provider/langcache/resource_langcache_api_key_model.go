package langcache

import "github.com/hashicorp/terraform-plugin-framework/types"

type LangCacheAPIKeyResourceModel struct {
	ID              types.String `tfsdk:"id"`
	CacheID         types.String `tfsdk:"cache_id"`
	Name            types.String `tfsdk:"name"`
	APIKey          types.String `tfsdk:"api_key"`
	ObfuscatedToken types.String `tfsdk:"obfuscated_token"`
	CreatedAt       types.Int64  `tfsdk:"created_at"`
}
