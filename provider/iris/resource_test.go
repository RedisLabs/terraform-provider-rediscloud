package iris

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/customtypes"
)

func TestSchemasMatchModels(t *testing.T) {
	models := []any{AgentMemoryModel{Timeouts: nullTimeouts(), Endpoints: types.SetNull(endpointElementType), CustomMemoryTypes: types.SetNull(customMemoryTypeElementType), BuiltInDetectors: types.SetNull(builtInDetectorElementType), CustomDetectors: types.SetNull(customDetectorElementType)}, AgentMemoryTypeModel{Timeouts: nullTimeouts(), Fields: types.SetNull(memoryFieldElementType)}, IdentityAPIKeyModel{Timeouts: nullTimeouts(), Grants: types.SetNull(grantElementType)}}
	for i, constructor := range Resources() {
		r := constructor()
		resp := resource.SchemaResponse{}
		r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
		require.True(t, resp.Schema.Type().Equal(types.ObjectType{AttrTypes: customtypes.AttrTypesOf(models[i])}))
	}
}
func resourceSchema(r resource.Resource) schema.Schema {
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	return resp.Schema
}
func modelState(t *testing.T, r resource.Resource, model any) tfsdk.State {
	t.Helper()
	s := tfsdk.State{Schema: resourceSchema(r)}
	d := s.Set(context.Background(), model)
	require.False(t, d.HasError(), d)
	return s
}
func testResource(t *testing.T, name string, handler http.HandlerFunc) resource.Resource {
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	connector, err := client.NewIrisServiceClient(client.IrisServiceConfig{URL: server.URL, Token: "test-token"})
	require.NoError(t, err)
	for _, constructor := range Resources() {
		r := constructor()
		var meta resource.MetadataResponse
		r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "test"}, &meta)
		if meta.TypeName != "test_"+name {
			continue
		}
		resp := resource.ConfigureResponse{}
		r.(resource.ResourceWithConfigure).Configure(context.Background(), resource.ConfigureRequest{ProviderData: &client.ApiClient{Iris: map[string]*client.IrisServiceClient{"agent-memory": connector, "identity": connector}}}, &resp)
		require.False(t, resp.Diagnostics.HasError())
		return r
	}
	t.Fatal("unknown resource")
	return nil
}
func objects(t *testing.T, r resource.Resource, configured map[string]attr.Value) (tfsdk.Config, tfsdk.Plan) {
	ctx := context.Background()
	s := resourceSchema(r)
	typ := s.Type().(types.ObjectType)
	configValues := map[string]attr.Value{}
	planValues := map[string]attr.Value{}
	for name, a := range s.Attributes {
		value := configured[name]
		if value == nil {
			value = nullValue(t, ctx, a.GetType())
		}
		configValues[name] = value
		planValues[name] = value
		if a.IsWriteOnly() {
			planValues[name] = nullValue(t, ctx, a.GetType())
		}
		if a.IsComputed() && value.IsNull() {
			planValues[name], _ = a.GetType().ValueFromTerraform(ctx, tftypes.NewValue(a.GetType().TerraformType(ctx), tftypes.UnknownValue))
		}
	}
	for name, block := range s.Blocks {
		value := configured[name]
		if value == nil {
			if name == "grant" {
				actions := types.SetValueMust(types.StringType, []attr.Value{types.StringValue("read")})
				set, d := types.SetValueFrom(ctx, grantElementType, []IdentityGrantModel{{Product: types.StringValue("memory"), ResourceType: types.StringValue("mem-store"), ResourceID: types.StringValue("store-1"), Actions: actions}})
				require.False(t, d.HasError())
				value = set
			} else {
				value = nullValue(t, ctx, block.Type())
			}
		}
		configValues[name] = value
		planValues[name] = value
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
	resp := resource.CreateResponse{State: tfsdk.State{Schema: resourceSchema(r)}}
	r.Create(context.Background(), resource.CreateRequest{Config: configuration, Plan: plan}, &resp)
	require.True(t, resp.Diagnostics.HasError())
	var id, token types.String
	require.False(t, resp.State.GetAttribute(context.Background(), path.Root("id"), &id).HasError())
	require.Equal(t, "key-1", id.ValueString())
	require.False(t, resp.State.GetAttribute(context.Background(), path.Root("token"), &token).HasError())
	require.Equal(t, "creation-secret", token.ValueString())
}

func TestMemoryTypeReadTimeoutRetainsCreationStateWithoutError(t *testing.T) {
	r := testResource(t, "iris_agent_memory_type", func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPost {
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"name": "profile", "description": "Profile", "fields": []any{}, "revision": "1"}))
			return
		}
		select {
		case <-req.Context().Done():
		case <-time.After(2 * time.Second):
		}
	})
	config, plan := objects(t, r, map[string]attr.Value{"name": types.StringValue("profile"), "store_id": types.StringValue("store-1"), "description": types.StringValue("Profile")})
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	resp := resource.CreateResponse{State: tfsdk.State{Schema: resourceSchema(r)}}
	r.Create(ctx, resource.CreateRequest{Config: config, Plan: plan}, &resp)
	require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
	require.Len(t, resp.Diagnostics.Warnings(), 1)
	var saved AgentMemoryTypeModel
	require.False(t, resp.State.Get(context.Background(), &saved).HasError())
	require.Equal(t, "store-1/profile", saved.ID.ValueString())
	require.Equal(t, "1", saved.Revision.ValueString())
}
func TestReadAndDeleteNotFound(t *testing.T) {
	r := testResource(t, "iris_identity_api_key", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) })
	_, plan := objects(t, r, map[string]attr.Value{"id": types.StringValue("key-1"), "name": types.StringValue("key")})
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
	configuration, plan := objects(t, r, map[string]attr.Value{"name": types.StringValue("store"), "database_id": types.StringValue("db-1")})
	resp := resource.CreateResponse{State: tfsdk.State{Schema: resourceSchema(r)}}
	r.Create(context.Background(), resource.CreateRequest{Config: configuration, Plan: plan}, &resp)
	require.True(t, resp.Diagnostics.HasError())
	var id, state types.String
	require.False(t, resp.State.GetAttribute(context.Background(), path.Root("id"), &id).HasError())
	require.Equal(t, "st_11111111111111111111111111111111", id.ValueString())
	require.False(t, resp.State.GetAttribute(context.Background(), path.Root("state"), &state).HasError())
	require.Equal(t, "UNAVAILABLE", state.ValueString())
}

