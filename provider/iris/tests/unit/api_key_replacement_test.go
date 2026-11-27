package unit_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/require"
)

func TestIrisKeyReplacementKeepsOldKeyUntilConsumerSwitch(t *testing.T) {
	f, server := newFixture(t)
	check := func(keys int, token string, oldExists bool) resource.TestCheckFunc {
		return resource.ComposeTestCheckFunc(
			resource.TestCheckOutput("active_token", token),
			func(state *terraform.State) error {
				f.mu.Lock()
				defer f.mu.Unlock()
				require.Len(t, f.objects, keys)
				_, exists := f.objects["/v1/api-keys/"+keyID]
				require.Equal(t, oldExists, exists)
				if keys == 2 {
					old := state.RootModule().Resources["rediscloud_iris_identity_api_key.old.0"].Primary
					next := state.RootModule().Resources["rediscloud_iris_identity_api_key.next.0"].Primary
					require.NotEqual(t, old.ID, next.ID)
					require.NotEqual(t, old.Attributes["token"], next.Attributes["token"])
				}
				return nil
			},
		)
	}
	resource.UnitTest(t, resource.TestCase{Steps: []resource.TestStep{
		{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/key_replacement.tf"), ConfigVariables: config.Variables{"identity_url": config.StringVariable(server.URL + "/identity-proxy")}, Check: check(1, "create-only-secret", true)},
		{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/key_replacement.tf"), ConfigVariables: config.Variables{"identity_url": config.StringVariable(server.URL + "/identity-proxy"), "create_next": config.BoolVariable(true), "use_next": config.BoolVariable(true)}, Check: check(2, "create-only-secret-2", true)},
		{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/key_replacement.tf"), ConfigVariables: config.Variables{"identity_url": config.StringVariable(server.URL + "/identity-proxy"), "keep_old": config.BoolVariable(false), "create_next": config.BoolVariable(true), "use_next": config.BoolVariable(true)}, Check: check(1, "create-only-secret-2", false)},
	}})
	f.mu.Lock()
	defer f.mu.Unlock()
	require.Empty(t, f.objects)
	require.Equal(t, 2, f.creates)
	require.Equal(t, 2, f.deletes)
	require.Zero(t, f.updates)
}
