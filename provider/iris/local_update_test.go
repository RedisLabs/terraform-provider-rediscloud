package iris

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func TestStoreLocalUpdatesDoNotCallAPI(t *testing.T) {
	ctx := context.Background()
	r := testResource(t, "iris_agent_memory", func(http.ResponseWriter, *http.Request) { t.Error("local update called the API") })
	previous := AgentMemoryModel{Timeouts: nullTimeouts(), Name: types.StringValue("store"), DatabaseID: types.StringValue("db-1"), ID: types.StringValue("store-1"), FlushOnDelete: types.BoolValue(false), Endpoints: types.SetNull(endpointElementType), CustomMemoryTypes: types.SetNull(customMemoryTypeElementType), BuiltInDetectors: types.SetNull(builtInDetectorElementType), CustomDetectors: types.SetNull(customDetectorElementType)}
	for _, flush := range []bool{false, true} {
		config, plan := objects(t, r, map[string]attr.Value{"name": previous.Name, "database_id": previous.DatabaseID, "flush_on_delete": types.BoolValue(flush)})
		state := modelState(t, r, &previous)
		response := resource.UpdateResponse{State: state}
		r.Update(ctx, resource.UpdateRequest{State: state, Config: config, Plan: plan}, &response)
		require.False(t, response.Diagnostics.HasError(), response.Diagnostics)
		var actual types.Bool
		require.False(t, response.State.GetAttribute(ctx, path.Root("flush_on_delete"), &actual).HasError())
		require.Equal(t, flush, actual.ValueBool())
	}
}
func TestStoreImportSetsLocalDefault(t *testing.T) {
	ctx := context.Background()
	r := NewAgentMemoryResource()
	config, _ := objects(t, r, nil)
	response := resource.ImportStateResponse{State: tfsdk.State{Raw: config.Raw, Schema: resourceSchema(r)}}
	r.(resource.ResourceWithImportState).ImportState(ctx, resource.ImportStateRequest{ID: "store-1"}, &response)
	require.False(t, response.Diagnostics.HasError())
	var flush types.Bool
	require.False(t, response.State.GetAttribute(ctx, path.Root("flush_on_delete"), &flush).HasError())
	require.Equal(t, types.BoolValue(false), flush)
}
