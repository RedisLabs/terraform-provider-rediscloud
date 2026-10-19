package agentmemory

import (
	"context"

	agentmemoryapi "github.com/RedisLabs/rediscloud-go-api/service/agentmemory"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func modelConfigWithWriteOnlyValue(model, config *AgentMemoryModelConfigModel) *AgentMemoryModelConfigModel {
	if model == nil || model.Credentials == nil || config == nil || config.Credentials == nil {
		return model
	}
	result := *model
	credentials := *model.Credentials
	credentials.APIKey = config.Credentials.APIKey
	result.Credentials = &credentials
	return &result
}

func modelConfigForRequest(ctx context.Context, config tfsdk.Config, block string, model *AgentMemoryModelConfigModel, diags *diag.Diagnostics) *agentmemoryapi.ModelConfig {
	result := modelConfigFromModel(model)
	if model == nil || model.Credentials == nil {
		return result
	}
	// Write-only secrets must be read from configuration, never from plan or state.
	keyPath := path.Root(block).AtName("credentials").AtName("api_key")
	var key types.String
	diags.Append(config.GetAttribute(ctx, keyPath, &key)...)
	if diags.HasError() {
		return nil
	}
	if missingModelString(key) {
		diags.AddAttributeError(keyPath, "Missing Agent Memory write-only API key", "api_key must resolve to a non-blank value before sending model credentials to the API.")
		return nil
	}
	result.Credentials.APIKey = key.ValueString()
	return result
}
