package agentmemory

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelConfigRequestStringPresence(t *testing.T) {
	for _, tc := range []struct {
		name    string
		value   types.String
		present bool
	}{
		{"null", types.StringNull(), false},
		{"unknown", types.StringUnknown(), false},
		{"explicit empty", types.StringValue(""), true},
		{"configured", types.StringValue("configured-value"), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := modelConfigFromModel(&AgentMemoryModelConfigModel{Provider: tc.value, Model: tc.value, Credentials: &AgentMemoryModelCredentialsModel{Type: tc.value}})
			encoded, err := json.Marshal(request)
			require.NoError(t, err)
			var body map[string]any
			require.NoError(t, json.Unmarshal(encoded, &body))
			credentials, ok := body["credentials"].(map[string]any)
			require.True(t, ok)
			credentialType, present := credentials["type"]
			assert.Equal(t, tc.present, present)
			if tc.present {
				assert.Equal(t, tc.value.ValueString(), credentialType)
			}
			for _, field := range []string{"provider", "model"} {
				value, present := body[field]
				assert.Equal(t, tc.present, present)
				if tc.present {
					assert.Equal(t, tc.value.ValueString(), value)
				}
			}
		})
	}
}

func TestCustomMemoryTypeUpdatePromptPresence(t *testing.T) {
	for _, tc := range []struct {
		name    string
		prompt  types.String
		present bool
	}{
		{"null", types.StringNull(), false},
		{"unknown", types.StringUnknown(), false},
		{"explicit empty", types.StringValue(""), true},
		{"configured", types.StringValue("updated prompt"), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := customMemoryTypeModel(t, "preference", nil)
			state.ExtractionStrategy = &AgentMemoryCustomExtractionStrategy{Prompt: types.StringValue("initial prompt"), Enabled: types.BoolValue(true)}
			plan := state
			plan.ExtractionStrategy = &AgentMemoryCustomExtractionStrategy{Prompt: tc.prompt, Enabled: types.BoolValue(false)}
			added, updates := customMemoryTypeUpdatesFromPlan(t.Context(), mustCustomMemoryTypeList(t, []AgentMemoryCustomMemoryTypeModel{plan}), mustCustomMemoryTypeList(t, []AgentMemoryCustomMemoryTypeModel{state}), testAgentMemoryDiags(t))
			require.Empty(t, added)
			require.Len(t, updates, 1)
			encoded, err := json.Marshal(updates[0])
			require.NoError(t, err)
			var body map[string]any
			require.NoError(t, json.Unmarshal(encoded, &body))
			assert.Equal(t, "preference", body["typeName"])
			assert.Equal(t, false, body["enabled"])
			value, present := body["prompt"]
			assert.Equal(t, tc.present, present)
			if tc.present {
				assert.Equal(t, tc.prompt.ValueString(), value)
			}
		})
	}
}
