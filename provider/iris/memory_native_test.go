package iris

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func TestMemoryCollectionSchemaTypes(t *testing.T) {
	for _, tc := range []struct {
		element types.ObjectType
		block   schema.SetNestedBlock
	}{
		{memoryFieldElementType, memoryFieldBlock()},
		{customMemoryTypeElementType, customMemoryTypeBlock()},
		{builtInDetectorElementType, builtInDetectorBlock()},
		{customDetectorElementType, customDetectorBlock()},
	} {
		require.True(t, tc.block.Type().Equal(types.SetType{ElemType: tc.element}))
		require.True(t, tc.element.Equal(tc.block.NestedObject.Type()))
	}
	nested := customMemoryTypeBlock().NestedObject.Blocks["field"].(schema.SetNestedBlock)
	require.True(t, memoryFieldElementType.Equal(nested.NestedObject.Type()))
}

func TestMemoryTypesRoundTripAndOrder(t *testing.T) {
	ctx := context.Background()
	prompt := "Extract preferences"
	enabled := true
	input := []memoryCustomType{{Name: "profile", Description: "Profile", Fields: []memoryField{{Name: "topic", Description: "Topic", Type: "STRING"}, {Name: "score", Description: "Score", Type: "FLOAT"}}, ExtractionStrategy: &memoryExtraction{Prompt: &prompt, Enabled: &enabled}}}
	fieldsValue, d := memoryFieldsFromAPI(ctx, input[0].Fields)
	require.False(t, d.HasError())
	raw, d := types.SetValueFrom(ctx, customMemoryTypeElementType, []CustomMemoryTypeModel{{Name: types.StringValue("profile"), Description: types.StringValue("Profile"), Fields: fieldsValue, ExtractionPrompt: types.StringValue(prompt), ExtractionEnabled: types.BoolValue(enabled)}})
	v := raw
	require.False(t, d.HasError())
	var models []CustomMemoryTypeModel
	d = v.ElementsAs(ctx, &models, false)
	require.False(t, d.HasError())
	require.Len(t, models, 1)
	var fields []MemoryFieldModel
	d = models[0].Fields.ElementsAs(ctx, &fields, false)
	require.False(t, d.HasError())
	require.Len(t, fields, 2)
	output, d := memoryTypesToAPI(ctx, v)
	require.False(t, d.HasError())
	require.ElementsMatch(t, input[0].Fields, output[0].Fields)
	require.Equal(t, input[0].ExtractionStrategy, output[0].ExtractionStrategy)
	input[0].Fields[0], input[0].Fields[1] = input[0].Fields[1], input[0].Fields[0]
	reorderedFields, d := memoryFieldsFromAPI(ctx, input[0].Fields)
	require.False(t, d.HasError())
	models[0].Fields = reorderedFields
	raw, d = types.SetValueFrom(ctx, customMemoryTypeElementType, models)
	reordered := raw
	require.False(t, d.HasError())
	require.True(t, v.Equal(reordered))

}

