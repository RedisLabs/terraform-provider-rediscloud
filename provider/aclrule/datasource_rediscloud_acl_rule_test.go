package aclrule_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func TestAccDataSourceRedisCloudAclRule_ForDefaultRule(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	// This rule already exists
	const testName = "Read-Write"
	const testRule = "+@all -@dangerous ~*"

	const AclRuleTest = "data.rediscloud_acl_rule.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testhelpers.BasicPreCheck(t) },
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             nil, // test doesn't create a resource, so don't need to check anything
		Steps: []resource.TestStep{
			{
				Config: utils.RenderTestConfig(t, "./testdata/datasource_basic.tf", map[string]string{
					"__NAME__": testName,
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(
						AclRuleTest, "id", regexp.MustCompile("^\\d*$")),
					resource.TestCheckResourceAttr(AclRuleTest, "name", testName),
					resource.TestCheckResourceAttr(AclRuleTest, "rule", testRule),
				),
			},
		},
	})
}
