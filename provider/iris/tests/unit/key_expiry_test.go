package unit_test

import (
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/stretchr/testify/require"
)

func TestIrisExpiredKeyIsRecreatedAfterRefresh(t *testing.T) {
	f, server := newFixture(t)
	resource.UnitTest(t, resource.TestCase{Steps: []resource.TestStep{
		{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/identity_api_key.tf"), ConfigVariables: config.Variables{"identity_url": config.StringVariable(server.URL + "/identity-proxy"), "expires_at": config.IntegerVariable(time.Now().Add(time.Hour).Unix())}},
		{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/identity_api_key.tf"), ConfigVariables: config.Variables{"identity_url": config.StringVariable(server.URL + "/identity-proxy"), "name": config.StringVariable("renewed")}, PreConfig: func() {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.objects["/v1/api-keys/"+keyID]["expiresAt"] = float64(time.Now().Unix() - 10)
		}, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("rediscloud_iris_identity_api_key.key", plancheck.ResourceActionCreate)}}},
	}})
	f.mu.Lock()
	defer f.mu.Unlock()
	require.Equal(t, 2, f.creates)
	require.Equal(t, 1, f.deletes, "the service already removed the expired key")
	require.Zero(t, f.updates, "an expired key must never receive PATCH")
}

// After a fixed expires_at passes, Iris deletes the key and refresh drops it from
// state. Planning must still succeed so the rest of the configuration is usable;
// only this resource fails, when Iris rejects the past expiry at create.
func TestIrisPastExpiryOnNewKeyFailsAtApplyNotPlan(t *testing.T) {
	f, server := newFixture(t)
	past := config.Variables{"identity_url": config.StringVariable(server.URL + "/identity-proxy"), "expires_at": config.IntegerVariable(time.Now().Add(-time.Hour).Unix())}
	resource.UnitTest(t, resource.TestCase{Steps: []resource.TestStep{
		{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/identity_api_key.tf"), ConfigVariables: past, PlanOnly: true, ExpectNonEmptyPlan: true},
		{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/identity_api_key.tf"), ConfigVariables: past, ExpectError: regexp.MustCompile("Failed to create Iris resource")},
	}})
	f.mu.Lock()
	defer f.mu.Unlock()
	require.Zero(t, f.creates)
}

func TestIrisPastKeyExpiryRejectedBeforeReplacement(t *testing.T) {
	f, server := newFixture(t)
	resource.UnitTest(t, resource.TestCase{Steps: []resource.TestStep{
		{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/identity_api_key.tf"), ConfigVariables: config.Variables{"identity_url": config.StringVariable(server.URL + "/identity-proxy")}},
		{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/identity_api_key.tf"), ConfigVariables: config.Variables{"identity_url": config.StringVariable(server.URL + "/identity-proxy"), "user_id": config.StringVariable("alice"), "expires_at": config.IntegerVariable(time.Now().Add(-time.Hour).Unix())}, PlanOnly: true, ExpectError: regexp.MustCompile("Invalid API key expiration")},
		{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/identity_api_key.tf"), ConfigVariables: config.Variables{"identity_url": config.StringVariable(server.URL + "/identity-proxy")}, Check: resource.TestCheckResourceAttr("rediscloud_iris_identity_api_key.key", "token", "create-only-secret")},
	}})
	f.mu.Lock()
	defer f.mu.Unlock()
	require.Equal(t, 1, f.creates)
	require.Equal(t, 1, f.deletes, "only test cleanup may delete the original key")
	require.Zero(t, f.updates)
}

func TestIrisKeyExpiryCanBeRemoved(t *testing.T) {
	f, server := newFixture(t)
	const address = "rediscloud_iris_identity_api_key.key"
	resource.UnitTest(t, resource.TestCase{Steps: []resource.TestStep{
		{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/identity_api_key.tf"), ConfigVariables: config.Variables{"identity_url": config.StringVariable(server.URL + "/identity-proxy"), "expires_at": config.IntegerVariable(time.Now().Add(time.Hour).Unix())}},
		{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/identity_api_key.tf"), ConfigVariables: config.Variables{"identity_url": config.StringVariable(server.URL + "/identity-proxy")}, Check: resource.TestCheckResourceAttr(address, "expires_at", "0")},
	}})
	f.mu.Lock()
	defer f.mu.Unlock()
	require.Equal(t, 1, f.creates)
	require.Equal(t, 1, f.updates)
}
