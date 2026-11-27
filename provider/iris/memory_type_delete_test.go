package iris

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func TestMemoryTypeDeleteDisablesExtractionWithFreshRevision(t *testing.T) {
	for _, mode := range []string{"enabled", "default-enabled", "disabled", "no-strategy"} {
		t.Run(mode, func(t *testing.T) {
			revision := "live-1"
			methods := []string{}
			disabled := mode == "disabled"
			r := testResource(t, "iris_agent_memory_type", func(w http.ResponseWriter, req *http.Request) {
				methods = append(methods, req.Method)
				if req.Method == http.MethodPatch {
					var body agentMemoryTypeUpdateRequest
					require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
					require.Equal(t, "live-1", body.ExpectedRevision)
					require.NotNil(t, body.ExtractionStrategy)
					require.NotNil(t, body.ExtractionStrategy.Enabled)
					require.False(t, *body.ExtractionStrategy.Enabled)
					require.Nil(t, body.ExtractionStrategy.Prompt)
					disabled = true
					revision = "live-2"
				}
				if req.Method == http.MethodDelete {
					if !disabled && mode != "no-strategy" {
						http.Error(w, "memory type in use", http.StatusConflict)
						return
					}
					require.Equal(t, revision, req.URL.Query().Get("expectedRevision"))
					w.WriteHeader(204)
					return
				}
				current := map[string]any{"name": "profile", "description": "Profile", "revision": revision, "fields": []any{}}
				if mode != "no-strategy" {
					strategy := map[string]any{"prompt": "Extract"}
					if mode != "default-enabled" || disabled {
						strategy["enabled"] = !disabled
					}
					current["extractionStrategy"] = strategy
				}
				require.NoError(t, json.NewEncoder(w).Encode(current))
			})
			previous := AgentMemoryTypeModel{Timeouts: nullTimeouts(), ID: types.StringValue("profile"), Name: types.StringValue("profile"), StoreID: types.StringValue("store-1"), Fields: types.SetNull(memoryFieldElementType), Revision: types.StringValue("stale"), Description: types.StringValue("Profile")}
			response := resource.DeleteResponse{}
			r.Delete(context.Background(), resource.DeleteRequest{State: modelState(t, r, &previous)}, &response)
			require.False(t, response.Diagnostics.HasError(), response.Diagnostics)
			expected := []string{"GET", "DELETE"}
			if mode == "enabled" || mode == "default-enabled" {
				expected = []string{"GET", "PATCH", "GET", "DELETE"}
			}
			require.Equal(t, expected, methods)
		})
	}
}
func TestMemoryTypeDeleteStopsAfterDisableFailure(t *testing.T) {
	deleted := false
	r := testResource(t, "iris_agent_memory_type", func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodGet:
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"name": "profile", "revision": "live", "extractionStrategy": map[string]any{"prompt": "Extract", "enabled": true}}))
		case http.MethodPatch:
			http.Error(w, "revision conflict", http.StatusPreconditionFailed)
		case http.MethodDelete:
			deleted = true
			w.WriteHeader(204)
		}
	})
	previous := AgentMemoryTypeModel{Timeouts: nullTimeouts(), ID: types.StringValue("profile"), Name: types.StringValue("profile"), StoreID: types.StringValue("store-1"), Fields: types.SetNull(memoryFieldElementType), Revision: types.StringValue("stale")}
	response := resource.DeleteResponse{}
	r.Delete(context.Background(), resource.DeleteRequest{State: modelState(t, r, &previous)}, &response)
	require.True(t, response.Diagnostics.HasError())
	require.False(t, deleted)
}
