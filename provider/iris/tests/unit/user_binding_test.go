package unit_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/stretchr/testify/require"
)

func TestIrisRemovingUserBindingReplacesKey(t *testing.T) {
	f, server := newFixture(t)
	const address = "rediscloud_iris_identity_api_key.key"
	resource.UnitTest(t, resource.TestCase{Steps: []resource.TestStep{
		{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/identity_api_key.tf"), ConfigVariables: config.Variables{"identity_url": config.StringVariable(server.URL + "/identity-proxy"), "user_id": config.StringVariable("alice")}, Check: resource.TestCheckResourceAttr(address, "user_id", "alice")},
		{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/identity_api_key.tf"), ConfigVariables: config.Variables{"identity_url": config.StringVariable(server.URL + "/identity-proxy")}, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(address, plancheck.ResourceActionDestroyBeforeCreate)}}, Check: resource.TestCheckNoResourceAttr(address, "user_id")},
	}})
	f.mu.Lock()
	defer f.mu.Unlock()
	require.Equal(t, 2, f.creates)
	require.Equal(t, 2, f.deletes)
	require.Zero(t, f.updates)
}
