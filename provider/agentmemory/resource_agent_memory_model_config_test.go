package agentmemory

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateModelConfigPlanRejectsProviderChange(t *testing.T) {
	tests := []struct {
		name  string
		plan  AgentMemoryResourceModel
		state AgentMemoryResourceModel
	}{
		{
			name: "llm provider",
			state: agentMemoryModelConfigResourceModel(
				agentMemoryModelConfigModel("openai", "gpt-4o-mini"),
				agentMemoryModelConfigModel("openai", "text-embedding-3-small"),
			),
			plan: agentMemoryModelConfigResourceModel(
				agentMemoryModelConfigModel("anthropic", "gpt-4o-mini"),
				agentMemoryModelConfigModel("openai", "text-embedding-3-small"),
			),
		},
		{
			name: "embedding provider",
			state: agentMemoryModelConfigResourceModel(
				agentMemoryModelConfigModel("openai", "gpt-4o-mini"),
				agentMemoryModelConfigModel("openai", "text-embedding-3-small"),
			),
			plan: agentMemoryModelConfigResourceModel(
				agentMemoryModelConfigModel("openai", "gpt-4o-mini"),
				agentMemoryModelConfigModel("cohere", "text-embedding-3-small"),
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp resource.ModifyPlanResponse
			validateModelConfigPlan(tt.plan, tt.state, true, &resp)

			require.True(t, resp.Diagnostics.HasError())
			require.Len(t, resp.Diagnostics.Errors(), 1)
			assert.Equal(t, "Unsupported Agent Memory model provider change", resp.Diagnostics.Errors()[0].Summary())
		})
	}
}

func agentMemoryModelConfigResourceModel(llm, embedding *AgentMemoryModelConfigModel) AgentMemoryResourceModel {
	return AgentMemoryResourceModel{
		LLM:       llm,
		Embedding: embedding,
	}
}

func agentMemoryModelConfigModel(provider, model string) *AgentMemoryModelConfigModel {
	return &AgentMemoryModelConfigModel{
		Provider: types.StringValue(provider),
		Model:    types.StringValue(model),
	}
}
