package aclrule_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

func TestAccDataSourceRedisCloudAclRule_ForDefaultRule(t *testing.T) {

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
				ConfigFile: config.StaticFile("./testdata/datasource_basic.tf"),
				ConfigVariables: config.Variables{
					"name": config.StringVariable(testName),
				},
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
