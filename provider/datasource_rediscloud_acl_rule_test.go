package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func TestAccDataSourceRedisCloudAclRule_ForDefaultRule(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	// This rule already exists
	const testName = "Read-Write"
	const testRule = "+@all -@dangerous ~*"
	getRuleTerraform := fmt.Sprintf(getDefaultDatasourceAclRuleDataSource, testName)

	const AclRuleTest = "data.rediscloud_acl_rule.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:      nil, // test doesn't create a resource, so don't need to check anything
		Steps: []resource.TestStep{
			{
				Config: getRuleTerraform,
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

const getDefaultDatasourceAclRuleDataSource = `
data "rediscloud_acl_rule" "test" {
	name = "%s"
}
`
