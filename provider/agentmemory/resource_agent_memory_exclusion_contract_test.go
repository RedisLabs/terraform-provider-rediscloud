package agentmemory

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	api "github.com/RedisLabs/rediscloud-go-api/service/agentmemory"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func TestExclusionListClearingWireContract(t *testing.T) {
	model := &AgentMemoryExclusionsModel{Enabled: types.BoolValue(true), BuiltInDetectors: mustBuiltInDetectorsList(t, &AgentMemoryBuiltInDetectorsModel{Enabled: types.BoolValue(true)}), CustomDetectors: mustCustomDetectorsList(t, &AgentMemoryCustomDetectorsModel{Enabled: types.BoolValue(true)})}
	raw, err := json.Marshal(exclusionsFromModel(t.Context(), model, testAgentMemoryDiags(t)))
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
	model := AgentMemoryResourceModel{ID: types.StringValue("store-1"), LongTermMemoryExclusions: mustExclusionsList(t, &AgentMemoryExclusionsModel{Enabled: types.BoolValue(false), Semantic: mustSemanticExclusionList(t, &AgentMemorySemanticExclusionModel{Enabled: types.BoolValue(false), Prompt: types.StringValue("Keep this instruction")}), BuiltInDetectors: mustBuiltInDetectorsList(t, &AgentMemoryBuiltInDetectorsModel{Enabled: types.BoolValue(false), Detectors: mustDetectorSelectionList(t, []AgentMemoryDetectorSelectionModel{{ID: types.StringValue("email"), Enabled: types.BoolValue(false), Action: types.StringValue("redact")}})}), CustomDetectors: mustCustomDetectorsList(t, &AgentMemoryCustomDetectorsModel{Enabled: types.BoolValue(false), Detectors: mustCustomDetectorList(t, []AgentMemoryCustomDetectorModel{{Name: types.StringValue("ticket"), Enabled: types.BoolValue(false), Action: types.StringValue("redact"), Matcher: mustMatcherSpecList(t, &AgentMemoryMatcherSpecModel{Kind: types.StringValue("regex"), Regex: mustRegexSpecList(t, &AgentMemoryRegexSpecModel{Pattern: types.StringValue("ticket+")})})}})})})}
	response := &api.Store{StoreID: redis.String("store-1"), Name: redis.String("fixture"), DatabaseID: redis.String("123"), Status: redis.String("READY"), LongTermMemoryExclusions: exclusionsFromModel(t.Context(), mustExclusions(t, model.LongTermMemoryExclusions), testAgentMemoryDiags(t))}
	var diags diag.Diagnostics
	readAgentMemoryIntoModel(context.Background(), response, &model, false, &diags)
	require.False(t, diags.HasError(), diags)
	require.Equal(t, "Keep this instruction", mustSemanticExclusion(t, mustExclusions(t, model.LongTermMemoryExclusions).Semantic).Prompt.ValueString())
	require.Len(t, mustBuiltInDetectors(t, mustExclusions(t, model.LongTermMemoryExclusions).BuiltInDetectors).Detectors.Elements(), 1)
	require.Len(t, mustCustomDetectors(t, mustExclusions(t, model.LongTermMemoryExclusions).CustomDetectors).Detectors.Elements(), 1)
}
