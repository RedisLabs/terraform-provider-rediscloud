package unit_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/require"
)

func TestIrisRejectsNonCanonicalGrantNamesBeforeCreate(t *testing.T) {
	for _, attribute := range []string{"product", "resource_type"} {
		for _, value := range []string{"Mixed-Case", " surrounding-whitespace "} {
			t.Run(attribute+"/"+value, func(t *testing.T) {
				f, server := newFixture(t)
				resource.UnitTest(t, resource.TestCase{Steps: []resource.TestStep{{
					ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/identity_api_key.tf"),
					ConfigVariables: config.Variables{"identity_url": config.StringVariable(server.URL + "/identity-proxy"), attribute: config.StringVariable(value)},
					PlanOnly:        true, ExpectError: regexp.MustCompile("Non-canonical grant name"),
				}}})
				require.Zero(t, f.creates)
			})
		}
	}
}

func TestIrisRejectsWhitespaceTenantsBeforeCreate(t *testing.T) {
	for _, tenant := range []string{" tenant-1 ", "\u00a0tenant-1\u00a0", "   "} {
		t.Run(tenant, func(t *testing.T) {
			f, server := newFixture(t)
			resource.UnitTest(t, resource.TestCase{Steps: []resource.TestStep{{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/identity_api_key.tf"), ConfigVariables: config.Variables{"identity_url": config.StringVariable(server.URL + "/identity-proxy"), "tenant": config.StringVariable(tenant)}, PlanOnly: true, ExpectError: regexp.MustCompile("Invalid tenant")}}})
			require.Zero(t, f.creates)
		})
	}
}
func TestIrisPreservesTenantCase(t *testing.T) {
	_, server := newFixture(t)
	resource.UnitTest(t, resource.TestCase{Steps: []resource.TestStep{{ProtoV5ProviderFactories: factories(t), ConfigFile: config.StaticFile("testdata/identity_api_key.tf"), ConfigVariables: config.Variables{"identity_url": config.StringVariable(server.URL + "/identity-proxy"), "tenant": config.StringVariable("Tenant-ONE")}, Check: resource.TestCheckResourceAttr("rediscloud_iris_identity_api_key.key", "tenant", "Tenant-ONE")}}})
}
