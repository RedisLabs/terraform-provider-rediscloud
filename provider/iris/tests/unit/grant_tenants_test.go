package unit_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/stretchr/testify/require"
)

const keyAddress = "rediscloud_iris_identity_api_key.key"

func replacesKey() resource.ConfigPlanChecks {
	return resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(keyAddress, plancheck.ResourceActionDestroyBeforeCreate)}}
}

func TestIrisKeyUpdatesKeepTheStoredTenant(t *testing.T) {
	f, server := newFixture(t)
	url := config.StringVariable(server.URL + "/identity-proxy")
	resource.UnitTest(t, resource.TestCase{Steps: []resource.TestStep{
		{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/identity_api_key.tf"), ConfigVariables: config.Variables{"identity_url": url, "tenant": config.StringVariable("Tenant-ONE")}, Check: resource.TestCheckResourceAttr(keyAddress, "tenant", "Tenant-ONE")},
		// Omitting the tenant later keeps the one Iris stored instead of replacing the key.
		{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/identity_api_key.tf"), ConfigVariables: config.Variables{"identity_url": url, "name": config.StringVariable("renamed")}, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(keyAddress, plancheck.ResourceActionUpdate)}}, Check: resource.TestCheckResourceAttr(keyAddress, "tenant", "Tenant-ONE")},
		{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/identity_api_key.tf"), ConfigVariables: config.Variables{"identity_url": url, "name": config.StringVariable("policy-update"), "actions": config.ListVariable(config.StringVariable("read"), config.StringVariable("write"))}, Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr(keyAddress, "tenant", "Tenant-ONE"), resource.TestCheckTypeSetElemNestedAttrs(keyAddress, "grant.*", map[string]string{"actions.#": "2"}))},
	}})
	f.mu.Lock()
	defer f.mu.Unlock()
	require.Equal(t, 1, f.creates)
}

func TestIrisTenantChangesReplaceKey(t *testing.T) {
	for _, mode := range []string{"tenant", "user_and_tenant", "forced"} {
		t.Run(mode, func(t *testing.T) {
			f, server := newFixture(t)
			user := "alice"
			if mode == "user_and_tenant" {
				user = "bob"
			}
			var taint []string
			if mode == "forced" {
				taint = []string{keyAddress}
			}
			url := config.StringVariable(server.URL + "/identity-proxy")
			resource.UnitTest(t, resource.TestCase{Steps: []resource.TestStep{
				{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/identity_api_key.tf"), ConfigVariables: config.Variables{"identity_url": url, "user_id": config.StringVariable("alice")}},
				{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/identity_api_key.tf"), ConfigVariables: config.Variables{"identity_url": url, "user_id": config.StringVariable(user), "tenant": config.StringVariable("another-tenant")}, Taint: taint, ConfigPlanChecks: replacesKey(), Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr(keyAddress, "tenant", "another-tenant"), resource.TestCheckResourceAttr(keyAddress, "token", "create-only-secret-2"))},
			}})
			f.mu.Lock()
			defer f.mu.Unlock()
			require.Equal(t, 2, f.creates)
			require.Equal(t, 2, f.deletes)
			require.Zero(t, f.updates)
		})
	}
}

// Iris refuses a key whose grants span tenants. A key with several grants can
// only change tenant as a whole, so the replacement must carry the new tenant on
// every grant; the fixture rejects a request whose grants disagree.
func TestIrisTenantChangeWithSeveralGrantsReplacesKey(t *testing.T) {
	f, server := newFixture(t)
	variables := func(tenant string) config.Variables {
		return config.Variables{"identity_url": config.StringVariable(server.URL + "/identity-proxy"), "tenant": config.StringVariable(tenant)}
	}
	resource.UnitTest(t, resource.TestCase{Steps: []resource.TestStep{
		{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/multi_grant_key.tf"), ConfigVariables: variables("tenant-one"), Check: resource.TestCheckResourceAttr(keyAddress, "grant.#", "2")},
		{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/multi_grant_key.tf"), ConfigVariables: variables("tenant-two"), ConfigPlanChecks: replacesKey(), Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr(keyAddress, "tenant", "tenant-two"), resource.TestCheckResourceAttr(keyAddress, "grant.#", "2"))},
	}})
	f.mu.Lock()
	defer f.mu.Unlock()
	require.Equal(t, 2, f.creates)
	require.Zero(t, f.updates)
}

// The tenant belongs to the key, so a grant cannot name its own and a key can
// never be configured with conflicting tenants.
func TestIrisGrantBlockRejectsATenant(t *testing.T) {
	_, server := newFixture(t)
	resource.UnitTest(t, resource.TestCase{Steps: []resource.TestStep{{ProtoV5ProviderFactories: factories(t), PlanOnly: true, ExpectError: regexp.MustCompile(`argument named "tenant" is not\s+expected here`), Config: fmt.Sprintf(`
provider "rediscloud" {
  iris_identity_url   = %q
  iris_identity_token = "identity-token"
}
resource "rediscloud_iris_identity_api_key" "key" {
  name = "grant-tenant"
  grant {
    tenant        = "tenant-one"
    product       = "memory"
    resource_type = "mem-store"
    resource_id   = "store-1"
    actions       = ["read"]
  }
}
`, server.URL+"/identity-proxy")}}})
}

// A tenant that is unknown during planning cannot be compared with the stored
// one, so, as for any attribute that forces replacement, the key is replaced. A
// replacement that is already planned stays valid however the value resolves.
func TestIrisUnknownTenantReplacesKey(t *testing.T) {
	for _, tc := range []struct{ name, tenant, trigger, want string }{
		{"resolves to the same tenant", "tenant-one", "two", "tenant-one"},
		{"resolves to another tenant", "tenant-two", "one", "tenant-two"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, server := newFixture(t)
			variables := func(tenant, trigger string) config.Variables {
				return config.Variables{"identity_url": config.StringVariable(server.URL + "/identity-proxy"), "tenant": config.StringVariable(tenant), "tenant_trigger": config.StringVariable(trigger)}
			}
			resource.UnitTest(t, resource.TestCase{Steps: []resource.TestStep{
				{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/computed_tenant.tf"), ConfigVariables: variables("tenant-one", "one")},
				{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/computed_tenant.tf"), ConfigVariables: variables(tc.tenant, tc.trigger), ConfigPlanChecks: replacesKey(), Check: resource.ComposeTestCheckFunc(resource.TestCheckResourceAttr(keyAddress, "tenant", tc.want), resource.TestCheckResourceAttr(keyAddress, "token", "create-only-secret-2"))},
			}})
			f.mu.Lock()
			defer f.mu.Unlock()
			require.Equal(t, 2, f.creates)
			require.Zero(t, f.updates)
		})
	}
}

func TestIrisComputedGrantResourceChangeDoesNotReplaceKey(t *testing.T) {
	f, server := newFixture(t)
	url := config.StringVariable(server.URL + "/identity-proxy")
	resource.UnitTest(t, resource.TestCase{Steps: []resource.TestStep{
		{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/computed_tenant.tf"), ConfigVariables: config.Variables{"identity_url": url, "tenant": config.StringVariable("tenant-one"), "computed_tenant": config.BoolVariable(false)}},
		{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/computed_tenant.tf"), ConfigVariables: config.Variables{"identity_url": url, "tenant": config.StringVariable("tenant-one"), "computed_tenant": config.BoolVariable(false), "store_id": config.StringVariable("store-2")}, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(keyAddress, plancheck.ResourceActionUpdate)}}, Check: resource.TestCheckResourceAttr(keyAddress, "token", "create-only-secret")},
	}})
	f.mu.Lock()
	defer f.mu.Unlock()
	require.Equal(t, 1, f.creates)
	require.Equal(t, 1, f.updates)
}
