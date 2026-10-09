package agentmemory

import "github.com/hashicorp/terraform-plugin-framework/types"

type AgentMemoryAPIKeyResourceModel struct {
	ID              types.String `tfsdk:"id"`
	StoreID         types.String `tfsdk:"store_id"`
	Name            types.String `tfsdk:"name"`
	APIKey          types.String `tfsdk:"api_key"`
	ObfuscatedToken types.String `tfsdk:"obfuscated_token"`
	CreatedAt       types.Int64  `tfsdk:"created_at"`
}
