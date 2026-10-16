package agentmemory

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"

	rediscloudapi "github.com/RedisLabs/rediscloud-go-api"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

func TestAgentMemoryAPIKeyReadMetadata(t *testing.T) {
	for _, imported := range []bool{false, true} {
		for _, scenario := range []struct {
			name    string
			payload string
			status  int
			invalid bool
			removed bool
		}{
			{name: "missing-name", payload: `{"apiKeys":[{"apiKeyId":"key-1"}]}`, invalid: true},
			{name: "empty-name", payload: `{"apiKeys":[{"apiKeyId":"key-1","keyName":""}]}`, invalid: true},
			{name: "complete", payload: `{"apiKeys":[{"apiKeyId":"key-1","keyName":"runtime","obfuscatedToken":"masked","createdAt":123}]}`},
			{name: "renamed", payload: `{"apiKeys":[{"apiKeyId":"key-1","keyName":"external-name","obfuscatedToken":"masked","createdAt":123}]}`},
			{name: "optional-metadata-absent", payload: `{"apiKeys":[{"apiKeyId":"key-1","keyName":"runtime"}]}`},
			{name: "key-missing", payload: `{"apiKeys":[]}`, removed: true},
			{name: "other-key", payload: `{"apiKeys":[{"apiKeyId":"key-2","keyName":"other"}]}`, removed: true},
			{name: "store-missing", status: http.StatusNotFound, removed: true},
			{name: "server-error", status: http.StatusInternalServerError, invalid: true},
		} {
			t.Run(fmt.Sprintf("%s/imported-%t", scenario.name, imported), func(t *testing.T) {
				var requests atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					if r.Method != http.MethodGet || r.URL.Path != "/memory-stores/store-1/api-keys" {
						t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					}
					w.Header().Set("Content-Type", "application/json")
					if scenario.status != 0 {
						w.WriteHeader(scenario.status)
						return
					}
					_, _ = w.Write([]byte(scenario.payload))
				}))
				defer server.Close()
				sdk, err := rediscloudapi.NewClient(rediscloudapi.BaseURL(server.URL), rediscloudapi.Auth("fake-key", "fake-secret"))
				if err != nil {
					t.Fatal(err)
				}
				r := &agentMemoryAPIKeyResource{client: &client.ApiClient{Client: sdk}}
				ctx := context.Background()
				var schema resource.SchemaResponse
				r.Schema(ctx, resource.SchemaRequest{}, &schema)
				model := AgentMemoryAPIKeyResourceModel{ID: types.StringValue("key-1"), StoreID: types.StringValue("store-1"), Name: types.StringValue("runtime"), APIKey: types.StringValue("FAKE-EXISTING-SECRET"), ObfuscatedToken: types.StringValue("previous-mask"), CreatedAt: types.Int64Value(42)}
				model.Timeouts = timeouts.Value{Object: types.ObjectNull(schema.Schema.Blocks["timeouts"].Type().(timeouts.Type).AttrTypes)}
				if imported {
					model.Name = types.StringNull()
					model.APIKey = types.StringNull()
					model.ObfuscatedToken = types.StringNull()
					model.CreatedAt = types.Int64Null()
				}
				prior := tfsdk.State{Schema: schema.Schema}
				if diags := prior.Set(ctx, &model); diags.HasError() {
					t.Fatal(diags)
				}
				response := resource.ReadResponse{State: prior}
				r.Read(ctx, resource.ReadRequest{State: prior}, &response)
				if requests.Load() != 1 {
					t.Errorf("expected one metadata read, got %d requests", requests.Load())
				}
				if response.Diagnostics.HasError() != scenario.invalid {
					t.Fatalf("unexpected diagnostic result: %v", response.Diagnostics)
				}
				if scenario.removed {
					if !response.State.Raw.IsNull() {
						t.Error("confirmed missing key remained in state")
					}
					return
				}
				if scenario.invalid {
					if !response.State.Raw.Equal(prior.Raw) {
						t.Error("failed metadata read changed identity, configuration or stored secret")
					}
					return
				}
				var retained AgentMemoryAPIKeyResourceModel
				if diags := response.State.Get(ctx, &retained); diags.HasError() {
					t.Fatal(diags)
				}
				if !retained.ID.Equal(model.ID) || !retained.StoreID.Equal(model.StoreID) || !retained.APIKey.Equal(model.APIKey) {
					t.Error("successful metadata read changed ownership or the write-only secret")
				}
				wantName := "runtime"
				if scenario.name == "renamed" {
					wantName = "external-name"
				}
				if retained.Name.ValueString() != wantName {
					t.Error("valid key name was not refreshed")
				}
			})
		}
	}
}
