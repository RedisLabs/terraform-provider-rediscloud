package aclrole_test

import (
	"testing"

	rediscloudApi "github.com/RedisLabs/rediscloud-go-api"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

// The role name is unknown at plan, so Terraform defers the read to apply.
// The computed rule set must then plan as unknown, not as an empty set.
const aclRoleDeferredReadConfig = `
resource "terraform_data" "name" {
  input = "deferred"
}

data "rediscloud_acl_role" "example" {
  name = terraform_data.name.output
}

output "rule_count" {
  value = length(data.rediscloud_acl_role.example.rule)
}

output "first_rule_name" {
  value = one(data.rediscloud_acl_role.example.rule).name
}
`

func TestDataSourceRedisCloudAclRole_deferredRead(t *testing.T) {
	// Configure needs credentials, but a deferred read never calls the API.
	t.Setenv(rediscloudApi.AccessKeyEnvVar, "unused")
	t.Setenv(rediscloudApi.SecretKeyEnvVar, "unused")

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:             aclRoleDeferredReadConfig,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectUnknownOutputValue("rule_count"),
						plancheck.ExpectUnknownOutputValue("first_rule_name"),
					},
				},
			},
		},
	})
}
