package agentmemory

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func validateModelConfigPlan(ctx context.Context, plan, state AgentMemoryResourceModel, hasState bool, resp *resource.ModifyPlanResponse) {
	if plan.LLM.IsUnknown() || plan.Embedding.IsUnknown() {
		// An unknown model block may switch between managed and customer models.
		if hasState {
			resp.RequiresReplace.Append(path.Root("llm"), path.Root("embedding"))
		}
		return
	}
	llm := modelConfigBlock(ctx, plan.LLM, &resp.Diagnostics)
	embedding := modelConfigBlock(ctx, plan.Embedding, &resp.Diagnostics)
	if llm == nil || embedding == nil {
		if llm == nil && embedding == nil && hasState {
			if len(state.LLM.Elements()) > 0 {
				resp.RequiresReplace.Append(path.Root("llm"))
			}
			if len(state.Embedding.Elements()) > 0 {
				resp.RequiresReplace.Append(path.Root("embedding"))
			}
		}
		return
	}
	validateModelConfigBlock(ctx, "llm", llm, !hasState, resp)
	validateModelConfigBlock(ctx, "embedding", embedding, !hasState, resp)
	if !hasState && plan.LongTermTTLSeconds.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("long_term_ttl_seconds"), "Missing Agent Memory long-term TTL", "long_term_ttl_seconds must be configured when embedding is configured because the Agent Memory API requires longTermMemory.ttlSeconds whenever longTermMemory.embedding is sent on create.")
	}
}

func validateModelConfigBlock(ctx context.Context, blockName string, model *AgentMemoryModelConfigModel, requireCredentials bool, resp *resource.ModifyPlanResponse) {
	if model == nil {
		return
	}
	if !model.Provider.IsUnknown() && missingModelString(model.Provider) {
		resp.Diagnostics.AddAttributeError(
			path.Root(blockName).AtListIndex(0).AtName("provider"),
			"Missing Agent Memory model provider",
			fmt.Sprintf("%s[0].provider must be configured.", blockName),
		)
	}
	if !model.Model.IsUnknown() && missingModelString(model.Model) {
		resp.Diagnostics.AddAttributeError(
			path.Root(blockName).AtListIndex(0).AtName("model"),
			"Missing Agent Memory model",
			fmt.Sprintf("%s[0].model must be configured.", blockName),
		)
	}
	if model.Credentials.IsUnknown() {
		return
	}
	credentials := modelCredentialsBlock(ctx, model.Credentials, &resp.Diagnostics)
	if requireCredentials && credentials == nil {
		resp.Diagnostics.AddAttributeError(
			path.Root(blockName).AtListIndex(0).AtName("credentials"),
			"Missing Agent Memory model credentials",
			fmt.Sprintf("%s[0].credentials must be configured when creating a store with customer-managed models.", blockName),
		)
		return
	}
	if credentials == nil {
		return
	}
	if !credentials.Type.IsUnknown() && missingModelString(credentials.Type) {
		resp.Diagnostics.AddAttributeError(
			path.Root(blockName).AtListIndex(0).AtName("credentials").AtListIndex(0).AtName("type"),
			"Missing Agent Memory model credential type",
			fmt.Sprintf("%s[0].credentials[0].type must be configured when credentials are present.", blockName),
		)
	}
	// A configured key may depend on another resource and remain unknown during plan.
	// Validate it once known; null and blank keys must still be rejected.
	if !credentials.APIKey.IsUnknown() && missingModelString(credentials.APIKey) {
		resp.Diagnostics.AddAttributeError(
			path.Root(blockName).AtListIndex(0).AtName("credentials").AtListIndex(0).AtName("api_key"),
			"Missing Agent Memory model API key",
			fmt.Sprintf("%s[0].credentials[0].api_key must be configured when credentials are present.", blockName),
		)
	}
}

func missingModelString(value types.String) bool {
	if value.IsNull() || value.IsUnknown() {
		return true
	}
	return strings.TrimSpace(value.ValueString()) == ""
}

// Empty block lists are absent, while an unknown list must defer the pair check.
type modelPairValidator struct{}

func (modelPairValidator) Description(context.Context) string {
	return "llm and embedding must be configured together."
}
func (v modelPairValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}
func (modelPairValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var llm, embedding ModelConfigListValue
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("llm"), &llm)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("embedding"), &embedding)...)
	if resp.Diagnostics.HasError() || llm.IsUnknown() || embedding.IsUnknown() {
		return
	}
	if (len(llm.Elements()) > 0) != (len(embedding.Elements()) > 0) {
		resp.Diagnostics.AddError("Invalid Attribute Combination", "These attributes must be configured together: [llm,embedding]")
	}
}
