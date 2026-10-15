package agentmemory

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	rediscloudapi "github.com/RedisLabs/rediscloud-go-api"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

type deadlineTransport func(*http.Request) (*http.Response, error)

func (f deadlineTransport) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestAgentMemoryOperationsAlwaysBoundNetworkWork(t *testing.T) {
	for _, operation := range []string{"create", "read", "update", "delete", "import", "key-create", "key-read", "key-delete"} {
		t.Run(operation, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			}))
			defer server.Close()
			transport := deadlineTransport(func(req *http.Request) (*http.Response, error) {
				requests.Add(1)
				deadline, ok := req.Context().Deadline()
				if !ok || time.Until(deadline) <= 0 || time.Until(deadline) > time.Hour {
					t.Error("Agent Memory network operation has no bounded execution deadline")
				}
				return server.Client().Transport.RoundTrip(req)
			})
			sdk, err := rediscloudapi.NewClient(rediscloudapi.BaseURL(server.URL), rediscloudapi.Auth("fake-key", "fake-secret"), rediscloudapi.Transporter(transport))
			if err != nil {
				t.Fatal(err)
			}
			c := &client.ApiClient{Client: sdk}
			r, key := &agentMemoryResource{client: c}, &agentMemoryAPIKeyResource{client: c}
			ctx := context.Background()
			var schema, keySchema resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schema)
			key.Schema(ctx, resource.SchemaRequest{}, &keySchema)
			model := AgentMemoryResourceModel{ID: types.StringValue("store-1"), Name: types.StringValue("bounded-store"), DatabaseID: types.Int64Value(123), Endpoints: types.ListNull(agentMemoryEndpointObjectType())}
			plan, state := tfsdk.Plan{Schema: schema.Schema}, tfsdk.State{Schema: schema.Schema}
			if diags := plan.Set(ctx, &model); diags.HasError() {
				t.Fatal(diags)
			}
			if diags := state.Set(ctx, &model); diags.HasError() {
				t.Fatal(diags)
			}
			keyModel := AgentMemoryAPIKeyResourceModel{ID: types.StringValue("key-1"), StoreID: types.StringValue("store-1"), Name: types.StringValue("runtime")}
			keyPlan, keyState := tfsdk.Plan{Schema: keySchema.Schema}, tfsdk.State{Schema: keySchema.Schema}
			if diags := keyPlan.Set(ctx, &keyModel); diags.HasError() {
				t.Fatal(diags)
			}
			if diags := keyState.Set(ctx, &keyModel); diags.HasError() {
				t.Fatal(diags)
			}
			switch operation {
			case "create":
				response := resource.CreateResponse{State: state}
				r.Create(ctx, resource.CreateRequest{Plan: plan}, &response)
			case "read":
				response := resource.ReadResponse{State: state}
				r.Read(ctx, resource.ReadRequest{State: state}, &response)
			case "update":
				response := resource.UpdateResponse{State: state}
				r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, &response)
			case "delete":
				response := resource.DeleteResponse{State: state}
				r.Delete(ctx, resource.DeleteRequest{State: state}, &response)
			case "import":
				response := resource.ImportStateResponse{State: state}
				r.ImportState(ctx, resource.ImportStateRequest{ID: "store-1"}, &response)
			case "key-create":
				response := resource.CreateResponse{State: keyState}
				key.Create(ctx, resource.CreateRequest{Plan: keyPlan}, &response)
			case "key-read":
				response := resource.ReadResponse{State: keyState}
				key.Read(ctx, resource.ReadRequest{State: keyState}, &response)
			case "key-delete":
				response := resource.DeleteResponse{State: keyState}
				key.Delete(ctx, resource.DeleteRequest{State: keyState}, &response)
			}
			if requests.Load() == 0 {
				t.Fatal("test did not exercise a network operation")
			}
		})
	}
}

func TestAgentMemoryCreateCancellationDoesNotRetryOrLoseAcknowledgedID(t *testing.T) {
	for _, acknowledged := range []bool{false, true} {
		t.Run(map[bool]string{false: "pending-task", true: "follow-up-read"}[acknowledged], func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			var posts atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.Method + " " + r.URL.Path {
				case "POST /memory-stores":
					posts.Add(1)
					_, _ = w.Write([]byte(`{"taskId":"task-1"}`))
				case "GET /tasks/task-1":
					if acknowledged {
						_, _ = w.Write([]byte(`{"status":"processing-completed","response":{"resource":{"storeId":"store-1"}}}`))
					} else {
						cancel()
						_, _ = w.Write([]byte(`{"status":"processing-in-progress"}`))
					}
				case "GET /memory-stores/store-1":
					cancel()
					<-r.Context().Done()
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			sdk, err := rediscloudapi.NewClient(rediscloudapi.BaseURL(server.URL), rediscloudapi.Auth("fake-key", "fake-secret"))
			if err != nil {
				t.Fatal(err)
			}
			r := &agentMemoryResource{client: &client.ApiClient{Client: sdk}}
			var schema resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schema)
			plan := tfsdk.Plan{Schema: schema.Schema}
			model := AgentMemoryResourceModel{Name: types.StringValue("cancelled-store"), DatabaseID: types.Int64Value(123), Endpoints: types.ListNull(agentMemoryEndpointObjectType())}
			if diags := plan.Set(ctx, &model); diags.HasError() {
				t.Fatal(diags)
			}
			response := resource.CreateResponse{State: tfsdk.State{Schema: schema.Schema}}
			r.Create(ctx, resource.CreateRequest{Plan: plan}, &response)
			if posts.Load() != 1 {
				t.Fatalf("cancelled create repeated POST: %d", posts.Load())
			}
			if acknowledged {
				if response.Diagnostics.HasError() {
					t.Fatal(response.Diagnostics)
				}
				var id types.String
				if diags := response.State.GetAttribute(context.Background(), path.Root("id"), &id); diags.HasError() {
					t.Fatal(diags)
				}
				if id.ValueString() != "store-1" {
					t.Fatal("acknowledged store ID lost after cancellation")
				}
			} else if !response.Diagnostics.HasError() {
				t.Fatal("expected a cancellation diagnostic for pending task")
			}
		})
	}
}
