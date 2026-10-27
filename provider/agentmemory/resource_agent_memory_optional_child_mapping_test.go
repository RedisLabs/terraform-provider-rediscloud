package agentmemory

import (
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	agentmemoryapi "github.com/RedisLabs/rediscloud-go-api/service/agentmemory"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func TestSummarizationOptionalChildOwnership(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		existing             *AgentMemorySummarizationModel
		includeAPIOnlyConfig bool
		wantEventCount       bool
	}{
		{name: "absent parent"},
		{name: "omitted event count", existing: &AgentMemorySummarizationModel{Enabled: types.BoolValue(true)}},
		{
			name: "configured event count",
			existing: &AgentMemorySummarizationModel{
				Enabled: types.BoolValue(true),
				EventCount: mustEventCountList(t, &AgentMemoryEventCountModel{
					Threshold:   types.Int64Value(10),
					RetainCount: types.Int64Value(5),
				}),
			},
			wantEventCount: true,
		},
		{name: "import", includeAPIOnlyConfig: true, wantEventCount: true},
		{
			name:                 "import with existing parent",
			existing:             &AgentMemorySummarizationModel{Enabled: types.BoolValue(true)},
			includeAPIOnlyConfig: true,
			wantEventCount:       true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Enabled policies also receive API defaults for omitted optional children.
			config := &agentmemoryapi.SummarizationConfig{
				Enabled:         redis.Bool(true),
				TriggerStrategy: redis.String("event_count"),
				EventCount: &agentmemoryapi.SummarizationThresholdConfig{
					Threshold:   redis.Int(20),
					RetainCount: redis.Int(10),
				},
			}
			model := summarizationToModel(t.Context(), config, tc.existing, tc.includeAPIOnlyConfig, testAgentMemoryDiags(t))
			require.NotNil(t, model)
			require.Equal(t, types.BoolValue(true), model.Enabled)
			require.Equal(t, types.StringValue("event_count"), model.TriggerStrategy)
			if !tc.wantEventCount {
				require.Nil(t, mustEventCount(t, model.EventCount))
				return
			}
			require.NotNil(t, mustEventCount(t, model.EventCount))
			require.Equal(t, types.Int64Value(20), mustEventCount(t, model.EventCount).Threshold)
			require.Equal(t, types.Int64Value(10), mustEventCount(t, model.EventCount).RetainCount)
		})
	}
}

func TestExclusionOptionalChildOwnership(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		existing             *AgentMemoryExclusionsModel
		includeAPIOnlyConfig bool
		wantSemantic         bool
		wantBuiltIn          bool
		wantCustom           bool
	}{
		{name: "absent parent"},
		{name: "omitted children", existing: optionalChildExclusionsModel(t, false, false, false)},
		{name: "semantic only", existing: optionalChildExclusionsModel(t, true, false, false), wantSemantic: true},
		{name: "built in only", existing: optionalChildExclusionsModel(t, false, true, false), wantBuiltIn: true},
		{name: "custom only", existing: optionalChildExclusionsModel(t, false, false, true), wantCustom: true},
		{name: "mixed children", existing: optionalChildExclusionsModel(t, true, false, true), wantSemantic: true, wantCustom: true},
		{name: "all configured", existing: optionalChildExclusionsModel(t, true, true, true), wantSemantic: true, wantBuiltIn: true, wantCustom: true},
		{name: "import", includeAPIOnlyConfig: true, wantSemantic: true, wantBuiltIn: true, wantCustom: true},
		{
			name:                 "import adopts omitted siblings",
			existing:             optionalChildExclusionsModel(t, true, false, false),
			includeAPIOnlyConfig: true,
			wantSemantic:         true,
			wantBuiltIn:          true,
			wantCustom:           true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			model := exclusionsToModel(t.Context(), optionalChildExclusionsAPI(true), tc.existing, tc.includeAPIOnlyConfig, testAgentMemoryDiags(t))
			require.NotNil(t, model)
			require.Equal(t, types.BoolValue(true), model.Enabled)
			require.Equal(t, tc.wantSemantic, mustSemanticExclusion(t, model.Semantic) != nil)
			require.Equal(t, tc.wantBuiltIn, mustBuiltInDetectors(t, model.BuiltInDetectors) != nil)
			require.Equal(t, tc.wantCustom, mustCustomDetectors(t, model.CustomDetectors) != nil)
			if tc.wantSemantic {
				require.Equal(t, types.StringValue("API prompt"), mustSemanticExclusion(t, model.Semantic).Prompt)
			}
			if tc.wantBuiltIn {
				require.Len(t, mustBuiltInDetectors(t, model.BuiltInDetectors).Detectors.Elements(), 1)
				require.Equal(t, types.StringValue("email"), mustDetectorSelectionModels(t, mustBuiltInDetectors(t, model.BuiltInDetectors).Detectors)[0].ID)
			}
			if tc.wantCustom {
				require.Len(t, mustCustomDetectors(t, model.CustomDetectors).Detectors.Elements(), 1)
				detector := mustCustomDetectorModels(t, mustCustomDetectors(t, model.CustomDetectors).Detectors)[0]
				require.Equal(t, types.StringValue("ticket"), detector.Name)
				require.NotNil(t, mustMatcherSpec(t, detector.Matcher))
				require.NotNil(t, mustRegexSpec(t, mustMatcherSpec(t, detector.Matcher).Regex))
				require.Equal(t, types.StringValue("ticket+"), mustRegexSpec(t, mustMatcherSpec(t, detector.Matcher).Regex).Pattern)
			}
		})
	}
}

