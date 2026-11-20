package iris

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
	store "github.com/RedisLabs/terraform-provider-rediscloud/provider/iris/generated/resource_iris_agent_memory"
	extractor "github.com/RedisLabs/terraform-provider-rediscloud/provider/iris/generated/resource_iris_agent_memory_extractor"
	memorytype "github.com/RedisLabs/terraform-provider-rediscloud/provider/iris/generated/resource_iris_agent_memory_type"
	key "github.com/RedisLabs/terraform-provider-rediscloud/provider/iris/generated/resource_iris_identity_api_key"
	idp "github.com/RedisLabs/terraform-provider-rediscloud/provider/iris/generated/resource_iris_identity_provider"
	grants "github.com/RedisLabs/terraform-provider-rediscloud/provider/iris/generated/resource_iris_identity_subject_grant_set"
)

func TestGeneratedSchemasMatchModels(t *testing.T) {
	models := map[string]any{"iris_agent_memory": store.IrisAgentMemoryModel{}, "iris_agent_memory_extractor": extractor.IrisAgentMemoryExtractorModel{}, "iris_agent_memory_type": memorytype.IrisAgentMemoryTypeModel{}, "iris_identity_api_key": key.IrisIdentityApiKeyModel{}, "iris_identity_provider": idp.IrisIdentityProviderModel{RequiredAlgorithms: types.ListNull(types.StringType)}, "iris_identity_subject_grant_set": grants.IrisIdentitySubjectGrantSetModel{}}
	for _, d := range definitions {
		t.Run(d.Name, func(t *testing.T) {
			r := &serviceResource{definition: d}
			require.True(t, r.schema(context.Background()).Type().Equal(types.ObjectType{AttrTypes: customtypes.AttrTypesOf(models[d.Name])}))
		})
	}
}
func TestJSONProjectionPreservesSecretsAndExactNumbers(t *testing.T) {
	template, err := decodeJSON(`{"llm":{"model":"test","credentials":{"apiKey":"secret"}},"ttlSeconds":9007199254740993}`)
	require.NoError(t, err)
	current, err := decodeJSON(`{"llm":{"model":"test"},"ttlSeconds":9007199254740993,"serverDefault":true}`)
	require.NoError(t, err)
	projected := projectJSON(template, current, map[string]any{"properties": map[string]any{"llm": map[string]any{"properties": map[string]any{"model": map[string]any{"type": "string"}}}}})
	require.Equal(t, template, projected)
	value, err := toWire(types.StringValue(`{"zero":0,"false":false,"large":9007199254740993}`), true)
	require.NoError(t, err)
	require.Equal(t, json.Number("9007199254740993"), value.(map[string]any)["large"])
	require.True(t, equalJSON(`{"a":1,"b":false}`, "{\n\"b\":false,\"a\":1}"))
	_, err = decodeJSON(`{} {}`)
	require.Error(t, err)
}
func TestJSONRequestValidation(t *testing.T) {
	var shape map[string]any
	for _, d := range definitions {
		if d.Name == "iris_identity_api_key" {
			for _, f := range d.Fields {
				if f.Wire == "grants" {
					shape = f.Shape
				}
			}
		}
	}
	for _, tc := range []struct {
		value string
		valid bool
	}{
		{`[{"product":"memory","resourceType":"store","resourceId":"store-1","actions":["read"],"constraints":{"schema":"example","nested":{"zero":0,"enabled":false}}}]`, true},
		{`[]`, false},
		{`[{"product":"memory","actions":["read"]}]`, false},
		{`[{"product":"bad-product","resourceType":"store","resourceId":"store-1","actions":["read"]}]`, false},
	} {
		v, err := decodeJSON(tc.value)
		require.NoError(t, err)
		err = validateShape(v, shape, "grants")
		require.Equal(t, tc.valid, err == nil)
	}
	for _, s := range []string{".", "..", "/path", "back\\slash", "trailing ", "\x00"} {
		require.Error(t, validatePathComponent(s, true))
	}
	require.NoError(t, validatePathComponent("tenant-1", true))
}
func testResource(t *testing.T, name string, handler http.HandlerFunc) *serviceResource {
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	connector, err := client.NewIrisServiceClient(client.IrisServiceConfig{URL: server.URL, Token: "test-token"})
	require.NoError(t, err)
	for _, d := range definitions {
		if d.Name == name {
			return &serviceResource{definition: d, client: connector, pollInterval: time.Millisecond}
		}
	}
	t.Fatal("unknown resource")
	return nil
}
func objects(t *testing.T, r *serviceResource, configured map[string]attr.Value) (tfsdk.Config, tfsdk.Plan) {
	ctx := context.Background()
	s := r.schema(ctx)
	typ := s.Type().(types.ObjectType)
	configValues := map[string]attr.Value{}
	planValues := map[string]attr.Value{}
	for name, a := range s.Attributes {
		value := configured[name]
		if value == nil {
			value = nullValue(ctx, a.GetType())
		}
		configValues[name] = value
		planValues[name] = value
		if a.IsComputed() && value.IsNull() {
			planValues[name], _ = a.GetType().ValueFromTerraform(ctx, tftypes.NewValue(a.GetType().TerraformType(ctx), tftypes.UnknownValue))
		}
	}
	configObject, d := types.ObjectValue(typ.AttrTypes, configValues)
	require.False(t, d.HasError())
	planObject, d := types.ObjectValue(typ.AttrTypes, planValues)
	require.False(t, d.HasError())
	configRaw, e := configObject.ToTerraformValue(ctx)
	require.NoError(t, e)
	planRaw, e := planObject.ToTerraformValue(ctx)
	require.NoError(t, e)
	return tfsdk.Config{Raw: configRaw, Schema: s}, tfsdk.Plan{Raw: planRaw, Schema: s}
}
func TestCreateReadFailureRetainsOwnership(t *testing.T) {
	r := testResource(t, "iris_identity_api_key", func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPost {
			_ = json.NewEncoder(w).Encode(map[string]any{"keyId": "key-1", "token": "creation-secret", "createdAt": 123})
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	configuration, plan := objects(t, r, map[string]attr.Value{"name": types.StringValue("key")})
	resp := resource.CreateResponse{State: tfsdk.State{Schema: r.schema(context.Background())}}
	r.Create(context.Background(), resource.CreateRequest{Config: configuration, Plan: plan}, &resp)
	require.True(t, resp.Diagnostics.HasError())
	var id, token types.String
	require.False(t, resp.State.GetAttribute(context.Background(), path.Root("id"), &id).HasError())
	require.Equal(t, "key-1", id.ValueString())
	require.False(t, resp.State.GetAttribute(context.Background(), path.Root("token"), &token).HasError())
	require.Equal(t, "creation-secret", token.ValueString())
}
func TestReadAndDeleteNotFound(t *testing.T) {
	r := testResource(t, "iris_identity_api_key", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) })
	_, plan := objects(t, r, map[string]attr.Value{"id": types.StringValue("key-1"), "key_id": types.StringValue("key-1"), "name": types.StringValue("key")})
	state := tfsdk.State(plan)
	read := resource.ReadResponse{State: state}
	r.Read(context.Background(), resource.ReadRequest{State: state}, &read)
	require.False(t, read.Diagnostics.HasError())
	require.True(t, read.State.Raw.IsNull())
	deleted := resource.DeleteResponse{}
	r.Delete(context.Background(), resource.DeleteRequest{State: state}, &deleted)
	require.False(t, deleted.Diagnostics.HasError())
}

