package iris

import (
	"context"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func TestIrisOperationTimeoutsCancelRequests(t *testing.T) {
	for _, kind := range []string{"iris_agent_memory", "iris_agent_memory_type", "iris_identity_api_key"} {
		for _, operation := range []string{"create", "read", "update", "delete"} {
			t.Run(kind+"/"+operation, func(t *testing.T) {
				var reached atomic.Bool
				r := testResource(t, kind, func(_ http.ResponseWriter, req *http.Request) {
					reached.Store(true)
					_, _ = io.Copy(io.Discard, req.Body)
					select {
					case <-req.Context().Done():
					case <-time.After(4 * time.Second):
					}
				})
				limits := nullTimeouts()
				values := map[string]attr.Value{}
				for name := range limits.AttributeTypes(context.Background()) {
					values[name] = types.StringNull()
				}
				values[operation] = types.StringValue("500ms")
				raw, d := types.ObjectValue(limits.AttributeTypes(context.Background()), values)
				require.False(t, d.HasError(), d)
				limits.Object = raw
				config, plan := objects(t, r, map[string]attr.Value{"name": types.StringValue("profile"), "database_id": types.StringValue("db-1"), "store_id": types.StringValue("store-1"), "description": types.StringValue("Profile"), "extraction_prompt": types.StringValue("Extract"), "extraction_enabled": types.BoolValue(true), "timeouts": limits})
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				var state tfsdk.State
				switch kind {
				case "iris_agent_memory":
					var m AgentMemoryModel
					require.False(t, config.Get(ctx, &m).HasError())

					m.ID = types.StringValue("store-1")
					m.Name = types.StringValue("before")
					state = modelState(t, r, &m)
				case "iris_agent_memory_type":
					var m AgentMemoryTypeModel
					require.False(t, config.Get(ctx, &m).HasError())

					m.ID = types.StringValue("store-1/profile")
					m.Revision = types.StringValue("1")
					m.ExtractionPrompt = types.StringValue("Before")
					state = modelState(t, r, &m)
				case "iris_identity_api_key":
					var m IdentityAPIKeyModel
					require.False(t, config.Get(ctx, &m).HasError())

					m.ID = types.StringValue("key-1")
					m.Tenant = types.StringValue("tenant-1")
					m.Name = types.StringValue("before")
					state = modelState(t, r, &m)
				}
				started := time.Now()
				switch operation {
				case "create":
					response := resource.CreateResponse{State: tfsdk.State{Schema: resourceSchema(r)}}
					r.Create(ctx, resource.CreateRequest{Config: config, Plan: plan}, &response)
					require.True(t, response.Diagnostics.HasError(), response.Diagnostics)
				case "read":
					response := resource.ReadResponse{State: state}
					r.Read(ctx, resource.ReadRequest{State: state}, &response)
					require.True(t, response.Diagnostics.HasError(), response.Diagnostics)
				case "update":
					response := resource.UpdateResponse{State: state}
					r.Update(ctx, resource.UpdateRequest{Config: config, Plan: plan, State: state}, &response)
					require.True(t, response.Diagnostics.HasError(), response.Diagnostics)
				case "delete":
					response := resource.DeleteResponse{}
					r.Delete(ctx, resource.DeleteRequest{State: state}, &response)
					require.True(t, response.Diagnostics.HasError(), response.Diagnostics)
				}
				require.True(t, reached.Load(), "the operation must attempt an HTTP request")
				require.Less(t, time.Since(started), 2*time.Second, "use the configured timeout rather than the parent context's three-second fallback")
			})
		}
	}
}
