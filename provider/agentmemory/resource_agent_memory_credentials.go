package agentmemory

import (
	"context"

	agentmemoryapi "github.com/RedisLabs/rediscloud-go-api/service/agentmemory"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func modelConfigWithWriteOnlyValue(ctx context.Context, model, config ModelConfigListValue, diags *diag.Diagnostics) ModelConfigListValue {
	if model.IsUnknown() || config.IsUnknown() {
		return model
	}
	m, c := modelConfigBlock(ctx, model, diags), modelConfigBlock(ctx, config, diags)
	if m == nil || c == nil || m.Credentials.IsUnknown() || c.Credentials.IsUnknown() {
		return model
	}
	credentials, configured := modelCredentialsBlock(ctx, m.Credentials, diags), modelCredentialsBlock(ctx, c.Credentials, diags)
	if credentials == nil || configured == nil {
		return model
	}
	credentials.APIKey = configured.APIKey
	m.Credentials = modelCredentialsList(ctx, credentials, diags)
	return modelConfigList(ctx, m, diags)
}

func modelConfigForRequest(ctx context.Context, config tfsdk.Config, block string, model *AgentMemoryModelConfigModel, diags *diag.Diagnostics) *agentmemoryapi.ModelConfig {
	result := modelConfigFromModel(ctx, model, diags)
	if model == nil || len(model.Credentials.Elements()) == 0 {
		return result
	}
	// Write-only secrets must be read from configuration, never from plan or state.
	keyPath := path.Root(block).AtListIndex(0).AtName("credentials").AtListIndex(0).AtName("api_key")
	var key types.String
	diags.Append(config.GetAttribute(ctx, keyPath, &key)...)
	if diags.HasError() {
		return nil
	}
	if missingModelString(key) {
		diags.AddAttributeError(keyPath, "Missing Agent Memory write-only API key", "api_key must resolve to a non-blank value before sending model credentials to the API.")
		return nil
	}
	result.Credentials.APIKey = key.ValueStringPointer()
	return result
}
