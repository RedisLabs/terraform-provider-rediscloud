package agentmemory

import (
	"context"
	"encoding/json"
	"testing"

	api "github.com/RedisLabs/rediscloud-go-api/service/agentmemory"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func TestExclusionListClearingWireContract(t *testing.T) {
	model := &AgentMemoryExclusionsModel{Enabled: types.BoolValue(true), BuiltInDetectors: &AgentMemoryBuiltInDetectorsModel{Enabled: types.BoolValue(true)}, CustomDetectors: &AgentMemoryCustomDetectorsModel{Enabled: types.BoolValue(true)}}
	raw, err := json.Marshal(exclusionsFromModel(model))
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(raw, &payload))
	for _, key := range []string{"builtInDetectors", "customDetectors"} {
		group, ok := payload[key].(map[string]any)
		require.True(t, ok)
		items, present := group["detectors"]
		require.True(t, present, "clearing must send detectors:[]; omission preserves the old list")
		require.Equal(t, []any{}, items)
	}
}

func TestReadPreservesConfiguredInactiveExclusions(t *testing.T) {
	model := AgentMemoryResourceModel{ID: types.StringValue("store-1"), LongTermMemoryExclusions: &AgentMemoryExclusionsModel{Enabled: types.BoolValue(false), Semantic: &AgentMemorySemanticExclusionModel{Enabled: types.BoolValue(false), Prompt: types.StringValue("Keep this instruction")}, BuiltInDetectors: &AgentMemoryBuiltInDetectorsModel{Enabled: types.BoolValue(false), Detectors: []AgentMemoryDetectorSelectionModel{{ID: types.StringValue("email"), Enabled: types.BoolValue(false), Action: types.StringValue("redact")}}}, CustomDetectors: &AgentMemoryCustomDetectorsModel{Enabled: types.BoolValue(false), Detectors: []AgentMemoryCustomDetectorModel{{Name: types.StringValue("ticket"), Enabled: types.BoolValue(false), Action: types.StringValue("redact"), Matcher: &AgentMemoryMatcherSpecModel{Kind: types.StringValue("regex"), Regex: &AgentMemoryRegexSpecModel{Pattern: types.StringValue("ticket+")}}}}}}}
	response := &api.Store{StoreID: "store-1", Name: "fixture", DatabaseID: "123", Status: "READY", LongTermMemoryExclusions: exclusionsFromModel(model.LongTermMemoryExclusions)}
	var diags diag.Diagnostics
	readAgentMemoryIntoModel(context.Background(), response, &model, false, &diags)
	require.False(t, diags.HasError(), diags)
	require.Equal(t, "Keep this instruction", model.LongTermMemoryExclusions.Semantic.Prompt.ValueString())
	require.Len(t, model.LongTermMemoryExclusions.BuiltInDetectors.Detectors, 1)
	require.Len(t, model.LongTermMemoryExclusions.CustomDetectors.Detectors, 1)
}
