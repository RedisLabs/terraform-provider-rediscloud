package agentmemory

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"

	rediscloudapi "github.com/RedisLabs/rediscloud-go-api"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

func TestAgentMemoryCreateRecoveryState(t *testing.T) {
	for _, unknownComputed := range []bool{false, true} {
		for _, scenario := range []string{"malformed", "404", "500", "503"} {
			t.Run(fmt.Sprintf("%s-unknown-%t", scenario, unknownComputed), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					switch {
					case r.Method == http.MethodPost && r.URL.Path == "/memory-stores":
						_, _ = w.Write([]byte(`{"taskId":"fixture-task"}`))
					case r.URL.Path == "/tasks/fixture-task":
						_, _ = w.Write([]byte(`{"taskId":"fixture-task","status":"processing-completed","response":{"resource":{"storeId":"store-1"}}}`))
					case r.Method == http.MethodGet && r.URL.Path == "/memory-stores/store-1":
						switch scenario {
						case "malformed":
							_, _ = w.Write([]byte(`{"storeId":"store-1","name":"recovery-store","databaseId":"not-a-number","status":"READY"}`))
						case "404":
							w.WriteHeader(404)
						case "500":
							w.WriteHeader(500)
						case "503":
							w.WriteHeader(503)
						}
					default:
						http.NotFound(w, r)
					}
				}))
				defer server.Close()
				t.Setenv(rediscloudapi.RedisCloudUrlEnvVar, server.URL)
				t.Setenv(rediscloudapi.AccessKeyEnvVar, "FAKE-KEY")
				t.Setenv(rediscloudapi.SecretKeyEnvVar, "FAKE-SECRET")
				c, err := client.NewClient()
				if err != nil {
					t.Fatal(err)
				}
				ctx := context.Background()
				r := &agentMemoryResource{client: c}
				var schema resource.SchemaResponse
				r.Schema(ctx, resource.SchemaRequest{}, &schema)
				model := AgentMemoryResourceModel{ID: types.StringUnknown(), Name: types.StringValue("recovery-store"), DatabaseID: types.Int64Value(123), ShortTermTTLSeconds: types.Int64Value(86400), LongTermTTLSeconds: types.Int64Value(31536000), ExtractionStrategy: types.StringValue("INSTRUCT"), ExtractionCadenceSeconds: types.Int64Value(300), Endpoint: types.StringUnknown(), Endpoints: types.ListUnknown(agentMemoryEndpointObjectType()), Status: types.StringUnknown(), CreatedAt: types.StringUnknown()}
				model.Timeouts = timeouts.Value{Object: types.ObjectNull(schema.Schema.Blocks["timeouts"].Type().(timeouts.Type).AttrTypes)}
				if !unknownComputed {
					model.LLM = &AgentMemoryModelConfigModel{Provider: types.StringValue("fake-provider"), Model: types.StringValue("fake-model"), Credentials: &AgentMemoryModelCredentialsModel{Type: types.StringValue("apiKey"), APIKey: types.StringValue("FAKE-MODEL-SECRET"), APIKeyVersion: types.Int64Value(1)}}
					model.Embedding = &AgentMemoryModelConfigModel{Provider: types.StringValue("fake-provider"), Model: types.StringValue("fake-model"), Credentials: &AgentMemoryModelCredentialsModel{Type: types.StringValue("apiKey"), APIKey: types.StringValue("FAKE-MODEL-SECRET"), APIKeyVersion: types.Int64Value(1)}}
				}
				if unknownComputed {
					model.ShortTermTTLSeconds = types.Int64Unknown()
					model.LongTermTTLSeconds = types.Int64Unknown()
					model.ExtractionStrategy = types.StringUnknown()
					model.ExtractionCadenceSeconds = types.Int64Unknown()
					model.Summarization = &AgentMemorySummarizationModel{Enabled: types.BoolValue(true), TriggerStrategy: types.StringUnknown()}
				}
				plan := tfsdk.Plan{Schema: schema.Schema}
				if diags := plan.Set(ctx, &model); diags.HasError() {
					t.Fatal(diags)
				}
				config := tfsdk.Config{Schema: schema.Schema, Raw: plan.Raw}
				response := resource.CreateResponse{State: tfsdk.State{Schema: schema.Schema}}
				r.Create(ctx, resource.CreateRequest{Plan: plan, Config: config}, &response)
				if response.Diagnostics.HasError() {
					t.Fatalf("unexpected error/warning semantics: %v", response.Diagnostics)
				}
				if len(response.Diagnostics) == 0 {
					t.Fatal("read failure produced no diagnostic")
				}
				if response.State.Raw.IsNull() {
					t.Fatal("acknowledged create lost its state")
				}
				if !response.State.Raw.IsFullyKnown() {
					t.Fatal("recovery state contains unknown values")
				}
				var recovered AgentMemoryResourceModel
				if diags := response.State.Get(ctx, &recovered); diags.HasError() {
					t.Fatal(diags)
				}
				if recovered.ID.ValueString() != "store-1" || recovered.Name.ValueString() != "recovery-store" || recovered.DatabaseID.ValueInt64() != 123 {
					t.Fatal("recovery lost identity or known inputs")
				}
				if unknownComputed {
					if !recovered.ShortTermTTLSeconds.IsNull() || !recovered.LongTermTTLSeconds.IsNull() {
						t.Fatal("unresolved TTLs were not null")
					}
					if recovered.Summarization == nil || !recovered.Summarization.Enabled.ValueBool() || !recovered.Summarization.TriggerStrategy.IsNull() {
						t.Fatal("nested recovery lost known enabled or retained unknown strategy")
					}
				} else if recovered.ShortTermTTLSeconds.ValueInt64() != 86400 || recovered.LongTermTTLSeconds.ValueInt64() != 31536000 {
					t.Fatal("known TTL inputs were not preserved")
				}
				if !unknownComputed && (recovered.LLM == nil || recovered.LLM.Credentials == nil || recovered.LLM.Credentials.APIKey.ValueString() != "FAKE-MODEL-SECRET" || recovered.LLM.Credentials.APIKeyVersion.ValueInt64() != 1 || recovered.Embedding == nil || recovered.Embedding.Credentials == nil || recovered.Embedding.Credentials.APIKey.ValueString() != "FAKE-MODEL-SECRET" || recovered.Embedding.Credentials.APIKeyVersion.ValueInt64() != 1) {
					t.Fatal("configured model credentials or revision were lost during recovery")
				}
			})
		}
	}
}

