package agentmemory

import (
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	api "github.com/RedisLabs/rediscloud-go-api/service/agentmemory"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentMemoryStringMappingPreservesNullAndEmpty(t *testing.T) {
	for _, tc := range []struct {
		name     string
		value    *string
		expected types.String
	}{
		{"null", nil, types.StringNull()},
		{"empty", redis.String(""), types.StringValue("")},
		{"configured", redis.String("configured"), types.StringValue("configured")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			var diags diag.Diagnostics
			summarization := summarizationToModel(ctx, &api.SummarizationConfig{Enabled: redis.Bool(true), TriggerStrategy: tc.value}, nil, true, &diags)
			selections := detectorSelectionsToModel([]api.DetectorSelection{{Action: tc.value}})
			detectors := customDetectorsToModel(ctx, []api.CustomDetector{{Name: tc.value, Action: tc.value, Matcher: &api.MatcherSpec{Kind: tc.value, Regex: &api.RegexSpec{Pattern: tc.value}}}}, &diags)
			matcher := matcherSpecBlock(ctx, detectors[0].Matcher, &diags)
			regex := regexSpecBlock(ctx, matcher.Regex, &diags)
			memoryTypes := customMemoryTypesToModel(ctx, []api.CustomMemoryType{{Name: tc.value, Description: tc.value, Fields: []api.CustomField{{Name: tc.value, Description: tc.value, Type: tc.value}}, ExtractionStrategy: &api.CustomExtractionStrategy{Prompt: tc.value}}}, nil, &diags)
			fields := customFieldModels(ctx, memoryTypes[0].Fields, &diags)
			strategy := customExtractionStrategyBlock(ctx, memoryTypes[0].ExtractionStrategy, &diags)
			require.False(t, diags.HasError(), diags.Errors())
			for name, value := range map[string]types.String{
				"trigger strategy":  summarization.TriggerStrategy,
				"built-in action":   selections[0].Action,
				"detector name":     detectors[0].Name,
				"custom action":     detectors[0].Action,
				"matcher kind":      matcher.Kind,
				"regex pattern":     regex.Pattern,
				"type name":         memoryTypes[0].Name,
				"type description":  memoryTypes[0].Description,
				"field name":        fields[0].Name,
				"field description": fields[0].Description,
				"field type":        fields[0].Type,
				"prompt":            strategy.Prompt,
			} {
				assert.True(t, tc.expected.Equal(value), "%s: got %s, expected %s", name, value, tc.expected)
			}
		})
	}
}