func TestFailedStoreReadinessRetainsLatestState(t *testing.T) {
	r := testResource(t, "iris_agent_memory", func(w http.ResponseWriter, req *http.Request) {
		state := "UNAVAILABLE"
		if req.Method == http.MethodPost {
			state = "PROVISIONING"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"storeId": "st_11111111111111111111111111111111", "name": "store", "state": state})
	})
	configuration, plan := objects(t, r, map[string]attr.Value{"name": types.StringValue("store"), "database_json": types.StringValue(`{"databaseId":"db-1"}`)})
	resp := resource.CreateResponse{State: tfsdk.State{Schema: r.schema(context.Background())}}
	r.Create(context.Background(), resource.CreateRequest{Config: configuration, Plan: plan}, &resp)
	require.True(t, resp.Diagnostics.HasError())
	var id, state types.String
	require.False(t, resp.State.GetAttribute(context.Background(), path.Root("id"), &id).HasError())
	require.Equal(t, "st_11111111111111111111111111111111", id.ValueString())
	require.False(t, resp.State.GetAttribute(context.Background(), path.Root("state"), &state).HasError())
	require.Equal(t, "UNAVAILABLE", state.ValueString())
}

func TestModelReplacementRules(t *testing.T) {
	decode := func(s string) any { v, err := decodeJSON(s); require.NoError(t, err); return v }
	old := decode(`{"embedding":{"provider":"openai","model":"embed-1"},"llm":{"provider":"openai","model":"llm-1"}}`)
	require.False(t, modelsRequireReplacement(old, decode(`{"ttlSeconds":42}`)))
	require.False(t, modelsRequireReplacement(old, decode(`{"llm":{"provider":"openai","model":"llm-2"}}`)))
	require.False(t, modelsRequireReplacement(old, decode(`{"embedding":{"provider":"openai","model":"embed-1","credentials":{"apiKey":"rotated"}}}`)))
	require.True(t, modelsRequireReplacement(old, decode(`{"embedding":{"provider":"openai","model":"embed-2"}}`)))
	require.True(t, modelsRequireReplacement(old, decode(`{"llm":{"provider":"other","model":"llm-1"}}`)))
	require.True(t, modelsRequireReplacement(decode(`{}`), decode(`{"embedding":{"provider":"openai","model":"embed-1"}}`)))
	require.False(t, modelsRequireReplacement(decode(`{}`), decode(`{"classification":{"provider":"openai","model":"classify-1"}}`)))
}

func TestRefreshDetectsExpiryRemovalWithoutErasingCreateOnlyToken(t *testing.T) {
	r := testResource(t, "iris_identity_api_key", func(http.ResponseWriter, *http.Request) {})
	values := map[string]attr.Value{"expires_at": types.Int64Value(42), "token": types.StringValue("retained-secret")}
	var diags diag.Diagnostics
	r.apply(context.Background(), values, map[string]any{}, true, &diags)
	require.False(t, diags.HasError())
	require.Equal(t, types.Int64Value(0), values["expires_at"])
	require.Equal(t, types.StringValue("retained-secret"), values["token"])
}
