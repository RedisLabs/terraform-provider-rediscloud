package provider

import (
	"context"
	"fmt"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"regexp"
	"strconv"
	"testing"
)

func TestAccRedisCloudAclRule_DataSourceForDefaultRule(t *testing.T) {
	// This rule already exists
	const testName = "Read-Write"
	const testRule = "+@all -@dangerous ~*"
	getRuleTerraform := fmt.Sprintf(getDefaultAclRuleDataSource, testName)

	const AclRuleTest = "data.rediscloud_acl_rule.test"
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
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

func TestAccRedisCloudAclRule_CRUDI(t *testing.T) {

	prefix := acctest.RandomWithPrefix(testResourcePrefix)
	testName := prefix + "-test-rule"
	const testRule = "+@all"

	testNameUpdated := testName + "_updated_name"
	const testRuleUpdated = testRule + " -@dangerous ~*"

	testCreateTerraform := fmt.Sprintf(testRedisRule, testName, testRule)
	testUpdateRuleTerraform := fmt.Sprintf(testRedisRule, testName, testRuleUpdated)
	testUpdateNameTerraform := fmt.Sprintf(testRedisRule, testNameUpdated, testRuleUpdated)

	const AclRuleTest = "rediscloud_acl_rule.test"
	const AclRuleTestData = "data.rediscloud_acl_rule.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckAclRuleDestroy,
		Steps: []resource.TestStep{
			// Test rule creation
			{
				Config: testCreateTerraform,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Test the resource
					resource.TestCheckResourceAttr(AclRuleTest, "name", testName),
					resource.TestCheckResourceAttr(AclRuleTest, "rule", testRule),

					// Test rule exists
					func(s *terraform.State) error {
						r := s.RootModule().Resources[AclRuleTest]

						var err error
						id, err := strconv.Atoi(r.Primary.ID)
						if err != nil {
							return fmt.Errorf("couldn't parse the rule ID: %s", redis.StringValue(&r.Primary.ID))
						}

						client := testProvider.Meta().(*apiClient)
						rule, err := client.client.RedisRules.Get(context.TODO(), id)
						if err != nil {
							return err
						}

						if redis.StringValue(rule.Name) != testName {
							return fmt.Errorf("unexpected name value: %s", redis.StringValue(rule.Name))
						}
						if redis.StringValue(rule.ACL) != testRule {
							return fmt.Errorf("unexpected rule value: %s", redis.StringValue(rule.ACL))
						}
						return nil
					},

					// Test the datasource
					resource.TestMatchResourceAttr(
						AclRuleTestData, "id", regexp.MustCompile("^\\d*$")),
					resource.TestCheckResourceAttr(AclRuleTestData, "name", testName),
					resource.TestCheckResourceAttr(AclRuleTestData, "rule", testRule),
				),
			},
			{
				Config: testUpdateRuleTerraform,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Test the resource
					resource.TestCheckResourceAttr("rediscloud_acl_rule.test", "name", testName),
					resource.TestCheckResourceAttr("rediscloud_acl_rule.test", "rule", testRuleUpdated),
					// Test the datasource
					resource.TestMatchResourceAttr(
						"data.rediscloud_acl_rule.test", "id", regexp.MustCompile("^\\d*$")),
					resource.TestCheckResourceAttr("data.rediscloud_acl_rule.test", "name", testName),
					resource.TestCheckResourceAttr("data.rediscloud_acl_rule.test", "rule", testRuleUpdated),
				),
			},
			{
				Config: testUpdateNameTerraform,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Test the resource
					resource.TestCheckResourceAttr(AclRuleTest, "name", testNameUpdated),
					resource.TestCheckResourceAttr(AclRuleTest, "rule", testRuleUpdated),
					// Test the datasource
					resource.TestMatchResourceAttr(
						AclRuleTestData, "id", regexp.MustCompile("^\\d*$")),
					resource.TestCheckResourceAttr(AclRuleTestData, "name", testNameUpdated),
					resource.TestCheckResourceAttr(AclRuleTestData, "rule", testRuleUpdated),
				),
			},
			// Test full update
			{
				Config: testCreateTerraform,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Test the resource
					resource.TestCheckResourceAttr("rediscloud_acl_rule.test", "name", testName),
					resource.TestCheckResourceAttr("rediscloud_acl_rule.test", "rule", testRule),
					// Test the datasource
					resource.TestMatchResourceAttr(
						"data.rediscloud_acl_rule.test", "id", regexp.MustCompile("^\\d*$")),
					resource.TestCheckResourceAttr("data.rediscloud_acl_rule.test", "name", testName),
					resource.TestCheckResourceAttr("data.rediscloud_acl_rule.test", "rule", testRule),
				),
			},
			// Test that that rule is imported successfully
			{
				Config:            fmt.Sprintf(testRedisRule, testNameUpdated, testRuleUpdated),
				ResourceName:      AclRuleTest,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

}

const getDefaultAclRuleDataSource = `
data "rediscloud_acl_rule" "test" {
	name = "%s"
}
`

const testRedisRule = `
resource "rediscloud_acl_rule" "test" {
    name = "%s"
    rule = "%s"
}

data "rediscloud_acl_rule" "test" {
	name = rediscloud_acl_rule.test.name
}
`

func testAccCheckAclRuleDestroy(s *terraform.State) error {
	client := testProvider.Meta().(*apiClient)

	for _, r := range s.RootModule().Resources {
		if r.Type != "rediscloud_acl_rule" {
			continue
		}

		id, err := strconv.Atoi(r.Primary.ID)
		if err != nil {
			return err
		}

		rules, err := client.client.RedisRules.List(context.TODO())
		if err != nil {
			return err
		}

		for _, rule := range rules {
			if redis.IntValue(rule.ID) == id {
				return fmt.Errorf("rule %d still exists", id)
			}
		}
	}

	return nil
}
