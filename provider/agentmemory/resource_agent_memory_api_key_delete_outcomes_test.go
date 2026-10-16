package agentmemory

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"

	rediscloudapi "github.com/RedisLabs/rediscloud-go-api"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

func TestAgentMemoryAPIKeyDeleteHTTPOutcomes(t *testing.T) {
	for _, status := range []int{http.StatusNoContent, http.StatusNotFound, http.StatusForbidden, http.StatusInternalServerError} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				assert.Equal(t, http.MethodDelete, r.Method)
				assert.Equal(t, "/memory-stores/store-1/api-keys/key-1", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				if status != http.StatusNoContent {
					_, _ = w.Write([]byte(`{"message":"parent store missing or unavailable"}`))
				}
			}))
			defer server.Close()
			t.Setenv(rediscloudapi.RedisCloudUrlEnvVar, server.URL)
			t.Setenv(rediscloudapi.AccessKeyEnvVar, "fake-access-key")
			t.Setenv(rediscloudapi.SecretKeyEnvVar, "fake-secret-key")
			c, err := client.NewClient()
			require.NoError(t, err)
			subject := &agentMemoryAPIKeyResource{client: c}
			ctx := context.Background()
			var schema resource.SchemaResponse
			subject.Schema(ctx, resource.SchemaRequest{}, &schema)
			require.False(t, schema.Diagnostics.HasError(), "%v", schema.Diagnostics)
			state := tfsdk.State{Schema: schema.Schema}
			diags := state.Set(ctx, &AgentMemoryAPIKeyResourceModel{
				ID: types.StringValue("key-1"), StoreID: types.StringValue("store-1"), Name: types.StringValue("runtime"),
				APIKey: types.StringValue("fake-runtime-secret"), ObfuscatedToken: types.StringNull(), CreatedAt: types.Int64Null(),
			})
			require.False(t, diags.HasError(), "%v", diags)
			response := resource.DeleteResponse{State: state}
			// Exercise DELETE directly: refresh can otherwise remove a missing key first.
			subject.Delete(ctx, resource.DeleteRequest{State: state}, &response)
			require.EqualValues(t, 1, requests.Load())
			if status == http.StatusForbidden || status == http.StatusInternalServerError {
				require.True(t, response.Diagnostics.HasError(), "HTTP %d must not be treated as successful deletion", status)
				require.Len(t, response.Diagnostics.Errors(), 1)
				diagnostic := response.Diagnostics.Errors()[0]
				assert.Equal(t, "Failed to delete Agent Memory API key", diagnostic.Summary())
				assert.Contains(t, diagnostic.Detail(), strconv.Itoa(status))
				return
			}
			assert.False(t, response.Diagnostics.HasError(), "%v", response.Diagnostics)
		})
	}
}