func TestOptionalChildMappingPreservesDisabledDefinitions(t *testing.T) {
	summarization := summarizationToModel(t.Context(), &agentmemoryapi.SummarizationConfig{
		Enabled:         redis.Bool(false),
		TriggerStrategy: redis.String("event_count"),
		EventCount: &agentmemoryapi.SummarizationThresholdConfig{
			Threshold: redis.Int(20), RetainCount: redis.Int(10),
		},
	}, &AgentMemorySummarizationModel{
		Enabled:         types.BoolValue(false),
		TriggerStrategy: types.StringValue("event_count"),
		EventCount: mustEventCountList(t, &AgentMemoryEventCountModel{
			Threshold: types.Int64Value(20), RetainCount: types.Int64Value(10),
		}),
	}, false, testAgentMemoryDiags(t))
	require.NotNil(t, mustEventCount(t, summarization.EventCount))
	require.Equal(t, types.StringValue("event_count"), summarization.TriggerStrategy)
	require.Equal(t, types.Int64Value(20), mustEventCount(t, summarization.EventCount).Threshold)
	require.Equal(t, types.Int64Value(10), mustEventCount(t, summarization.EventCount).RetainCount)

	existing := optionalChildExclusionsModel(t, true, true, true)
	existing.Enabled = types.BoolValue(false)
	semanticExclusion := mustSemanticExclusion(t, existing.Semantic)
	semanticExclusion.Enabled = types.BoolValue(false)
	existing.Semantic = mustSemanticExclusionList(t, semanticExclusion)
	builtInDetectors := mustBuiltInDetectors(t, existing.BuiltInDetectors)
	builtInDetectors.Enabled = types.BoolValue(false)
	existing.BuiltInDetectors = mustBuiltInDetectorsList(t, builtInDetectors)
	customDetectors := mustCustomDetectors(t, existing.CustomDetectors)
	customDetectors.Enabled = types.BoolValue(false)
	existing.CustomDetectors = mustCustomDetectorsList(t, customDetectors)
	model := exclusionsToModel(t.Context(), optionalChildExclusionsAPI(false), existing, false, testAgentMemoryDiags(t))
	require.Equal(t, types.BoolValue(false), model.Enabled)
	require.NotNil(t, mustSemanticExclusion(t, model.Semantic))
	require.Equal(t, types.BoolValue(false), mustSemanticExclusion(t, model.Semantic).Enabled)
	require.Equal(t, types.StringValue("API prompt"), mustSemanticExclusion(t, model.Semantic).Prompt)
	require.NotNil(t, mustBuiltInDetectors(t, model.BuiltInDetectors))
	require.Equal(t, types.BoolValue(false), mustBuiltInDetectors(t, model.BuiltInDetectors).Enabled)
	require.Len(t, mustBuiltInDetectors(t, model.BuiltInDetectors).Detectors.Elements(), 1)
	require.Equal(t, types.StringValue("email"), mustDetectorSelectionModels(t, mustBuiltInDetectors(t, model.BuiltInDetectors).Detectors)[0].ID)
	require.NotNil(t, mustCustomDetectors(t, model.CustomDetectors))
	require.Equal(t, types.BoolValue(false), mustCustomDetectors(t, model.CustomDetectors).Enabled)
	require.Len(t, mustCustomDetectors(t, model.CustomDetectors).Detectors.Elements(), 1)
	require.Equal(t, types.StringValue("ticket"), mustCustomDetectorModels(t, mustCustomDetectors(t, model.CustomDetectors).Detectors)[0].Name)
}

