package iris

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func TestRemovingLastDetectorsPreservesConfiguredToggle(t *testing.T) {
	for _, configuredToggle := range []bool{false, true} {
		t.Run(map[bool]string{false: "explicit false", true: "unconfigured"}[configuredToggle], func(t *testing.T) {
			ctx := context.Background()
			enabled := true
			action := "DROP"
			id := "st_11111111111111111111111111111111"
			name := "store"
			ready := "READY"
			previous := AgentMemoryModel{Timeouts: nullTimeouts()}
			require.False(t, previous.setFromAPI(ctx, &agentMemoryResponse{StoreID: &id, State: &ready, agentMemorySettings: agentMemorySettings{Name: &name, LongTermMemoryExclusions: &memoryExclusions{Enabled: &enabled, BuiltInDetectors: &memoryBuiltInDetectors{Enabled: &enabled, Detectors: []memoryBuiltInDetector{{ID: "email", Enabled: true, Action: &action}}}, CustomDetectors: &memoryCustomDetectors{Enabled: &enabled, Detectors: []memoryCustomDetector{{Name: "private", Enabled: true, Action: &action, Matcher: memoryMatcher{Kind: "REGEX", Regex: memoryRegex{Pattern: "private[0-9]+"}}}}}}}}).HasError())
			current := map[string]any{}
			r := testResource(t, "iris_agent_memory", func(w http.ResponseWriter, req *http.Request) {
				if req.Method == http.MethodPatch {
					require.NoError(t, json.NewDecoder(req.Body).Decode(&current))
					groups := current["longTermMemoryExclusions"].(map[string]any)
					for _, name := range []string{"builtInDetectors", "customDetectors"} {
						group := groups[name].(map[string]any)
						require.Equal(t, configuredToggle, group["enabled"])
						require.Empty(t, group["detectors"])
					}
				}
				current["storeId"] = id
				current["name"] = name
				current["state"] = ready
				require.NoError(t, json.NewEncoder(w).Encode(current))
			})
			values := map[string]attr.Value{"id": types.StringValue(id), "name": types.StringValue(name), "database_id": types.StringValue("db-1"), "exclusions_enabled": types.BoolValue(true)}
			if !configuredToggle {
				values["built_in_detectors_enabled"] = types.BoolValue(false)
				values["custom_detectors_enabled"] = types.BoolValue(false)
			}
			config, plan := objects(t, r, values)
			state := modelState(t, r, &previous)
			response := resource.UpdateResponse{State: state}
			r.Update(ctx, resource.UpdateRequest{Config: config, Plan: plan, State: state}, &response)
			require.False(t, response.Diagnostics.HasError(), response.Diagnostics)
		})
	}
}
