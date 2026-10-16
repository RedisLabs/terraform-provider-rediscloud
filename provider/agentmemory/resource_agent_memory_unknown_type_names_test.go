package agentmemory

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func TestCustomTypeUnknownNameValidation(t *testing.T) {
	existing := AgentMemoryCustomMemoryTypeModel{Name: types.StringValue("preference"), Description: types.StringValue("Preferences"), ExtractionStrategy: &AgentMemoryCustomExtractionStrategy{Prompt: types.StringValue("Extract preferences"), Enabled: types.BoolValue(true)}}
	unknown := existing
	unknown.Name = types.StringUnknown()
	renamed := existing
	renamed.Name = types.StringValue("renamed")
	redefined := existing
	redefined.Description = types.StringValue("Changed definition")
	withoutStrategy := existing
	withoutStrategy.ExtractionStrategy = nil
	for _, c := range []struct {
		name         string
		planned      []AgentMemoryCustomMemoryTypeModel
		errorSummary string
	}{
		{name: "unchanged", planned: []AgentMemoryCustomMemoryTypeModel{existing}},
		{name: "unknown-existing-name", planned: []AgentMemoryCustomMemoryTypeModel{unknown}},
		{name: "known-existing-and-unknown-addition", planned: []AgentMemoryCustomMemoryTypeModel{existing, unknown}},
		{name: "all-removed", errorSummary: "Unsupported Agent Memory custom memory type removal"},
		{name: "known-rename", planned: []AgentMemoryCustomMemoryTypeModel{renamed}, errorSummary: "Unsupported Agent Memory custom memory type removal"},
		{name: "redefinition-with-unknown-addition", planned: []AgentMemoryCustomMemoryTypeModel{redefined, unknown}, errorSummary: "Unsupported Agent Memory custom memory type redefinition"},
		{name: "strategy-removal-with-unknown-addition", planned: []AgentMemoryCustomMemoryTypeModel{withoutStrategy, unknown}, errorSummary: "Unsupported Agent Memory custom memory type strategy removal"},
		{name: "duplicate-with-unknown-addition", planned: []AgentMemoryCustomMemoryTypeModel{existing, existing, unknown}, errorSummary: "Duplicate Agent Memory custom memory type"},
	} {
		t.Run(c.name, func(t *testing.T) {
			var resp resource.ModifyPlanResponse
			validateCustomMemoryTypesPlan(AgentMemoryResourceModel{CustomMemoryTypes: c.planned}, AgentMemoryResourceModel{CustomMemoryTypes: []AgentMemoryCustomMemoryTypeModel{existing}}, &resp)
			if c.errorSummary == "" {
				require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics.Errors())
				return
			}
			require.True(t, resp.Diagnostics.HasError())
			require.Equal(t, c.errorSummary, resp.Diagnostics.Errors()[0].Summary())
		})
	}
}