func TestNativeStoreWriteOnlyCredentials(t *testing.T) {
	ctx := context.Background()
	m := AgentMemoryModel{Timeouts: nullTimeouts(), LLMProvider: types.StringValue("openai"), LLMModel: types.StringValue("gpt-4o-mini"), LLMAPIKey: types.StringValue("fake-shared-key"), EmbeddingProvider: types.StringValue("openai"), EmbeddingModel: types.StringValue("text-embedding-3-small"), EmbeddingAPIKey: types.StringValue("fake-shared-key"), ShortMemoryTTLSeconds: types.Int64Value(86400), LLMAPIKeyVersion: types.Int64Value(1), EmbeddingAPIKeyVersion: types.Int64Value(1)}
	body, d := m.apiPatch(ctx, m, AgentMemoryModel{Timeouts: nullTimeouts()})
	require.False(t, d.HasError())
	payload, err := json.Marshal(body)
	require.NoError(t, err)
	var parsed struct {
		LongTermMemory longTermMemoryConfig `json:"longTermMemory"`
	}
	require.NoError(t, json.Unmarshal(payload, &parsed))
	for _, model := range []*modelConfig{parsed.LongTermMemory.LLM, parsed.LongTermMemory.Embedding} {
		require.NotNil(t, model.Credentials)
		require.Equal(t, "API_KEY", model.Credentials.Type)
		require.Equal(t, "fake-shared-key", model.Credentials.APIKey)
	}
	ttl := int64(42)
	state := "READY"
	d = m.setFromAPI(ctx, &agentMemoryResponse{State: &state, agentMemorySettings: agentMemorySettings{ShortMemory: &memoryTTL{TTLSeconds: &ttl}, LongTermMemory: &longTermMemoryConfig{LLM: &modelConfig{Provider: optionalString(m.LLMProvider), Model: optionalString(m.LLMModel)}, Embedding: &modelConfig{Provider: optionalString(m.EmbeddingProvider), Model: optionalString(m.EmbeddingModel)}}}})
	require.False(t, d.HasError())
	require.Equal(t, int64(42), m.ShortMemoryTTLSeconds.ValueInt64())

	schema := AgentMemoryResourceSchema(ctx)
	require.True(t, schema.Attributes["llm_api_key"].IsSensitive())
	require.True(t, schema.Attributes["llm_api_key"].IsWriteOnly())
	require.True(t, schema.Attributes["embedding_api_key"].IsSensitive())
	for name := range schema.Attributes {
		require.NotContains(t, name, "_json")
	}
}

func TestDetectorAndEndpointMapping(t *testing.T) {
	ctx := context.Background()
	action := "REDACT"
	enabled := true
	m := AgentMemoryModel{Timeouts: nullTimeouts()}
	current := &agentMemoryResponse{agentMemorySettings: agentMemorySettings{LongTermMemoryExclusions: &memoryExclusions{Enabled: &enabled, BuiltInDetectors: &memoryBuiltInDetectors{Enabled: &enabled, Detectors: []memoryBuiltInDetector{{ID: "email", Enabled: true, Action: &action}}}, CustomDetectors: &memoryCustomDetectors{Enabled: &enabled, Detectors: []memoryCustomDetector{{Name: "secret", Enabled: true, Action: &action, Matcher: memoryMatcher{Kind: "REGEX", Regex: memoryRegex{Pattern: "secret[0-9]+"}}}}}}}, Endpoints: []memoryEndpoint{{URL: "https://memory.example.com", Provider: "AWS", Region: "eu-west-1", EgressIPs: []string{"127.0.0.2", "127.0.0.1"}, Accessible: true}}}
	d := m.setFromAPI(ctx, current)
	require.False(t, d.HasError())
	var builtin []BuiltInDetectorModel
	d = m.BuiltInDetectors.ElementsAs(ctx, &builtin, false)
	require.False(t, d.HasError())
	require.Equal(t, "email", builtin[0].ID.ValueString())
	var custom []CustomDetectorModel
	d = m.CustomDetectors.ElementsAs(ctx, &custom, false)
	require.False(t, d.HasError())
	require.Equal(t, "secret[0-9]+", custom[0].Pattern.ValueString())
	var endpoints []EndpointModel
	require.False(t, m.Endpoints.ElementsAs(ctx, &endpoints, false).HasError())
	require.Len(t, endpoints, 1)
	require.True(t, endpoints[0].Accessible.ValueBool())
	body, d := m.apiPatch(ctx, m, AgentMemoryModel{Timeouts: nullTimeouts()})
	require.False(t, d.HasError())
	require.Equal(t, current.LongTermMemoryExclusions, body.LongTermMemoryExclusions)
	previous := m.Endpoints
	current.Endpoints[0].EgressIPs = []string{"127.0.0.1", "127.0.0.2"}
	d = m.setFromAPI(ctx, current)
	require.False(t, d.HasError())
	require.True(t, previous.Equal(m.Endpoints))
	require.True(t, m.Endpoints.Type(ctx).Equal(types.SetType{ElemType: endpointElementType}))
}
