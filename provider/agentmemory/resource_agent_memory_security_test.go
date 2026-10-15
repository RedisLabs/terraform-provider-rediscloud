package agentmemory

import (
	"context"
	"fmt"
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

func TestAgentMemoryDiagnosticsNeverEchoAPISecrets(t *testing.T) {
	const secret = "FAKE_UNSTRUCTURED_BACKEND_SECRET_7b6953d2"
	cases := []string{"create", "create-read-warning", "create-malformed-read", "read", "update", "update-read", "update-malformed-read", "delete", "import", "key-create", "key-read", "key-delete"}
	for _, scenario := range cases {
		t.Run(scenario, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if (scenario == "create-read-warning" || scenario == "create-malformed-read") && r.Method == "POST" {
					_, _ = w.Write([]byte(`{"taskId":"operation-task"}`))
					return
				}
				if (scenario == "update-read" || scenario == "update-malformed-read") && r.Method == "PATCH" {
					_, _ = w.Write([]byte(`{"taskId":"operation-task"}`))
					return
				}
				if r.URL.Path == "/tasks/operation-task" {
					_, _ = w.Write([]byte(`{"taskId":"operation-task","status":"processing-completed","response":{"resource":{"storeId":"store-1"}}}`))
					return
				}
				if scenario == "create-malformed-read" || scenario == "update-malformed-read" {
					_, _ = fmt.Fprintf(w, `{"storeId":"store-1","name":"safe-store","databaseId":%q}`, secret)
					return
				}
				if scenario == "key-create" && r.Method == "GET" {
					_, _ = w.Write([]byte(`{"apiKeys":[]}`))
					return
				}
				w.WriteHeader(500)
				_, _ = fmt.Fprintf(w, `{"message":%q}`, secret)
			}))
			defer server.Close()
			t.Setenv(rediscloudapi.RedisCloudUrlEnvVar, server.URL)
			t.Setenv(rediscloudapi.AccessKeyEnvVar, "test-api-key")
			t.Setenv(rediscloudapi.SecretKeyEnvVar, secret)
			c, err := client.NewClient()
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			r := &agentMemoryResource{client: c}
			key := &agentMemoryAPIKeyResource{client: c}
			var storeSchema, keySchema resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &storeSchema)
			key.Schema(ctx, resource.SchemaRequest{}, &keySchema)
			model := AgentMemoryResourceModel{ID: types.StringValue("store-1"), Name: types.StringValue("safe-store"), DatabaseID: types.Int64Value(123), ShortTermTTLSeconds: types.Int64Value(86400), LongTermTTLSeconds: types.Int64Value(31536000), ExtractionStrategy: types.StringValue("INSTRUCT"), ExtractionCadenceSeconds: types.Int64Value(300), Endpoint: types.StringNull(), Endpoints: types.ListNull(agentMemoryEndpointObjectType()), Status: types.StringValue("READY"), CreatedAt: types.StringNull()}
			storeState := tfsdk.State{Schema: storeSchema.Schema}
			if d := storeState.Set(ctx, &model); d.HasError() {
				t.Fatal(d)
			}
			storePlan := tfsdk.Plan{Schema: storeSchema.Schema}
			if d := storePlan.Set(ctx, &model); d.HasError() {
				t.Fatal(d)
			}
			keyModel := AgentMemoryAPIKeyResourceModel{ID: types.StringValue("key-1"), StoreID: types.StringValue("store-1"), Name: types.StringValue("runtime"), APIKey: types.StringValue(secret), ObfuscatedToken: types.StringNull(), CreatedAt: types.Int64Null()}
			keyState := tfsdk.State{Schema: keySchema.Schema}
			if d := keyState.Set(ctx, &keyModel); d.HasError() {
				t.Fatal(d)
			}
			keyPlan := tfsdk.Plan{Schema: keySchema.Schema}
			if d := keyPlan.Set(ctx, &keyModel); d.HasError() {
				t.Fatal(d)
			}
			var diagnostics diag.Diagnostics
			switch scenario {
			case "create", "create-read-warning", "create-malformed-read":
				response := resource.CreateResponse{State: tfsdk.State{Schema: storeSchema.Schema}}
				r.Create(ctx, resource.CreateRequest{Plan: storePlan}, &response)
				diagnostics = response.Diagnostics
			case "read":
				response := resource.ReadResponse{State: storeState}
				r.Read(ctx, resource.ReadRequest{State: storeState}, &response)
				diagnostics = response.Diagnostics
			case "update", "update-read", "update-malformed-read":
				response := resource.UpdateResponse{State: storeState}
				r.Update(ctx, resource.UpdateRequest{State: storeState, Plan: storePlan}, &response)
				diagnostics = response.Diagnostics
			case "delete":
				response := resource.DeleteResponse{State: storeState}
				r.Delete(ctx, resource.DeleteRequest{State: storeState}, &response)
				diagnostics = response.Diagnostics
			case "import":
				response := resource.ImportStateResponse{State: tfsdk.State{Schema: storeSchema.Schema}}
				r.ImportState(ctx, resource.ImportStateRequest{ID: "store-1"}, &response)
				diagnostics = response.Diagnostics
			case "key-create":
				response := resource.CreateResponse{State: tfsdk.State{Schema: keySchema.Schema}}
				key.Create(ctx, resource.CreateRequest{Plan: keyPlan}, &response)
				diagnostics = response.Diagnostics
			case "key-read":
				response := resource.ReadResponse{State: keyState}
				key.Read(ctx, resource.ReadRequest{State: keyState}, &response)
				diagnostics = response.Diagnostics
			case "key-delete":
				response := resource.DeleteResponse{State: keyState}
				key.Delete(ctx, resource.DeleteRequest{State: keyState}, &response)
				diagnostics = response.Diagnostics
			}
			if len(diagnostics) == 0 {
				t.Fatal("injected failure produced no diagnostic")
			}
			if scenario == "create-read-warning" {
				if diagnostics.HasError() {
					t.Fatal("successful create lost recoverable warning semantics")
				}
			} else if !diagnostics.HasError() {
				t.Fatal("injected failure was hidden")
			}
			for _, d := range diagnostics {
				if strings.Contains(d.Summary()+d.Detail(), secret) {
					t.Fatal("backend credential leaked in diagnostic")
				}
			}
		})
	}
}