func nullValue(t *testing.T, ctx context.Context, typ attr.Type) attr.Value {
	t.Helper()
	v, err := typ.ValueFromTerraform(ctx, tftypes.NewValue(typ.TerraformType(ctx), nil))
	require.NoError(t, err)
	return v
}
func TestRefreshDetectsExpiryRemovalWithoutErasingCreateOnlyToken(t *testing.T) {
	m := IdentityAPIKeyModel{Timeouts: nullTimeouts(), ExpiresAt: types.Int64Value(42), Token: types.StringValue("retained-secret")}
	m.setFromAPI(context.Background(), &identityAPIKeyResponse{})
	require.Equal(t, types.Int64Value(0), m.ExpiresAt)
	require.Equal(t, types.StringValue("retained-secret"), m.Token)
}

func nullTimeouts() timeouts.Value {
	typ := timeouts.BlockAll(context.Background()).Type().(timeouts.Type)
	return timeouts.Value{Object: types.ObjectNull(typ.AttrTypes)}
}

func TestIdentityKeyExpiryPlanValidation(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name    string
		value   types.Int64
		invalid bool
	}{
		{"past", types.Int64Value(time.Now().Add(-time.Hour).Unix()), true},
		{"negative", types.Int64Value(-1), true},
		{"future", types.Int64Value(time.Now().Add(time.Hour).Unix()), false},
		{"never", types.Int64Value(0), false},
		{"unknown", types.Int64Unknown(), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := NewIdentityAPIKeyResource()
			config, plan := objects(t, r, map[string]attr.Value{"expires_at": tc.value})
			existing, _ := objects(t, r, map[string]attr.Value{"expires_at": types.Int64Value(0)})
			modify := func(state tfsdk.State) resource.ModifyPlanResponse {
				var resp resource.ModifyPlanResponse
				r.(resource.ResourceWithModifyPlan).ModifyPlan(ctx, resource.ModifyPlanRequest{Config: config, Plan: plan, State: state}, &resp)
				return resp
			}
			// Iris rejects a past expiry when it creates the key, before anything
			// exists, so a new key never fails during planning.
			created := modify(tfsdk.State{Schema: config.Schema, Raw: tftypes.NewValue(config.Schema.Type().TerraformType(ctx), nil)})
			require.False(t, created.Diagnostics.HasError(), created.Diagnostics)
			// Changing an existing key's expiry must not end in a rejected request.
			changed := modify(tfsdk.State(existing))
			require.Equal(t, tc.invalid, changed.Diagnostics.HasError(), changed.Diagnostics)
			require.Empty(t, changed.RequiresReplace)
		})
	}
}

// Replacing a key deletes it first, so a tenant change warns before apply. A
// tenant that is unknown during planning replaces the key as well.
func TestIdentityKeyTenantChangeWarns(t *testing.T) {
	ctx := context.Background()
	r := NewIdentityAPIKeyResource()
	existing, _ := objects(t, r, map[string]attr.Value{"tenant": types.StringValue("tenant-1")})
	for _, tc := range []struct {
		name   string
		tenant types.String
		warns  bool
	}{
		{"unchanged", types.StringValue("tenant-1"), false},
		{"another tenant", types.StringValue("tenant-2"), true},
		{"unknown", types.StringUnknown(), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config, plan := objects(t, r, map[string]attr.Value{"tenant": tc.tenant})
			var resp resource.ModifyPlanResponse
			r.(resource.ResourceWithModifyPlan).ModifyPlan(ctx, resource.ModifyPlanRequest{Config: config, Plan: plan, State: tfsdk.State(existing)}, &resp)
			require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
			require.Equal(t, tc.warns, resp.Diagnostics.WarningsCount() == 1, resp.Diagnostics)
			if tc.warns {
				require.Equal(t, "Changing the tenant replaces the key", resp.Diagnostics.Warnings()[0].Summary())
			}
		})
	}
}

