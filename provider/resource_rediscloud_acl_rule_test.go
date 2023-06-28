package provider

import (
	"context"
	"fmt"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"strconv"
	"testing"
)

func TestAccResourceRedisCloudAclRule_CRUDI(t *testing.T) {

	prefix := acctest.RandomWithPrefix(testResourcePrefix)
	testName := prefix + "-test-rule"
	testRule := "+@all"

	testCreateTerraform := fmt.Sprintf(testRedisRule, testName, testRule)
	testUpdateTerraform := fmt.Sprintf(testRedisRule, testName+"_updated_name", testRule+" -@dangerous ~*")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckAclRuleDestroy,
		Steps: []resource.TestStep{
			// Test rule creation
			{
				Config: testCreateTerraform,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("rediscloud_acl_rule.test", "name", testName),
					resource.TestCheckResourceAttr("rediscloud_acl_rule.test", "rule", testRule),

					// Test rule exists
					func(s *terraform.State) error {
						r := s.RootModule().Resources["rediscloud_acl_rule.test"]

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
				),
			},
			// Test rule update
			{
				Config: testUpdateTerraform,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("rediscloud_acl_rule.test", "name", testName+"_updated_name"),
					resource.TestCheckResourceAttr("rediscloud_acl_rule.test", "rule", testRule+" -@dangerous ~*"),
				),
			},
			// Test that that rule is imported successfully
			{
				Config:            fmt.Sprintf(testRedisRule, testName+"_updated_name", testRule+" -@dangerous ~*"),
				ResourceName:      "rediscloud_acl_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

}

const testRedisRule = `
resource "rediscloud_acl_rule" "test" {
    name = "%s"
    rule = "%s"
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
