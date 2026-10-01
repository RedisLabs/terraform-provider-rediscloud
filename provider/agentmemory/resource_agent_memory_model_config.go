package agentmemory

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func validateModelConfigPlan(plan, state AgentMemoryResourceModel, hasState bool, resp *resource.ModifyPlanResponse) {
	switch {
	case plan.LLM == nil && plan.Embedding == nil:
		if hasState && (state.LLM != nil || state.Embedding != nil) {
			resp.Diagnostics.AddAttributeError(
				path.Root("llm"),
				"Unsupported Agent Memory model configuration removal",
				"The Agent Memory API does not support removing customer-managed model configuration after it has been configured.",
			)
		}
		return
	case plan.LLM == nil:
		resp.Diagnostics.AddAttributeError(
			path.Root("llm"),
			"Missing Agent Memory LLM model configuration",
			"llm must be configured when embedding is configured. The Agent Memory API requires llm and embedding to be supplied together for customer-managed models.",
		)
		return
	case plan.Embedding == nil:
		resp.Diagnostics.AddAttributeError(
			path.Root("embedding"),
			"Missing Agent Memory embedding model configuration",
			"embedding must be configured when llm is configured. The Agent Memory API requires llm and embedding to be supplied together for customer-managed models.",
		)
		return
	}

	validateModelConfigBlock("llm", plan.LLM, !hasState, resp)
	validateModelConfigBlock("embedding", plan.Embedding, !hasState, resp)
	if !hasState || resp.Diagnostics.HasError() {
		if !hasState && (plan.Embedding != nil) && (plan.LongTermTTLSeconds.IsNull() || plan.LongTermTTLSeconds.IsUnknown()) {
			resp.Diagnostics.AddAttributeError(
				path.Root("long_term_ttl_seconds"),
				"Missing Agent Memory long-term TTL",
				"long_term_ttl_seconds must be configured when embedding is configured because the Agent Memory API requires longTermMemory.ttlSeconds whenever longTermMemory.embedding is sent on create.",
			)
		}
		return
	}

	if state.LLM == nil || state.Embedding == nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("llm"),
			"Unsupported Agent Memory model configuration change",
			"The Agent Memory API does not allow a store created with platform-managed models to move to customer-managed models later. Configure llm and embedding when the store is first created.",
		)
		return
	}
	validateConfiguredModelConfigChange("llm", plan.LLM, state.LLM, true, resp)
	validateConfiguredModelConfigChange("embedding", plan.Embedding, state.Embedding, false, resp)
}

func validateModelConfigBlock(blockName string, model *AgentMemoryModelConfigModel, requireCredentials bool, resp *resource.ModifyPlanResponse) {
	if model == nil {
		return
	}
	if missingModelString(model.Provider) {
		resp.Diagnostics.AddAttributeError(
			path.Root(blockName).AtName("provider"),
			"Missing Agent Memory model provider",
			fmt.Sprintf("%s.provider must be configured.", blockName),
		)
	}
	if missingModelString(model.Model) {
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
	if missingModelString(model.Credentials.Type) {
		resp.Diagnostics.AddAttributeError(
			path.Root(blockName).AtName("credentials").AtName("type"),
			"Missing Agent Memory model credential type",
			fmt.Sprintf("%s.credentials.type must be configured when credentials are present.", blockName),
		)
	}
	if missingModelString(model.Credentials.APIKey) {
		resp.Diagnostics.AddAttributeError(
			path.Root(blockName).AtName("credentials").AtName("api_key"),
			"Missing Agent Memory model API key",
			fmt.Sprintf("%s.credentials.api_key must be configured when credentials are present.", blockName),
		)
	}
}

func validateConfiguredModelConfigChange(blockName string, plan, state *AgentMemoryModelConfigModel, allowModelChange bool, resp *resource.ModifyPlanResponse) {
	if plan == nil {
		resp.Diagnostics.AddAttributeError(
			path.Root(blockName),
			"Unsupported Agent Memory model configuration removal",
			fmt.Sprintf("The Agent Memory API does not support removing %s after customer-managed models have been configured.", blockName),
		)
		return
	}
	if !sameKnownString(plan.Provider, state.Provider) {
		resp.Diagnostics.AddAttributeError(
			path.Root(blockName).AtName("provider"),
			"Unsupported Agent Memory model provider change",
			fmt.Sprintf("The Agent Memory API treats %s.provider as immutable after creation.", blockName),
		)
	}
	if !allowModelChange && !sameKnownString(plan.Model, state.Model) {
		resp.Diagnostics.AddAttributeError(
			path.Root(blockName).AtName("model"),
			"Unsupported Agent Memory embedding model change",
			"The Agent Memory API treats embedding.model as immutable after creation. Only the embedding credential can be rotated.",
		)
	}
}

func missingModelString(value types.String) bool {
	if value.IsNull() || value.IsUnknown() {
		return true
	}
	return strings.TrimSpace(value.ValueString()) == ""
}

func sameKnownString(left, right types.String) bool {
	if left.IsNull() || left.IsUnknown() || right.IsNull() || right.IsUnknown() {
		return true
	}
	return left.Equal(right)
}
