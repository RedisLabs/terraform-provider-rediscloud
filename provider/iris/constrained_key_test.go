package iris

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func TestConstrainedKeyRefreshMetadataAndDelete(t *testing.T) {
	ctx := context.Background()
	tenant := "tenant-1"
	id := "key-1"
	name := "key"
	remote := []identityGrantResponse{{Product: "memory", ResourceType: "mem-store", ResourceID: "store-1", Actions: []string{"read"}, Constraints: json.RawMessage(`{"scope":"private"}`)}}
	api := identityAPIKeyResponse{KeyID: &id, Name: &name, Grants: &remote, TenantID: &tenant}
	patches, deletes := 0, 0
	r := testResource(t, "iris_identity_api_key", func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodPatch:
			var body map[string]any
			require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
			require.NotContains(t, body, "grants")
			patches++
			name = body["name"].(string)
		case http.MethodDelete:
			deletes++
			w.WriteHeader(204)
			return
		}
		require.NoError(t, json.NewEncoder(w).Encode(api))
	})
	previous := IdentityAPIKeyModel{Timeouts: nullTimeouts()}
	d := previous.setFromAPI(ctx, &api)
	require.False(t, d.HasError())
	require.NotEmpty(t, d.Warnings())
	state := modelState(t, r, &previous)
	read := resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, &read)
	require.False(t, read.Diagnostics.HasError())
	require.False(t, read.State.Raw.IsNull())
	config, _ := objects(t, r, map[string]attr.Value{"name": types.StringValue("renamed"), "grant": previous.Grants})
	planned := previous
	planned.Name = types.StringValue("renamed")
	plan := tfsdk.Plan(modelState(t, r, &planned))
	updated := resource.UpdateResponse{State: read.State}
	r.Update(ctx, resource.UpdateRequest{State: read.State, Config: config, Plan: plan}, &updated)
	require.False(t, updated.Diagnostics.HasError(), updated.Diagnostics)
	require.Equal(t, 1, patches)
	var models []IdentityGrantModel
	d = previous.Grants.ElementsAs(ctx, &models, false)
	require.False(t, d.HasError())
	models[0].Actions = types.SetValueMust(types.StringType, []attr.Value{types.StringValue("write")})
	set, d := types.SetValueFrom(ctx, grantElementType, models)
	require.False(t, d.HasError())
	config, _ = objects(t, r, map[string]attr.Value{"name": types.StringValue("renamed"), "grant": set})
	planned.Grants = set
	plan = tfsdk.Plan(modelState(t, r, &planned))
	blocked := resource.UpdateResponse{State: updated.State}
	r.Update(ctx, resource.UpdateRequest{State: updated.State, Config: config, Plan: plan}, &blocked)
	require.True(t, blocked.Diagnostics.HasError())
	require.Equal(t, 1, patches)
	deleted := resource.DeleteResponse{}
	r.Delete(ctx, resource.DeleteRequest{State: updated.State}, &deleted)
	require.False(t, deleted.Diagnostics.HasError())
	require.Equal(t, 1, deletes)
}
