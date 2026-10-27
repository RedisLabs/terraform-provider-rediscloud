package agentmemory

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"

	rediscloudapi "github.com/RedisLabs/rediscloud-go-api"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	agentmemoryapi "github.com/RedisLabs/rediscloud-go-api/service/agentmemory"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

func TestAgentMemoryReadResponseIdentity(t *testing.T) {
	for _, operation := range []string{"create", "read", "update", "import"} {
		for _, scenario := range []string{"missing", "mismatched", "matching"} {
			t.Run(operation+"/"+scenario, func(t *testing.T) {
				returnedID := "store-1"
				switch scenario {
				case "missing":
					returnedID = ""
				case "mismatched":
					returnedID = "unrelated-store"
				}
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					switch {
					case r.Method == http.MethodPost && r.URL.Path == "/memory-stores":
						_, _ = w.Write([]byte(`{"taskId":"task-1"}`))
					case r.Method == http.MethodPatch && r.URL.Path == "/memory-stores/store-1":
						_, _ = w.Write([]byte(`{"taskId":"task-1"}`))
					case r.Method == http.MethodGet && r.URL.Path == "/tasks/task-1":
						_, _ = w.Write([]byte(`{"status":"processing-completed","response":{"resource":{"storeId":"store-1"}}}`))
					case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-1":
						_ = json.NewEncoder(w).Encode(map[string]any{"storeId": returnedID, "databaseId": "123", "name": "identity-store", "status": "READY"})
					default:
						t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
						http.NotFound(w, r)
					}
				}))
				defer server.Close()
				sdk, err := rediscloudapi.NewClient(rediscloudapi.BaseURL(server.URL), rediscloudapi.Auth("fake-key", "fake-secret"))
				if err != nil {
					t.Fatal(err)
				}
				ctx := context.Background()
				r := &agentMemoryResource{client: &client.ApiClient{Client: sdk}}
				var schema resource.SchemaResponse
				r.Schema(ctx, resource.SchemaRequest{}, &schema)
				model := AgentMemoryResourceModel{ID: types.StringValue("store-1"), Name: types.StringValue("identity-store"), DatabaseID: types.Int64Value(123), Endpoints: types.ListNull(agentMemoryEndpointObjectType())}
				model.Timeouts = testMemoryTimeoutsList(t, timeouts.Value{Object: types.ObjectNull(schema.Schema.Blocks["timeouts"].Type().(MemoryTimeoutsListType).ElemType.(timeouts.Type).AttrTypes)})
				prior := tfsdk.State{Schema: schema.Schema}
				if diags := prior.Set(ctx, &model); diags.HasError() {
					t.Fatal(diags)
				}
				// Update can receive an unknown computed ID; the request still targets the prior ID.
				model.ID = types.StringUnknown()
				plan := tfsdk.Plan{Schema: schema.Schema}
				if diags := plan.Set(ctx, &model); diags.HasError() {
					t.Fatal(diags)
				}
				output := tfsdk.State{Schema: schema.Schema}
				var diagnostics diag.Diagnostics
				switch operation {
				case "create":
					response := resource.CreateResponse{State: output}
					r.Create(ctx, resource.CreateRequest{Plan: plan}, &response)
					output, diagnostics = response.State, response.Diagnostics
				case "read":
					response := resource.ReadResponse{State: prior}
					r.Read(ctx, resource.ReadRequest{State: prior}, &response)
					output, diagnostics = response.State, response.Diagnostics
				case "update":
					response := resource.UpdateResponse{State: prior}
					r.Update(ctx, resource.UpdateRequest{Plan: plan, State: prior}, &response)
					output, diagnostics = response.State, response.Diagnostics
				case "import":
					response := resource.ImportStateResponse{State: output}
					r.ImportState(ctx, resource.ImportStateRequest{ID: "store-1"}, &response)
					output, diagnostics = response.State, response.Diagnostics
				}
				invalid := scenario != "matching"
				if diagnostics.HasError() != invalid {
					t.Fatalf("unexpected identity validation result: %v", diagnostics)
				}
				if invalid {
					if operation == "read" || operation == "update" {
						if !output.Raw.Equal(prior.Raw) {
							t.Error("invalid response changed prior state")
						}
					} else if operation == "import" && !output.Raw.IsNull() {
						t.Error("invalid response populated imported state")
					} else if operation == "create" && !output.Raw.IsNull() {
						// Create recovery is separate; any saved identity must be the acknowledged one.
						var retained AgentMemoryResourceModel
						if diags := output.Get(ctx, &retained); diags.HasError() {
							t.Fatal(diags)
						}
						if retained.ID.ValueString() != "store-1" {
							t.Error("invalid response replaced acknowledged create ID")
						}
					}
					return
				}
				var retained AgentMemoryResourceModel
				if diags := output.Get(ctx, &retained); diags.HasError() {
					t.Fatal(diags)
				}
				if retained.ID.ValueString() != "store-1" || retained.Name.ValueString() != "identity-store" || retained.DatabaseID.ValueInt64() != 123 {
					t.Error("matching response did not populate the expected resource")
				}
			})
		}
	}
}

func TestAgentMemoryIdentityGuardBeforeModelMutation(t *testing.T) {
	for _, scenario := range []string{"nil", "missing", "mismatched", "matching"} {
		t.Run(scenario, func(t *testing.T) {
			model := AgentMemoryResourceModel{ID: types.StringValue("store-1"), Name: types.StringValue("prior-name"), DatabaseID: types.Int64Value(123)}
			store := &agentmemoryapi.Store{StoreID: redis.String("store-1"), Name: redis.String("fresh-name"), DatabaseID: redis.String("456")}
			switch scenario {
			case "nil":
				store = nil
			case "missing":
				store.StoreID = redis.String("")
			case "mismatched":
				store.StoreID = redis.String("unrelated-store")
			}
			var diagnostics diag.Diagnostics
			// A malformed SDK object must not panic or partially overwrite the owned model.
			defer func() {
				if recover() != nil {
					t.Error("nil API response panicked")
				}
			}()
			readAgentMemoryIntoModel(context.Background(), store, &model, false, &diagnostics)
			if diagnostics.HasError() != (scenario != "matching") {
				t.Fatalf("unexpected identity validation result: %v", diagnostics)
			}
			if model.ID.ValueString() != "store-1" {
				t.Error("response replaced tracked ID")
			}
			if scenario != "matching" && (model.Name.ValueString() != "prior-name" || model.DatabaseID.ValueInt64() != 123) {
				t.Error("invalid identity partially mutated the model")
			}
			if scenario == "matching" && (model.Name.ValueString() != "fresh-name" || model.DatabaseID.ValueInt64() != 456) {
				t.Error("matching response did not refresh the model")
			}
		})
	}
}
