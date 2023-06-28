package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceRedisCloudAclRule_Default(t *testing.T) {
	// This rule already exists
	testName := "Read-Write"
	testRule := "+@all -@dangerous ~*"
	getRuleTerraform := fmt.Sprintf(getDefaultAclRuleDataSource, testName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      nil, // test doesn't create a resource, so don't need to check anything
		Steps: []resource.TestStep{
			{
				Config: getRuleTerraform,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"data.rediscloud_acl_rule.test", "id", regexp.MustCompile("^\\d*$")),
					resource.TestCheckResourceAttr("data.rediscloud_acl_rule.test", "name", testName),
					resource.TestCheckResourceAttr("data.rediscloud_acl_rule.test", "rule", testRule),
				),
			},
		},
	})
}

func TestAccDataSourceRedisCloudAclRule_Custom(t *testing.T) {
	testName := "custom-test-rule"
	testRule := "+@read ~*"
	createAndGetRuleTerraform := fmt.Sprintf(createAndGetCustomAclRuleTerraform, testName, testRule)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckAclRuleDestroy,
		Steps: []resource.TestStep{
			{
				Config: createAndGetRuleTerraform,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"data.rediscloud_acl_rule.test", "id", regexp.MustCompile("^\\d*$")),
					resource.TestCheckResourceAttr("data.rediscloud_acl_rule.test", "name", testName),
					resource.TestCheckResourceAttr("data.rediscloud_acl_rule.test", "rule", testRule),
				),
			},
		},
	})
}

const getDefaultAclRuleDataSource = `
data "rediscloud_acl_rule" "test" {
	name = "%s"
}
`

const createAndGetCustomAclRuleTerraform = `
resource "rediscloud_acl_rule" "test" {
	name = "%s"
	rule = "%s"
}

data "rediscloud_acl_rule" "test" {
	name = rediscloud_acl_rule.test.name
}
`
