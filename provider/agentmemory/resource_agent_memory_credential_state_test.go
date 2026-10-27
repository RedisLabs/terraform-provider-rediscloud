package agentmemory

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	rediscloudapi "github.com/RedisLabs/rediscloud-go-api"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

func TestAgentMemoryReadClearsLegacyModelCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet || req.URL.Path != "/memory-stores/store-1" {
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			http.NotFound(w, req)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"storeId":"store-1","name":"legacy-store","databaseId":"123","status":"READY","llm":{"provider":"openai","model":"gpt-4o-mini"},"longTermMemory":{"embedding":{"provider":"openai","model":"text-embedding-3-small"}}}`))
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
	model := AgentMemoryResourceModel{ID: types.StringValue("store-1"), Name: types.StringValue("legacy-store"), DatabaseID: types.Int64Value(123), Endpoints: types.ListNull(agentMemoryEndpointObjectType())}
	model.Timeouts = timeouts.Value{Object: types.ObjectNull(schema.Schema.Blocks["timeouts"].Type().(timeouts.Type).AttrTypes)}
	legacyModel := func(name string) *AgentMemoryModelConfigModel {
		return &AgentMemoryModelConfigModel{
			Provider: types.StringValue("openai"), Model: types.StringValue(name),
			Credentials: &AgentMemoryModelCredentialsModel{Type: types.StringValue("apiKey"), APIKey: types.StringValue("FAKE-LEGACY-SECRET"), APIKeyVersion: types.Int64Value(7)},
		}
	}
	model.LLM = legacyModel("gpt-4o-mini")
	model.Embedding = legacyModel("text-embedding-3-small")
	// Reproduce state from the Sensitive development build, including its saved keys.
	prior := tfsdk.State{Schema: schema.Schema}
	if diags := prior.Set(ctx, &model); diags.HasError() {
		t.Fatal(diags)
	}
	response := resource.ReadResponse{State: prior}
	r.Read(ctx, resource.ReadRequest{State: prior}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	var refreshed AgentMemoryResourceModel
	if diags := response.State.Get(ctx, &refreshed); diags.HasError() {
		t.Fatal(diags)
	}
	if !refreshed.ID.Equal(model.ID) || !refreshed.DatabaseID.Equal(model.DatabaseID) {
		t.Fatal("refresh changed store identity")
	}
	for _, block := range []*AgentMemoryModelConfigModel{refreshed.LLM, refreshed.Embedding} {
		if block == nil || block.Credentials == nil || !block.Credentials.APIKey.IsNull() || block.Credentials.Type.ValueString() != "apiKey" || block.Credentials.APIKeyVersion.ValueInt64() != 7 {
			t.Fatal("refresh must clear legacy keys and retain credential metadata")
		}
	}
}
