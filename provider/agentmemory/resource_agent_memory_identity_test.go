package agentmemory

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	rediscloudapi "github.com/RedisLabs/rediscloud-go-api"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

func TestAgentMemoryRejectsMissingOrMismatchedReadIdentity(t *testing.T) {
	for _, returnedID := range []string{"", "unrelated-store"} {
		for _, operation := range []string{"create", "read", "update", "import"} {
			t.Run(operation+"/"+returnedID, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					if r.Method == http.MethodPost || r.Method == http.MethodPatch {
						_, _ = w.Write([]byte(`{"taskId":"task-1"}`))
						return
					}
					if r.URL.Path == "/tasks/task-1" {
						_, _ = w.Write([]byte(`{"status":"processing-completed","response":{"resource":{"storeId":"store-1"}}}`))
						return
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"storeId": returnedID, "databaseId": "123", "name": "identity-store", "status": "READY"})
				}))
				defer server.Close()
				sdk, err := rediscloudapi.NewClient(rediscloudapi.BaseURL(server.URL), rediscloudapi.Auth("fake-key", "fake-secret"))
				if err != nil {
					t.Fatal(err)
				}
				r := &agentMemoryResource{client: &client.ApiClient{Client: sdk}}
				ctx := context.Background()
				var schema resource.SchemaResponse
				r.Schema(ctx, resource.SchemaRequest{}, &schema)
				model := AgentMemoryResourceModel{ID: types.StringValue("store-1"), Name: types.StringValue("identity-store"), DatabaseID: types.Int64Value(123), Endpoints: types.ListNull(agentMemoryEndpointObjectType())}
				state := tfsdk.State{Schema: schema.Schema}
				if diags := state.Set(ctx, &model); diags.HasError() {
					t.Fatal(diags)
				}
				model.ID = types.StringUnknown()
				plan := tfsdk.Plan{Schema: schema.Schema}
				if diags := plan.Set(ctx, &model); diags.HasError() {
					t.Fatal(diags)
				}
				var failed bool
				switch operation {
				case "create":
					response := resource.CreateResponse{State: state}
					r.Create(ctx, resource.CreateRequest{Plan: plan}, &response)
					failed, state = response.Diagnostics.HasError(), response.State
				case "read":
					response := resource.ReadResponse{State: state}
					r.Read(ctx, resource.ReadRequest{State: state}, &response)
					failed, state = response.Diagnostics.HasError(), response.State
				case "update":
					response := resource.UpdateResponse{State: state}
					r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, &response)
					failed, state = response.Diagnostics.HasError(), response.State
				case "import":
					response := resource.ImportStateResponse{State: state}
					r.ImportState(ctx, resource.ImportStateRequest{ID: "store-1"}, &response)
					failed, state = response.Diagnostics.HasError(), response.State
				}
				if !failed {
					t.Error("malformed API identity was silently accepted")
				}
				var id types.String
				if diags := state.GetAttribute(ctx, path.Root("id"), &id); diags.HasError() {
					t.Fatal(diags)
				}
				if id.ValueString() != "store-1" {
					t.Error("malformed API response replaced the tracked resource identity")
				}
			})
		}
	}
}

func TestAgentMemoryAPIKeyMalformedReadPreservesOwnershipAndSecret(t *testing.T) {
	for _, payload := range []string{
		`{"apiKeys":[{"keyName":"runtime"}]}`,
		`{"apiKeys":[{"apiKeyId":"key-1"}]}`,
	} {
		t.Run(payload, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(payload))
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
			state := tfsdk.State{Schema: schema.Schema}
			model := AgentMemoryAPIKeyResourceModel{ID: types.StringValue("key-1"), StoreID: types.StringValue("store-1"), Name: types.StringValue("runtime"), APIKey: types.StringValue("FAKE_EXISTING_STATE_SECRET")}
			if diags := state.Set(ctx, &model); diags.HasError() {
				t.Fatal(diags)
			}
			response := resource.ReadResponse{State: state}
			r.Read(ctx, resource.ReadRequest{State: state}, &response)
			if !response.Diagnostics.HasError() {
				t.Error("incomplete API key metadata did not produce an error")
			}
			var retained AgentMemoryAPIKeyResourceModel
			if diags := response.State.Get(ctx, &retained); diags.HasError() {
				t.Fatal("malformed metadata removed the key state")
			}
			if retained.ID.ValueString() != "key-1" || retained.APIKey.ValueString() != "FAKE_EXISTING_STATE_SECRET" {
				t.Error("malformed metadata lost ownership or the stored secret")
			}
		})
	}
}
