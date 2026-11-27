package unit_test

import (
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/require"
)

func TestIrisTimeoutChangesDoNotMutateResources(t *testing.T) {
	f, server := newFixture(t)
	expiry := time.Now().Add(time.Hour).Unix()
	resource.UnitTest(t, resource.TestCase{Steps: []resource.TestStep{
		{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/timeouts.tf"), ConfigVariables: config.Variables{"memory_url": config.StringVariable(server.URL + "/memory-proxy"), "identity_url": config.StringVariable(server.URL + "/identity-proxy"), "expires_at": config.IntegerVariable(expiry)}},
		{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/timeouts.tf"), ConfigVariables: config.Variables{"memory_url": config.StringVariable(server.URL + "/memory-proxy"), "identity_url": config.StringVariable(server.URL + "/identity-proxy"), "read_timeout": config.StringVariable("20m"), "expires_at": config.IntegerVariable(expiry)}, Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr("rediscloud_iris_agent_memory.store", "timeouts.read", "20m"), resource.TestCheckResourceAttr("rediscloud_iris_agent_memory_type.type", "timeouts.read", "20m"), resource.TestCheckResourceAttr("rediscloud_iris_identity_api_key.key", "timeouts.read", "20m"))},
	}})
	f.mu.Lock()
	defer f.mu.Unlock()
	require.Equal(t, 3, f.creates)
	require.Equal(t, 3, f.deletes)
	require.Zero(t, f.updates, "local timeout changes must never send PATCH")
}
