package agentmemory

import (
	"context"
	"encoding/json"
	"slices"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
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
			require.Equal(t, tc.omitted, mustExclusions(t, state.LongTermMemoryExclusions).Enabled.IsNull())
			endpoint := state.Endpoints.Elements()[0].(types.Object)
			require.Equal(t, tc.omitted, endpoint.Attributes()["is_accessible"].IsNull())
			if !tc.omitted {
				require.Zero(t, state.ShortTermTTLSeconds.ValueInt64())
				require.False(t, mustExclusions(t, state.LongTermMemoryExclusions).Enabled.ValueBool())
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

func TestAgentMemoryEndpointsNullAndEmpty(t *testing.T) {
	for _, tc := range []struct {
		name, payload string
		nullEndpoints bool
		nullEgressIPs bool
	}{
		{name: "null endpoints", payload: `null`, nullEndpoints: true},
		{name: "empty endpoints", payload: `[]`},
		{name: "omitted IPs", payload: `[{}]`, nullEgressIPs: true},
		{name: "null IPs", payload: `[{"egressIps":null}]`, nullEgressIPs: true},
		{name: "empty IPs", payload: `[{"egressIps":[]}]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var endpoints []api.Endpoint
			require.NoError(t, json.Unmarshal([]byte(tc.payload), &endpoints))
			var diags diag.Diagnostics
			got := endpointsFromAPI(context.Background(), endpoints, &diags)
			require.False(t, diags.HasError(), diags)
			require.Equal(t, tc.nullEndpoints, got.IsNull())
			if tc.nullEndpoints {
				return
			}
			if len(endpoints) == 0 {
				require.Empty(t, got.Elements())
				return
			}
			require.Len(t, got.Elements(), 1)
			attrs := got.Elements()[0].(types.Object).Attributes()
			ips := attrs["egress_ips"].(types.List)
			require.Equal(t, tc.nullEgressIPs, ips.IsNull())
			require.Empty(t, ips.Elements())
		})
	}
}

func TestAgentMemoryReadEndpointOrder(t *testing.T) {
	first := api.Endpoint{Provider: redis.String("AWS"), Region: redis.String("us-east-1"), URL: redis.String("https://a.example.invalid"), IsAccessible: redis.Bool(false)}
	second := api.Endpoint{Provider: redis.String("AWS"), Region: redis.String("us-east-1"), URL: redis.String("https://b.example.invalid")}
	third := api.Endpoint{Provider: redis.String("AWS"), Region: redis.String("us-east-2"), URL: redis.String("https://c.example.invalid")}
	fourth := api.Endpoint{Provider: redis.String("GCP"), Region: redis.String("us-east4"), URL: redis.String("https://d.example.invalid"), IsAccessible: redis.Bool(true)}

	for _, tc := range []struct {
		name      string
		primary   *string
		endpoints []api.Endpoint
		want      types.String
	}{
		{name: "reversed response", endpoints: []api.Endpoint{fourth, third, second, first}, want: types.StringValue(*first.URL)},
		{name: "permuted response", endpoints: []api.Endpoint{second, first, fourth, third}, want: types.StringValue(*first.URL)},
		{name: "empty primary uses fallback", primary: redis.String(""), endpoints: []api.Endpoint{third, fourth, first, second}, want: types.StringValue(*first.URL)},
		{name: "explicit primary wins", primary: fourth.URL, endpoints: []api.Endpoint{second, third, first, fourth}, want: types.StringValue(*fourth.URL)},
		{name: "nil endpoints", want: types.StringNull()},
		{name: "empty endpoints", endpoints: []api.Endpoint{}, want: types.StringNull()},
		{name: "explicit primary without regional endpoints", primary: fourth.URL, want: types.StringValue(*fourth.URL)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			original := slices.Clone(tc.endpoints)
			store := &api.Store{StoreID: redis.String("store-1"), Name: redis.String("fixture"), DatabaseID: redis.String("123"), Endpoint: tc.primary, Endpoints: tc.endpoints}
			state := AgentMemoryResourceModel{ID: types.StringValue("store-1")}
			var diags diag.Diagnostics
			readImportedAgentMemoryIntoModel(context.Background(), store, &state, &diags)
			require.False(t, diags.HasError(), diags)
			require.Equal(t, tc.want, state.Endpoint)
			require.Equal(t, original, store.Endpoints, "API response must not be changed")
			if len(tc.endpoints) == 0 {
				require.Equal(t, tc.endpoints == nil, state.Endpoints.IsNull())
				require.Empty(t, state.Endpoints.Elements())
				return
			}
			var models []AgentMemoryEndpointModel
			require.False(t, state.Endpoints.ElementsAs(context.Background(), &models, false).HasError())
			var urls []string
			for _, model := range models {
				urls = append(urls, model.URL.ValueString())
			}
			require.Equal(t, []string{*first.URL, *second.URL, *third.URL, *fourth.URL}, urls)
		})
	}
}
