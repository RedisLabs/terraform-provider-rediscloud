package agentmemory

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	rediscloudapi "github.com/RedisLabs/rediscloud-go-api"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

func TestAgentMemoryDatabaseIDDiagnostic(t *testing.T) {
	for _, operation := range []string{"create", "read", "update", "import"} {
		for _, scenario := range []struct {
			name       string
			databaseID string
			secret     string
		}{
			{name: "credential", databaseID: "FAKE-CREDENTIAL-DO-NOT-ECHO", secret: "FAKE-CREDENTIAL-DO-NOT-ECHO"},
			{name: "url", databaseID: "https://user:FAKE-PASSWORD-DO-NOT-ECHO@example.test", secret: "FAKE-PASSWORD-DO-NOT-ECHO"},
			{name: "json", databaseID: `{"apiKey":"FAKE-API-KEY-DO-NOT-ECHO"}`, secret: "FAKE-API-KEY-DO-NOT-ECHO"},
			{name: "multiline", databaseID: "invalid\nFAKE-TOKEN-DO-NOT-ECHO", secret: "FAKE-TOKEN-DO-NOT-ECHO"},
			{name: "valid", databaseID: "123"},
		} {
			t.Run(operation+"/"+scenario.name, func(t *testing.T) {
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
						_ = json.NewEncoder(w).Encode(map[string]any{"storeId": "store-1", "databaseId": scenario.databaseID, "name": "diagnostic-store", "status": "READY"})
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
				model := AgentMemoryResourceModel{ID: types.StringValue("store-1"), Name: types.StringValue("diagnostic-store"), DatabaseID: types.Int64Value(123), Endpoints: types.ListNull(agentMemoryEndpointObjectType())}
				prior := tfsdk.State{Schema: schema.Schema}
				if diags := prior.Set(ctx, &model); diags.HasError() {
					t.Fatal(diags)
				}
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
				invalid := scenario.name != "valid"
				if diagnostics.HasError() != invalid {
					t.Fatalf("unexpected database ID validation result: error=%t", diagnostics.HasError())
				}
				if invalid {
					found := false
					for _, diagnostic := range diagnostics {
						if diagnostic.Summary() == "Unexpected Agent Memory database ID" {
							found = true
							if !strings.Contains(diagnostic.Detail(), "numeric") {
								t.Error("diagnostic no longer explains the expected database ID format")
							}
						}
						if strings.Contains(diagnostic.Summary(), scenario.secret) || strings.Contains(diagnostic.Detail(), scenario.secret) {
							t.Error("diagnostic exposed sensitive content from the invalid API response")
						}
					}
					if !found {
						t.Error("missing database ID diagnostic")
					}
					return
				}
				var retained AgentMemoryResourceModel
				if diags := output.Get(ctx, &retained); diags.HasError() {
					t.Fatal(diags)
				}
				if retained.ID.ValueString() != "store-1" || retained.Name.ValueString() != "diagnostic-store" || retained.DatabaseID.ValueInt64() != 123 {
					t.Error("valid response did not populate the expected resource")
				}
			})
		}
	}
}
