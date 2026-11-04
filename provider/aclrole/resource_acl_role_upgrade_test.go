package aclrole_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/aclrole"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

const aclRoleTypeName = testhelpers.FrameworkTestProviderTypeName + "_acl_role"

func aclRoleServer(t *testing.T) (tfprotov5.ProviderServer, tftypes.Type) {
	t.Helper()
	ctx := context.Background()
	server, err := testhelpers.FrameworkProviderFactories(nil, aclrole.NewAclRoleResource)[testhelpers.FrameworkTestProviderTypeName]()
	require.NoError(t, err)

	var resp resource.SchemaResponse
	aclrole.NewAclRoleResource().Schema(ctx, resource.SchemaRequest{}, &resp)
	require.False(t, resp.Diagnostics.HasError())
	return server, resp.Schema.Type().TerraformType(ctx)
}

// upgradeSDKAclRoleJSON runs UpgradeResourceState on a state stored by SDKv2 2.20.0
// (schema version 0), as Terraform does before planning, even without refresh.
func upgradeSDKAclRoleJSON(t *testing.T, raw string) tftypes.Value {
	t.Helper()
	server, typ := aclRoleServer(t)
	resp, err := server.UpgradeResourceState(context.Background(), &tfprotov5.UpgradeResourceStateRequest{
		TypeName: aclRoleTypeName,
		Version:  0,
		RawState: &tfprotov5.RawState{JSON: []byte(raw)},
	})
	require.NoError(t, err)
	require.Empty(t, resp.Diagnostics)
	upgraded, err := resp.UpgradedState.Unmarshal(typ)
	require.NoError(t, err)
	return upgraded
}

func aclRoleValue(t *testing.T, typ tftypes.Type, raw string) tftypes.Value {
	t.Helper()
	v, err := (&tfprotov5.RawState{JSON: []byte(raw)}).Unmarshal(typ)
	require.NoError(t, err)
	return v
}

func planAclRole(t *testing.T, prior tftypes.Value, config, proposed string) tftypes.Value {
	t.Helper()
	ctx := context.Background()
	server, typ := aclRoleServer(t)
	_, err := server.ConfigureProvider(ctx, &tfprotov5.ConfigureProviderRequest{})
	require.NoError(t, err)

	dynamic := func(v tftypes.Value) *tfprotov5.DynamicValue {
		dv, err := tfprotov5.NewDynamicValue(typ, v)
		require.NoError(t, err)
		return &dv
	}
	resp, err := server.PlanResourceChange(ctx, &tfprotov5.PlanResourceChangeRequest{
		TypeName:         aclRoleTypeName,
		PriorState:       dynamic(prior),
		ProposedNewState: dynamic(aclRoleValue(t, typ, proposed)),
		Config:           dynamic(aclRoleValue(t, typ, config)),
	})
	require.NoError(t, err)
	require.Empty(t, resp.Diagnostics)
	planned, err := resp.PlannedState.Unmarshal(typ)
	require.NoError(t, err)
	return planned
}

// SDKv2 stored an omitted regions argument as an empty set.
const sdkAclRoleState = `{"id":"1466","name":"role","rule":[{"name":"rule","database":[` +
	`{"subscription":152509,"database":51178935,"regions":[]},` +
	`{"subscription":152509,"database":51178936,"regions":["us-east-1"]}]}],"timeouts":null}`

func TestAclRoleUpgradeSDKStateWithOmittedRegionsGivesEmptyPlan(t *testing.T) {
	prior := upgradeSDKAclRoleJSON(t, sdkAclRoleState)

	const databases = `[{"subscription":152509,"database":51178935,"regions":null},` +
		`{"subscription":152509,"database":51178936,"regions":["us-east-1"]}]`
	// Terraform proposes the configured value for regions, which is not computed.
	config := `{"id":null,"name":"role","rule":[{"name":"rule","database":` + databases + `}],"timeouts":null}`
	proposed := `{"id":"1466","name":"role","rule":[{"name":"rule","database":` + databases + `}],"timeouts":null}`

	planned := planAclRole(t, prior, config, proposed)
	assert.True(t, planned.Equal(prior), "an omitted regions argument must give an empty plan after the upgrade, got %s", planned)
}

// SDKv2 stored a configured empty regions argument the same way, so it is applied once.
func TestAclRoleUpgradeSDKStateWithEmptyRegionsConfigured(t *testing.T) {
	prior := upgradeSDKAclRoleJSON(t, sdkAclRoleState)

	const databases = `[{"subscription":152509,"database":51178935,"regions":[]},` +
		`{"subscription":152509,"database":51178936,"regions":["us-east-1"]}]`
	config := `{"id":null,"name":"role","rule":[{"name":"rule","database":` + databases + `}],"timeouts":null}`
	proposed := `{"id":"1466","name":"role","rule":[{"name":"rule","database":` + databases + `}],"timeouts":null}`

	planned := planAclRole(t, prior, config, proposed)
	_, typ := aclRoleServer(t)
	want := aclRoleValue(t, typ, proposed)
	assert.False(t, planned.Equal(prior), "a configured empty regions argument must be planned")
	assert.True(t, planned.Equal(want), "the plan must keep the configured empty regions, got %s", planned)
}
