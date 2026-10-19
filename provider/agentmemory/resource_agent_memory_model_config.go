package agentmemory

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func validateModelConfigPlan(plan, state AgentMemoryResourceModel, hasState bool, resp *resource.ModifyPlanResponse) {
	if plan.LLM == nil || plan.Embedding == nil {
		// Child modifiers do not run when a whole model block is removed.
		// Configuration validators enforce the pair before planning.
		if plan.LLM == nil && plan.Embedding == nil && hasState {
			if state.LLM != nil {
				resp.RequiresReplace.Append(path.Root("llm").AtName("provider"))
			}
			if state.Embedding != nil {
				resp.RequiresReplace.Append(path.Root("embedding").AtName("provider"))
			}
		}
		return
	}

	validateModelConfigBlock("llm", plan.LLM, !hasState, resp)
	validateModelConfigBlock("embedding", plan.Embedding, !hasState, resp)
	if !hasState && plan.LongTermTTLSeconds.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("long_term_ttl_seconds"),
			"Missing Agent Memory long-term TTL",
			"long_term_ttl_seconds must be configured when embedding is configured because the Agent Memory API requires longTermMemory.ttlSeconds whenever longTermMemory.embedding is sent on create.",
		)
	}
}

func validateModelConfigBlock(blockName string, model *AgentMemoryModelConfigModel, requireCredentials bool, resp *resource.ModifyPlanResponse) {
	if model == nil {
		return
	}
	if !model.Provider.IsUnknown() && missingModelString(model.Provider) {
		resp.Diagnostics.AddAttributeError(
			path.Root(blockName).AtName("provider"),
			"Missing Agent Memory model provider",
			fmt.Sprintf("%s.provider must be configured.", blockName),
		)
	}
	if !model.Model.IsUnknown() && missingModelString(model.Model) {
		resp.Diagnostics.AddAttributeError(
			path.Root(blockName).AtName("model"),
			"Missing Agent Memory model",
			fmt.Sprintf("%s.model must be configured.", blockName),
		)
	}
	if requireCredentials && model.Credentials == nil {
		resp.Diagnostics.AddAttributeError(
			path.Root(blockName).AtName("credentials"),
			"Missing Agent Memory model credentials",
			fmt.Sprintf("%s.credentials must be configured when creating a store with customer-managed models.", blockName),
		)
		return
	}
	if model.Credentials == nil {
		return
	}
	if !model.Credentials.Type.IsUnknown() && missingModelString(model.Credentials.Type) {
		resp.Diagnostics.AddAttributeError(
			path.Root(blockName).AtName("credentials").AtName("type"),
			"Missing Agent Memory model credential type",
			fmt.Sprintf("%s.credentials.type must be configured when credentials are present.", blockName),
		)
	}
	// A configured key may depend on another resource and remain unknown during plan.
	// Validate it once known; null and blank keys must still be rejected.
	if !model.Credentials.APIKey.IsUnknown() && missingModelString(model.Credentials.APIKey) {
		resp.Diagnostics.AddAttributeError(
			path.Root(blockName).AtName("credentials").AtName("api_key"),
			"Missing Agent Memory model API key",
			fmt.Sprintf("%s.credentials.api_key must be configured when credentials are present.", blockName),
		)
	}
}

func missingModelString(value types.String) bool {
	if value.IsNull() || value.IsUnknown() {
		return true
	}
	return strings.TrimSpace(value.ValueString()) == ""
}
