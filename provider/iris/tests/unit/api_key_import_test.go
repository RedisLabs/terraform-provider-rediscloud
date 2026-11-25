package unit_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/require"
)

func TestIrisAPIKeyImportIsRejected(t *testing.T) {
	f, server := newFixture(t)
	resource.UnitTest(t, resource.TestCase{Steps: []resource.TestStep{
		{
			ProtoV5ProviderFactories: factories(t),
			ConfigFile:               config.StaticFile("testdata/identity_api_key.tf"),
			ConfigVariables:          config.Variables{"identity_url": config.StringVariable(server.URL + "/identity-proxy")},
			Check:                    resource.TestCheckResourceAttr("rediscloud_iris_identity_api_key.key", "token", "create-only-secret"),
		},
		{
			ProtoV5ProviderFactories: factories(t),
			ConfigFile:               config.StaticFile("testdata/identity_api_key.tf"),
			ConfigVariables:          config.Variables{"identity_url": config.StringVariable(server.URL + "/identity-proxy")},
			ResourceName:             "rediscloud_iris_identity_api_key.key",
			ImportState:              true,
			ImportStateId:            keyID,
			ExpectError:              regexp.MustCompile("Resource Import Not Implemented"),
		},
	}})
	f.mu.Lock()
	defer f.mu.Unlock()
	require.Empty(t, f.objects)
	require.Equal(t, 1, f.creates)
	require.Equal(t, 1, f.deletes)
}
