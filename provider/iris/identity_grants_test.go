package iris

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func TestGrantCollectionSchemaType(t *testing.T) {
	block := IdentityAPIKeyResourceSchema(context.Background()).Blocks["grant"].(schema.SetNestedBlock)
	require.True(t, block.Type().Equal(types.SetType{ElemType: grantElementType}))
	require.True(t, grantElementType.Equal(block.NestedObject.Type()))
}
func TestGrantsFromAPIAndRequest(t *testing.T) {
	ctx := context.Background()
	tenant := "tenant-1"
	remote := []identityGrantResponse{{Product: "memory", ResourceType: "mem-store", ResourceID: "store-1", Actions: []string{"write", "read"}}}
	grants, d := grantsFromAPI(ctx, remote)
	require.False(t, d.HasError())
	require.False(t, grants.IsNull())
	var models []IdentityGrantModel
	d = grants.ElementsAs(ctx, &models, false)
	require.False(t, d.HasError())
	require.Len(t, models, 1)
	require.Equal(t, "store-1", models[0].ResourceID.ValueString())
	wire, d := grantsToAPI(ctx, grants, &tenant)
	require.False(t, d.HasError())
	require.Len(t, wire, 1)
	require.Equal(t, &tenant, wire[0].Tenant)
	require.ElementsMatch(t, []string{"read", "write"}, wire[0].Actions)
	payload, err := json.Marshal(identityAPIKeyCreateRequest{Grants: wire})
	require.NoError(t, err)
	require.NotContains(t, string(payload), "constraints")
	require.Contains(t, string(payload), `"resourceType":"mem-store"`)
	// Grant/action ordering carries no meaning and must not cause refresh differences.
	remote[0].Actions = []string{"read", "write"}
	reordered, d := grantsFromAPI(ctx, remote)
	require.False(t, d.HasError())
	require.True(t, grants.Equal(reordered))
	for _, empty := range [][]identityGrantResponse{nil, {}} {
		value, d := grantsFromAPI(ctx, empty)
		require.False(t, d.HasError())
		require.False(t, value.IsNull())
		require.Empty(t, value.Elements())
	}
}
func TestGrantsWarnOnUnsupportedConstraints(t *testing.T) {
	ctx := context.Background()
	for _, constraints := range []string{`{"schema":"restricted","owner":"alice"}`, `{"scope":"limited"}`} {
		_, d := grantsFromAPI(ctx, []identityGrantResponse{{Constraints: json.RawMessage(constraints)}})
		require.False(t, d.HasError())
		require.NotEmpty(t, d.Warnings())
	}
	for _, constraints := range []string{"", `{}`, `null`} {
		_, d := grantsFromAPI(ctx, []identityGrantResponse{{Product: "memory", ResourceType: "mem-store", ResourceID: "store-1", Actions: []string{"read"}, Constraints: json.RawMessage(constraints)}})
		require.False(t, d.HasError())
	}
	for _, value := range []types.Set{types.SetNull(grantElementType), types.SetUnknown(grantElementType), types.SetValueMust(grantElementType, []attr.Value{})} {
		_, d := grantsToAPI(ctx, value, nil)
		require.True(t, d.HasError())
	}
}

// Iris requires one tenant per key, so the key-level tenant must reach every
// grant, and an omitted tenant must stay omitted so Iris can apply its default.
func TestKeyTenantIsStampedOnEveryGrant(t *testing.T) {
	ctx := context.Background()
	grants, d := grantsFromAPI(ctx, []identityGrantResponse{
		{Product: "memory", ResourceType: "mem-store", ResourceID: "store-1", Actions: []string{"read"}},
		{Product: "memory", ResourceType: "mem-store", ResourceID: "store-2", Actions: []string{"write"}},
	})
	require.False(t, d.HasError())
	tenant := "tenant-1"
	stamped, d := grantsToAPI(ctx, grants, &tenant)
	require.False(t, d.HasError(), d)
	require.Len(t, stamped, 2)
	for _, grant := range stamped {
		require.Equal(t, &tenant, grant.Tenant)
	}
	unstamped, d := grantsToAPI(ctx, grants, nil)
	require.False(t, d.HasError(), d)
	for _, grant := range unstamped {
		require.Nil(t, grant.Tenant)
	}
}