func TestAgentMemoryCreateRejectedHasNoRecoveryID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(400) }))
	defer server.Close()
	t.Setenv(rediscloudapi.RedisCloudUrlEnvVar, server.URL)
	t.Setenv(rediscloudapi.AccessKeyEnvVar, "FAKE-KEY")
	t.Setenv(rediscloudapi.SecretKeyEnvVar, "FAKE-SECRET")
	c, err := client.NewClient()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	r := &agentMemoryResource{client: c}
	var schema resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schema)
	plan := tfsdk.Plan{Schema: schema.Schema}
	model := AgentMemoryResourceModel{Name: types.StringValue("rejected-store"), DatabaseID: types.Int64Value(123), Endpoints: types.ListNull(agentMemoryEndpointObjectType())}
	model.Timeouts = timeouts.Value{Object: types.ObjectNull(schema.Schema.Blocks["timeouts"].Type().(timeouts.Type).AttrTypes)}
	if diags := plan.Set(ctx, &model); diags.HasError() {
		t.Fatal(diags)
	}
	response := resource.CreateResponse{State: tfsdk.State{Schema: schema.Schema}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, &response)
	if !response.Diagnostics.HasError() || !response.State.Raw.IsNull() {
		t.Fatal("rejected create must report error without inventing state ownership")
	}
}
