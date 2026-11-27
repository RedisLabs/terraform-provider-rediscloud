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

func TestMemoryTypeParentComesFromRequestPath(t *testing.T) {
	for _, extra := range []bool{false, true} {
		r := testResource(t, "iris_agent_memory_type", func(w http.ResponseWriter, req *http.Request) {
			require.Equal(t, "/iris/v1/agent-memory/stores/parent/memory-types/profile", req.URL.Path)
			body := map[string]any{"name": "profile", "description": "Profile", "revision": "1", "fields": []any{}}
			if extra {
				body["storeId"] = "future-extra-field"
			}
			require.NoError(t, json.NewEncoder(w).Encode(body))
		})
		config, _ := objects(t, r, map[string]attr.Value{"name": types.StringValue("profile"), "store_id": types.StringValue("parent")})
		state := tfsdk.State(config)
		resp := resource.ReadResponse{State: state}
		r.Read(context.Background(), resource.ReadRequest{State: state}, &resp)
		require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
		var got AgentMemoryTypeModel
		require.False(t, resp.State.Get(context.Background(), &got).HasError())
		require.Equal(t, "parent", got.StoreID.ValueString())
		require.Equal(t, "parent/profile", got.ID.ValueString())
	}
}
