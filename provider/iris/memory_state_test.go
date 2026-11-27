package iris

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func TestFailedMemoryMutationReadPreservesPlannedDetectors(t *testing.T) {
	for _, operation := range []string{"create", "update"} {
		t.Run(operation, func(t *testing.T) {
			ctx := context.Background()
			r := testResource(t, "iris_agent_memory", func(w http.ResponseWriter, req *http.Request) {
				if req.Method == http.MethodGet {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				// A partial mutation response must not replace the configured detectors.
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"storeId": "store-1", "state": "PROVISIONING", "longTermMemoryExclusions": map[string]any{"semantic": map[string]any{"enabled": false}}}))
			})
			detectors, d := types.SetValueFrom(ctx, builtInDetectorElementType, []BuiltInDetectorModel{{ID: types.StringValue("email"), Enabled: types.BoolValue(true)}})
			require.False(t, d.HasError())
			config, plan := objects(t, r, map[string]attr.Value{"name": types.StringValue("after"), "database_id": types.StringValue("db-1"), "built_in_detector": detectors})
			var state tfsdk.State
			if operation == "create" {
				resp := resource.CreateResponse{State: tfsdk.State{Schema: resourceSchema(r)}}
				r.Create(ctx, resource.CreateRequest{Config: config, Plan: plan}, &resp)
				require.True(t, resp.Diagnostics.HasError())
				state = resp.State
			} else {
				var previous AgentMemoryModel
				require.False(t, config.Get(ctx, &previous).HasError())

				previous.ID = types.StringValue("store-1")
				previous.Name = types.StringValue("before")
				planned := previous
				planned.Name = types.StringValue("after")
				resp := resource.UpdateResponse{State: modelState(t, r, &previous)}
				r.Update(ctx, resource.UpdateRequest{Config: config, Plan: tfsdk.Plan(modelState(t, r, &planned)), State: resp.State}, &resp)
				require.True(t, resp.Diagnostics.HasError())
				state = resp.State
			}
			var saved AgentMemoryModel
			require.False(t, state.Get(ctx, &saved).HasError())
			require.Equal(t, "store-1", saved.ID.ValueString())
			require.Equal(t, "after", saved.Name.ValueString())
			require.True(t, saved.BuiltInDetectors.Equal(detectors), saved.BuiltInDetectors)
		})
	}
}

func TestMemoryStateFullReadClearsMissingSettingsAndPreservesLocalState(t *testing.T) {
	ctx := context.Background()
	fields, d := memoryFieldsFromAPI(ctx, []memoryField{{Name: "topic", Type: "STRING", Description: "Topic"}})
	require.False(t, d.HasError(), d)
	inline, d := types.SetValueFrom(ctx, customMemoryTypeElementType, []CustomMemoryTypeModel{{Name: types.StringValue("profile"), Description: types.StringValue("Profile"), Fields: fields}})
	require.False(t, d.HasError(), d)
	detectors, d := types.SetValueFrom(ctx, builtInDetectorElementType, []BuiltInDetectorModel{{ID: types.StringValue("email"), Enabled: types.BoolValue(true)}})
	require.False(t, d.HasError(), d)
	m := AgentMemoryModel{Timeouts: nullTimeouts(),
		ID: types.StringValue("store-1"), Name: types.StringValue("store"), DatabaseID: types.StringValue("db-1"),
		LLMProvider: types.StringValue("openai"), LLMModel: types.StringValue("gpt-4o-mini"), LLMAPIKeyVersion: types.Int64Value(2),
		SummarizationEnabled: types.BoolValue(true), SummarizationEventThreshold: types.Int64Value(100),
		SemanticExclusionsPrompt: types.StringValue("private data"), FailureMessage: types.StringValue("old failure"),
		FlushOnDelete: types.BoolValue(true), CustomMemoryTypes: inline, BuiltInDetectors: detectors,
	}
	status := "READY"
	d = m.setFromAPI(ctx, &agentMemoryResponse{State: &status})
	require.False(t, d.HasError(), d)
	require.Equal(t, types.StringValue("store-1"), m.ID)
	require.Equal(t, types.StringValue("db-1"), m.DatabaseID)
	require.Equal(t, types.StringValue(status), m.State)
	require.True(t, m.LLMProvider.IsNull())
	require.True(t, m.LLMModel.IsNull())
	require.True(t, m.SummarizationEnabled.IsNull())
	require.True(t, m.SummarizationEventThreshold.IsNull())
	require.True(t, m.SemanticExclusionsPrompt.IsNull())
	require.True(t, m.FailureMessage.IsNull())
	require.Empty(t, m.BuiltInDetectors.Elements())
	require.Empty(t, m.CustomDetectors.Elements())
	require.Empty(t, m.Endpoints.Elements())
	require.True(t, inline.Equal(m.CustomMemoryTypes), "GET cannot recover inline create-only types")
	require.Equal(t, types.Int64Value(2), m.LLMAPIKeyVersion)
	require.Equal(t, types.BoolValue(true), m.FlushOnDelete)
}