func TestUnchangedPastKeyExpiryAllowsNoOpPlan(t *testing.T) {
	r := NewIdentityAPIKeyResource()
	config, plan := objects(t, r, map[string]attr.Value{"name": types.StringValue("expired"), "expires_at": types.Int64Value(time.Now().Add(-time.Hour).Unix())})
	var resp resource.ModifyPlanResponse
	r.(resource.ResourceWithModifyPlan).ModifyPlan(context.Background(), resource.ModifyPlanRequest{Config: config, Plan: plan, State: tfsdk.State(config)}, &resp)
	require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
	// A refreshed-away record is a creation. Planning must not fail, otherwise the
	// whole configuration is blocked; Iris rejects the past expiry at create.
	ctx := context.Background()
	resp = resource.ModifyPlanResponse{}
	r.(resource.ResourceWithModifyPlan).ModifyPlan(ctx, resource.ModifyPlanRequest{Config: config, Plan: plan, State: tfsdk.State{Schema: config.Schema, Raw: tftypes.NewValue(config.Schema.Type().TerraformType(ctx), nil)}}, &resp)
	require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
}

func TestSavedStatePreservesKnownValuesAndNormalizesNestedUnknowns(t *testing.T) {
	ctx := context.Background()
	grants, d := types.SetValueFrom(ctx, grantElementType, []IdentityGrantModel{{
		Product: types.StringValue("memory"), ResourceType: types.StringValue("mem-store"), ResourceID: types.StringValue("store-1"), Actions: types.SetUnknown(types.StringType),
	}})
	require.False(t, d.HasError())
	m := IdentityAPIKeyModel{Timeouts: nullTimeouts(), ID: types.StringValue("key-1"), Token: types.StringValue("create-only-token"), Name: types.StringValue(""), ExpiresAt: types.Int64Value(0), Tenant: types.StringUnknown(), Grants: grants}
	state := tfsdk.State{Schema: resourceSchema(NewIdentityAPIKeyResource())}
	d = saveState(ctx, &state, &m)
	require.False(t, d.HasError(), d)
	require.True(t, state.Raw.IsFullyKnown())
	var got IdentityAPIKeyModel
	require.False(t, state.Get(ctx, &got).HasError())
	require.Equal(t, m.ID, got.ID)
	require.Equal(t, m.Token, got.Token)
	require.Equal(t, m.Name, got.Name)
	require.Equal(t, m.ExpiresAt, got.ExpiresAt)
	require.True(t, got.Tenant.IsNull())
	var rows []IdentityGrantModel
	require.False(t, got.Grants.ElementsAs(ctx, &rows, false).HasError())
	require.True(t, rows[0].Actions.IsNull())
	require.Equal(t, "memory", rows[0].Product.ValueString())
}

func TestMemoryCreateReadFailureKeepsIDAndOmitsWriteOnlyKeys(t *testing.T) {
	r := testResource(t, "iris_agent_memory", func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPost {
			require.NoError(t, json.NewEncoder(w).Encode(map[string]string{"storeId": "store-1", "state": "PROVISIONING"}))
			return
		}
		w.WriteHeader(http.StatusForbidden)
	})
	config, plan := objects(t, r, map[string]attr.Value{
		"name": types.StringValue("store"), "database_id": types.StringValue("db-1"),
		"llm_provider": types.StringValue("openai"), "llm_model": types.StringValue("llm"), "llm_api_key": types.StringValue("fake-llm-key"), "llm_api_key_version": types.Int64Value(1),
		"embedding_provider": types.StringValue("openai"), "embedding_model": types.StringValue("embedding"), "embedding_api_key": types.StringValue("fake-embedding-key"), "embedding_api_key_version": types.Int64Value(1),
	})
	resp := resource.CreateResponse{State: tfsdk.State{Schema: resourceSchema(r)}}
	r.Create(context.Background(), resource.CreateRequest{Config: config, Plan: plan}, &resp)
	require.True(t, resp.Diagnostics.HasError())
	require.True(t, resp.State.Raw.IsFullyKnown())
	var got AgentMemoryModel
	require.False(t, resp.State.Get(context.Background(), &got).HasError())
	require.Equal(t, "store-1", got.ID.ValueString())
	require.True(t, got.LLMAPIKey.IsNull())
	require.True(t, got.EmbeddingAPIKey.IsNull())
	require.Equal(t, int64(1), got.LLMAPIKeyVersion.ValueInt64())
}
