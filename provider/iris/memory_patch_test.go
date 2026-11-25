package iris

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func TestMemoryPatchPreservesUnconfiguredGroupSettings(t *testing.T) {
	previous := AgentMemoryModel{Timeouts: nullTimeouts(),
		Name:                 types.StringValue("store"),
		SummarizationEnabled: types.BoolValue(true), SummarizationTriggerStrategy: types.StringValue("EVENT_COUNT"),
		SummarizationEventThreshold: types.Int64Value(100), SummarizationRetainCount: types.Int64Value(10),
		ExclusionsEnabled: types.BoolValue(true), SemanticExclusionsEnabled: types.BoolValue(true), SemanticExclusionsPrompt: types.StringValue("old prompt"),
		BuiltInDetectors: types.SetValueMust(builtInDetectorElementType, nil), CustomDetectors: types.SetValueMust(customDetectorElementType, nil),
	}
	for _, tc := range []struct {
		name string
		edit func(*AgentMemoryModel)
		body string
	}{
		{"rename", func(m *AgentMemoryModel) { m.Name = types.StringValue("renamed") }, `{"name":"renamed"}`},
		{"disable summary", func(m *AgentMemoryModel) { m.SummarizationEnabled = types.BoolValue(false) }, `{"summarization":{"enabled":false,"triggerStrategy":"EVENT_COUNT","eventCount":{"threshold":100,"retainCount":10}}}`},
		{"zero retained count", func(m *AgentMemoryModel) { m.SummarizationRetainCount = types.Int64Value(0) }, `{"summarization":{"enabled":true,"triggerStrategy":"EVENT_COUNT","eventCount":{"threshold":100,"retainCount":0}}}`},
		{"semantic prompt", func(m *AgentMemoryModel) { m.SemanticExclusionsPrompt = types.StringValue("new prompt") }, `{"longTermMemoryExclusions":{"enabled":true,"semantic":{"enabled":true,"prompt":"new prompt"}}}`},
		{"disable exclusions", func(m *AgentMemoryModel) { m.ExclusionsEnabled = types.BoolValue(false) }, `{"longTermMemoryExclusions":{"enabled":false}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plan := previous
			// Optional+Computed settings omitted from HCL can be unknown during
			// an update. They must neither trigger writes nor reset sibling fields.
			plan.SummarizationEnabled = types.BoolUnknown()
			plan.SummarizationTriggerStrategy = types.StringUnknown()
			plan.SummarizationEventThreshold = types.Int64Unknown()
			plan.SummarizationRetainCount = types.Int64Unknown()
			plan.ExclusionsEnabled = types.BoolUnknown()
			plan.SemanticExclusionsEnabled = types.BoolUnknown()
			plan.SemanticExclusionsPrompt = types.StringUnknown()
			tc.edit(&plan)
			body, d := plan.apiPatch(context.Background(), AgentMemoryModel{Timeouts: nullTimeouts()}, previous)
			require.False(t, d.HasError(), d)
			encoded, err := json.Marshal(body)
			require.NoError(t, err)
			require.JSONEq(t, tc.body, string(encoded))
		})
	}
}

func TestMemoryPatchRotatesOnlyVersionedCredentials(t *testing.T) {
	previous := AgentMemoryModel{Timeouts: nullTimeouts(),
		LLMProvider: types.StringValue("openai"), LLMModel: types.StringValue("llm"), LLMAPIKeyVersion: types.Int64Value(1),
		EmbeddingProvider: types.StringValue("openai"), EmbeddingModel: types.StringValue("embedding"), EmbeddingAPIKeyVersion: types.Int64Value(1),
		ClassificationProvider: types.StringValue("openai"), ClassificationModel: types.StringValue("classification"), ClassificationAPIKeyVersion: types.Int64Value(1),
	}
	config := AgentMemoryModel{Timeouts: nullTimeouts(), LLMAPIKey: types.StringValue("fake-llm"), EmbeddingAPIKey: types.StringValue("fake-embedding"), ClassificationAPIKey: types.StringValue("fake-classification")}
	plan := previous
	body, d := plan.apiPatch(context.Background(), config, previous)
	require.False(t, d.HasError(), d)
	require.Equal(t, agentMemorySettings{}, body, "keys alone do not trigger rotation")
	plan.LLMProvider, plan.LLMModel = types.StringUnknown(), types.StringUnknown()
	plan.EmbeddingProvider, plan.EmbeddingModel = types.StringUnknown(), types.StringUnknown()
	plan.ClassificationProvider, plan.ClassificationModel = types.StringUnknown(), types.StringUnknown()
	plan.LLMAPIKeyVersion, plan.EmbeddingAPIKeyVersion, plan.ClassificationAPIKeyVersion = types.Int64Value(2), types.Int64Value(2), types.Int64Value(2)
	body, d = plan.apiPatch(context.Background(), config, previous)
	require.False(t, d.HasError(), d)
	encoded, err := json.Marshal(body)
	require.NoError(t, err)
	require.JSONEq(t, `{"longTermMemory":{
		"llm":{"provider":"openai","model":"llm","credentials":{"type":"API_KEY","apiKey":"fake-llm"}},
		"embedding":{"provider":"openai","model":"embedding","credentials":{"type":"API_KEY","apiKey":"fake-embedding"}},
		"classification":{"provider":"openai","model":"classification","credentials":{"type":"API_KEY","apiKey":"fake-classification"}}
	}}`, string(encoded))
	plan = previous
	plan.ClassificationModel = types.StringValue("updated-classification")
	body, d = plan.apiPatch(context.Background(), config, previous)
	require.False(t, d.HasError(), d)
	require.Nil(t, body.LongTermMemory.Classification.Credentials, "model edits must not resend unrotated credentials")
	require.Nil(t, body.LongTermMemory.LLM)
	require.Nil(t, body.LongTermMemory.Embedding)
}