func TestOptionalChildMappingDoesNotInventMissingAPIChildren(t *testing.T) {
	for _, importing := range []bool{false, true} {
		name := "managed"
		if importing {
			name = "import"
		}
		t.Run(name, func(t *testing.T) {
			existing := &AgentMemorySummarizationModel{EventCount: mustEventCountList(t, &AgentMemoryEventCountModel{Threshold: types.Int64Value(20), RetainCount: types.Int64Value(10)})}
			require.Nil(t, summarizationToModel(t.Context(), nil, existing, importing, testAgentMemoryDiags(t)))
			summarization := summarizationToModel(t.Context(), &agentmemoryapi.SummarizationConfig{Enabled: redis.Bool(true)}, existing, importing, testAgentMemoryDiags(t))
			require.NotNil(t, summarization)
			require.Nil(t, mustEventCount(t, summarization.EventCount))

			exclusions := optionalChildExclusionsModel(t, true, true, true)
			require.Nil(t, exclusionsToModel(t.Context(), nil, exclusions, importing, testAgentMemoryDiags(t)))
			model := exclusionsToModel(t.Context(), &agentmemoryapi.LongTermMemoryExclusions{Enabled: redis.Bool(true)}, exclusions, importing, testAgentMemoryDiags(t))
			require.NotNil(t, model)
			require.Nil(t, mustSemanticExclusion(t, model.Semantic))
			require.Nil(t, mustBuiltInDetectors(t, model.BuiltInDetectors))
			require.Nil(t, mustCustomDetectors(t, model.CustomDetectors))
		})
	}
}

func optionalChildExclusionsModel(t *testing.T, semantic, builtIn, custom bool) *AgentMemoryExclusionsModel {
	model := &AgentMemoryExclusionsModel{Enabled: types.BoolValue(true)}
	if semantic {
		model.Semantic = mustSemanticExclusionList(t, &AgentMemorySemanticExclusionModel{Enabled: types.BoolValue(true), Prompt: types.StringValue("Configured prompt")})
	}
	if builtIn {
		model.BuiltInDetectors = mustBuiltInDetectorsList(t, &AgentMemoryBuiltInDetectorsModel{Enabled: types.BoolValue(true), Detectors: mustDetectorSelectionList(t, []AgentMemoryDetectorSelectionModel{{ID: types.StringValue("email")}})})
	}
	if custom {
		model.CustomDetectors = mustCustomDetectorsList(t, &AgentMemoryCustomDetectorsModel{Enabled: types.BoolValue(true), Detectors: mustCustomDetectorList(t, []AgentMemoryCustomDetectorModel{{Name: types.StringValue("ticket")}})})
	}
	return model
}

func optionalChildExclusionsAPI(enabled bool) *agentmemoryapi.LongTermMemoryExclusions {
	return &agentmemoryapi.LongTermMemoryExclusions{
		Enabled:  redis.Bool(enabled),
		Semantic: &agentmemoryapi.SemanticExclusion{Enabled: redis.Bool(enabled), Prompt: redis.String("API prompt")},
		BuiltInDetectors: &agentmemoryapi.BuiltInDetectorsConfig{
			Enabled: redis.Bool(enabled),
			Detectors: []agentmemoryapi.DetectorSelection{{
				ID: redis.String("email"), Enabled: redis.Bool(false), Action: redis.String("redact"),
			}},
		},
		CustomDetectors: &agentmemoryapi.CustomDetectorsConfig{
			Enabled: redis.Bool(enabled),
			Detectors: []agentmemoryapi.CustomDetector{{
				Name: redis.String("ticket"), Enabled: redis.Bool(false), Action: redis.String("redact"),
				Matcher: &agentmemoryapi.MatcherSpec{Kind: redis.String("regex"), Regex: &agentmemoryapi.RegexSpec{Pattern: redis.String("ticket+")}},
			}},
		},
	}
}
