package agentmemory

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	rediscloudapi "github.com/RedisLabs/rediscloud-go-api"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

func TestAgentMemoryCreateRejectsIncompleteTaskWithoutPanic(t *testing.T) {
	for _, payload := range []string{
		`{"taskId":"task-1","status":"processing-completed"}`,
		`{"taskId":"task-1","status":"processing-completed","response":{}}`,
		`{"taskId":"task-1","status":"processing-completed","response":{"resource":null}}`,
		`{"taskId":"task-1","status":"processing-completed","response":{"resource":{}}}`,
	} {
		t.Run(payload, func(t *testing.T) {
			var creates atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.Method == http.MethodPost && r.URL.Path == "/memory-stores" {
					creates.Add(1)
					_, _ = w.Write([]byte(`{"taskId":"task-1"}`))
					return
				}
				if r.Method == http.MethodGet && r.URL.Path == "/tasks/task-1" {
					_, _ = w.Write([]byte(payload))
					return
				}
				http.NotFound(w, r)
			}))
			defer server.Close()
			t.Setenv(rediscloudapi.RedisCloudUrlEnvVar, server.URL)
			t.Setenv(rediscloudapi.AccessKeyEnvVar, "fake-key")
			t.Setenv(rediscloudapi.SecretKeyEnvVar, "fake-secret")
			c, err := client.NewClient()
			if err != nil {
				t.Fatal(err)
			}
			r := &agentMemoryResource{client: c}
			ctx := context.Background()
			var schema resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schema)
			plan := tfsdk.Plan{Schema: schema.Schema}
			model := AgentMemoryResourceModel{Name: types.StringValue("task-response-test"), DatabaseID: types.Int64Value(123), Endpoints: types.ListNull(agentMemoryEndpointObjectType())}
			if diags := plan.Set(ctx, &model); diags.HasError() {
				t.Fatal(diags)
			}
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Error("incomplete successful task response crashed provider create")
				}
			}()
			response := resource.CreateResponse{State: tfsdk.State{Schema: schema.Schema}}
			r.Create(ctx, resource.CreateRequest{Plan: plan}, &response)
			if !response.Diagnostics.HasError() {
				t.Fatal("expected an error for a completed task without a store ID")
			}
			if creates.Load() != 1 {
				t.Fatalf("unknown create outcome retried POST %d times", creates.Load())
			}
		})
	}
}
