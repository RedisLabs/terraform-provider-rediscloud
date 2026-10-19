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

func TestAgentMemoryReadResponsePresence(t *testing.T) {
	for _, tc := range []struct {
		name, payload string
		omitted       bool
	}{
		{name: "omitted", payload: `{"shortMemory":{},"longTermMemory":{},"extractionCadence":{},"longTermMemoryExclusions":{},"endpoints":[{}]}`, omitted: true},
		{name: "explicit-zero", payload: `{"shortMemory":{"ttlSeconds":0},"longTermMemory":{"ttlSeconds":0},"extractionCadence":{"activeIntervalSeconds":0},"longTermMemoryExclusions":{"enabled":false},"endpoints":[{"isAccessible":false}]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var response api.Store
			require.NoError(t, json.Unmarshal([]byte(tc.payload), &response))
			id, name, db := "store-1", "fixture", "123"
			response.StoreID, response.Name, response.DatabaseID = &id, &name, &db
			state := AgentMemoryResourceModel{ID: types.StringValue(id)}
			var diags diag.Diagnostics
			readImportedAgentMemoryIntoModel(context.Background(), &response, &state, &diags)
			require.False(t, diags.HasError(), diags)
			require.Equal(t, tc.omitted, state.ShortTermTTLSeconds.IsNull())
			require.Equal(t, tc.omitted, state.LongTermTTLSeconds.IsNull())
			require.Equal(t, tc.omitted, state.ExtractionCadenceSeconds.IsNull())
			require.Equal(t, tc.omitted, state.LongTermMemoryExclusions.Enabled.IsNull())
			endpoint := state.Endpoints.Elements()[0].(types.Object)
			require.Equal(t, tc.omitted, endpoint.Attributes()["is_accessible"].IsNull())
			if !tc.omitted {
				require.Zero(t, state.ShortTermTTLSeconds.ValueInt64())
				require.False(t, state.LongTermMemoryExclusions.Enabled.ValueBool())
				require.Equal(t, types.BoolValue(false), endpoint.Attributes()["is_accessible"])
			}
		})
	}
}

func TestAgentMemoryReadRejectsMissingName(t *testing.T) {
	for _, payload := range []string{`{"storeId":"store-1","databaseId":"123"}`, `{"storeId":"store-1","databaseId":"123","name":""}`} {
		var response api.Store
		require.NoError(t, json.Unmarshal([]byte(payload), &response))
		original := AgentMemoryResourceModel{ID: types.StringValue("store-1"), Name: types.StringValue("original")}
		state := original
		var diags diag.Diagnostics
		readAgentMemoryIntoModel(context.Background(), &response, &state, false, &diags)
		require.True(t, diags.HasError())
		require.Equal(t, original, state)
	}
}
