package aclrole_test

import (
	"regexp"
	"testing"

	rediscloudApi "github.com/RedisLabs/rediscloud-go-api"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

// Planning a create never calls the API, so these steps exercise the schema as served through the mux.
func TestResourceRedisCloudAclRole_plan(t *testing.T) {
	t.Setenv(rediscloudApi.AccessKeyEnvVar, "unused")
	t.Setenv(rediscloudApi.SecretKeyEnvVar, "unused")

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: `
resource "rediscloud_acl_role" "test" {
  name = "plan-only"
  rule {
    name = "Read-Only"
    database {
      subscription = 1
      database     = 2
    }
    database {
      subscription = 3
      database     = 4
      regions      = ["us-east-1"]
    }
  }
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				Config: `
resource "rediscloud_acl_role" "test" {
  name = "plan-only"
}
`,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`(?i)rule`),
			},
			{
				Config: `
resource "rediscloud_acl_role" "test" {
  name = "plan-only"
  rule {
    name = "Read-Only"
  }
}
`,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`(?i)database`),
			},
		},
	})
}
