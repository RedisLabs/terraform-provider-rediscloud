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
				EventCount: &AgentMemoryEventCountModel{
					Threshold:   types.Int64Value(10),
					RetainCount: types.Int64Value(5),
				},
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
			model := summarizationToModel(config, tc.existing, tc.includeAPIOnlyConfig)
			require.NotNil(t, model)
			require.Equal(t, types.BoolValue(true), model.Enabled)
			require.Equal(t, types.StringValue("event_count"), model.TriggerStrategy)
			if !tc.wantEventCount {
				require.Nil(t, model.EventCount)
				return
			}
			require.NotNil(t, model.EventCount)
			require.Equal(t, types.Int64Value(20), model.EventCount.Threshold)
			require.Equal(t, types.Int64Value(10), model.EventCount.RetainCount)
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
		{name: "omitted children", existing: optionalChildExclusionsModel(false, false, false)},
		{name: "semantic only", existing: optionalChildExclusionsModel(true, false, false), wantSemantic: true},
		{name: "built in only", existing: optionalChildExclusionsModel(false, true, false), wantBuiltIn: true},
		{name: "custom only", existing: optionalChildExclusionsModel(false, false, true), wantCustom: true},
		{name: "mixed children", existing: optionalChildExclusionsModel(true, false, true), wantSemantic: true, wantCustom: true},
		{name: "all configured", existing: optionalChildExclusionsModel(true, true, true), wantSemantic: true, wantBuiltIn: true, wantCustom: true},
		{name: "import", includeAPIOnlyConfig: true, wantSemantic: true, wantBuiltIn: true, wantCustom: true},
		{
			name:                 "import adopts omitted siblings",
			existing:             optionalChildExclusionsModel(true, false, false),
			includeAPIOnlyConfig: true,
			wantSemantic:         true,
			wantBuiltIn:          true,
			wantCustom:           true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			model := exclusionsToModel(optionalChildExclusionsAPI(true), tc.existing, tc.includeAPIOnlyConfig)
			require.NotNil(t, model)
			require.Equal(t, types.BoolValue(true), model.Enabled)
			require.Equal(t, tc.wantSemantic, model.Semantic != nil)
			require.Equal(t, tc.wantBuiltIn, model.BuiltInDetectors != nil)
			require.Equal(t, tc.wantCustom, model.CustomDetectors != nil)
			if tc.wantSemantic {
				require.Equal(t, types.StringValue("API prompt"), model.Semantic.Prompt)
			}
			if tc.wantBuiltIn {
				require.Len(t, model.BuiltInDetectors.Detectors, 1)
				require.Equal(t, types.StringValue("email"), model.BuiltInDetectors.Detectors[0].ID)
			}
			if tc.wantCustom {
				require.Len(t, model.CustomDetectors.Detectors, 1)
				detector := model.CustomDetectors.Detectors[0]
				require.Equal(t, types.StringValue("ticket"), detector.Name)
				require.NotNil(t, detector.Matcher)
				require.NotNil(t, detector.Matcher.Regex)
				require.Equal(t, types.StringValue("ticket+"), detector.Matcher.Regex.Pattern)
			}
		})
	}
}

func TestOptionalChildMappingPreservesDisabledDefinitions(t *testing.T) {
	summarization := summarizationToModel(&agentmemoryapi.SummarizationConfig{
		Enabled:         redis.Bool(false),
		TriggerStrategy: redis.String("event_count"),
		EventCount: &agentmemoryapi.SummarizationThresholdConfig{
			Threshold: redis.Int(20), RetainCount: redis.Int(10),
		},
	}, &AgentMemorySummarizationModel{
		Enabled:         types.BoolValue(false),
		TriggerStrategy: types.StringValue("event_count"),
		EventCount: &AgentMemoryEventCountModel{
			Threshold: types.Int64Value(20), RetainCount: types.Int64Value(10),
		},
	}, false)
	require.NotNil(t, summarization.EventCount)
	require.Equal(t, types.StringValue("event_count"), summarization.TriggerStrategy)
	require.Equal(t, types.Int64Value(20), summarization.EventCount.Threshold)
	require.Equal(t, types.Int64Value(10), summarization.EventCount.RetainCount)

	existing := optionalChildExclusionsModel(true, true, true)
	existing.Enabled = types.BoolValue(false)
	existing.Semantic.Enabled = types.BoolValue(false)
	existing.BuiltInDetectors.Enabled = types.BoolValue(false)
	existing.CustomDetectors.Enabled = types.BoolValue(false)
	model := exclusionsToModel(optionalChildExclusionsAPI(false), existing, false)
	require.Equal(t, types.BoolValue(false), model.Enabled)
	require.NotNil(t, model.Semantic)
	require.Equal(t, types.BoolValue(false), model.Semantic.Enabled)
	require.Equal(t, types.StringValue("API prompt"), model.Semantic.Prompt)
	require.NotNil(t, model.BuiltInDetectors)
	require.Equal(t, types.BoolValue(false), model.BuiltInDetectors.Enabled)
	require.Len(t, model.BuiltInDetectors.Detectors, 1)
	require.Equal(t, types.StringValue("email"), model.BuiltInDetectors.Detectors[0].ID)
	require.NotNil(t, model.CustomDetectors)
	require.Equal(t, types.BoolValue(false), model.CustomDetectors.Enabled)
	require.Len(t, model.CustomDetectors.Detectors, 1)
	require.Equal(t, types.StringValue("ticket"), model.CustomDetectors.Detectors[0].Name)
}

func TestOptionalChildMappingDoesNotInventMissingAPIChildren(t *testing.T) {
	for _, importing := range []bool{false, true} {
		name := "managed"
		if importing {
			name = "import"
		}
		t.Run(name, func(t *testing.T) {
			existing := &AgentMemorySummarizationModel{EventCount: &AgentMemoryEventCountModel{Threshold: types.Int64Value(20), RetainCount: types.Int64Value(10)}}
			require.Nil(t, summarizationToModel(nil, existing, importing))
			summarization := summarizationToModel(&agentmemoryapi.SummarizationConfig{Enabled: redis.Bool(true)}, existing, importing)
			require.NotNil(t, summarization)
			require.Nil(t, summarization.EventCount)

			exclusions := optionalChildExclusionsModel(true, true, true)
			require.Nil(t, exclusionsToModel(nil, exclusions, importing))
			model := exclusionsToModel(&agentmemoryapi.LongTermMemoryExclusions{Enabled: redis.Bool(true)}, exclusions, importing)
			require.NotNil(t, model)
			require.Nil(t, model.Semantic)
			require.Nil(t, model.BuiltInDetectors)
			require.Nil(t, model.CustomDetectors)
		})
	}
}

func optionalChildExclusionsModel(semantic, builtIn, custom bool) *AgentMemoryExclusionsModel {
	model := &AgentMemoryExclusionsModel{Enabled: types.BoolValue(true)}
	if semantic {
		model.Semantic = &AgentMemorySemanticExclusionModel{Enabled: types.BoolValue(true), Prompt: types.StringValue("Configured prompt")}
	}
	if builtIn {
		model.BuiltInDetectors = &AgentMemoryBuiltInDetectorsModel{Enabled: types.BoolValue(true), Detectors: []AgentMemoryDetectorSelectionModel{{ID: types.StringValue("email")}}}
	}
	if custom {
		model.CustomDetectors = &AgentMemoryCustomDetectorsModel{Enabled: types.BoolValue(true), Detectors: []AgentMemoryCustomDetectorModel{{Name: types.StringValue("ticket")}}}
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
