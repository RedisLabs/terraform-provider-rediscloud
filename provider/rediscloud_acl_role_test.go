package provider

import (
	"context"
	"fmt"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"os"
	"regexp"
	"strconv"
	"testing"
)

func TestAccCreateReadUpdateImportDeleteAclRole_Flexible(t *testing.T) {

	prefix := acctest.RandomWithPrefix(testResourcePrefix)
	exampleCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")
	exampleSubscriptionName := prefix + "-subscription"
	exampleDatabasePassword := prefix + "aA.1"
	exampleRuleName := prefix + "-rule"

	testRoleName := prefix + "-test-role"
	testRoleNameUpdated := testRoleName + "-updated"

	testCreateTerraform := fmt.Sprintf(testAccResourceRedisCloudFlexibleDatabase, exampleCloudAccountName, exampleSubscriptionName, exampleDatabasePassword) +
		fmt.Sprintf(referencableRule, exampleRuleName) +
		fmt.Sprintf(testRole, testRoleName)

	testUpdateTerraform := fmt.Sprintf(testAccResourceRedisCloudFlexibleDatabase, exampleCloudAccountName, exampleSubscriptionName, exampleDatabasePassword) +
		fmt.Sprintf(referencableRule, exampleRuleName) +
		fmt.Sprintf(testRole, testRoleNameUpdated)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		// Sometimes after deletion, the entity 'flickers'
		CheckDestroy: nil,
		Steps: []resource.TestStep{
			// Test role creation including association with database
			{
				Config: testCreateTerraform,
				Check: resource.ComposeTestCheckFunc(
					// Test the resource
					resource.TestCheckResourceAttr("rediscloud_acl_role.test", "name", testRoleName),
					resource.TestCheckResourceAttr("rediscloud_acl_role.test", "rule.#", "1"),
					resource.TestCheckResourceAttr("rediscloud_acl_role.test", "rule.0.name", exampleRuleName),
					resource.TestCheckResourceAttr("rediscloud_acl_role.test", "rule.0.database.#", "1"),
					resource.TestMatchResourceAttr("rediscloud_acl_role.test", "rule.0.database.0.subscription", regexp.MustCompile("^\\d*$")),
					resource.TestMatchResourceAttr("rediscloud_acl_role.test", "rule.0.database.0.database", regexp.MustCompile("^\\d*$")),
					resource.TestCheckResourceAttr("rediscloud_acl_role.test", "rule.0.database.0.regions.#", "0"),

					// Test role exists
					func(s *terraform.State) error {
						r := s.RootModule().Resources["rediscloud_acl_role.test"]

						id, err := strconv.Atoi(r.Primary.ID)
						if err != nil {
							return fmt.Errorf("couldn't parse the role ID: %s", redis.StringValue(&r.Primary.ID))
						}

						client := testProvider.Meta().(*apiClient)
						role, err := client.client.Roles.Get(context.TODO(), id)
						if err != nil {
							return err
						}

						if redis.StringValue(role.Name) != testRoleName {
							return fmt.Errorf("unexpected name value: %s", redis.StringValue(role.Name))
						}

						return nil
					},

					// Test the datasource
					resource.TestMatchResourceAttr(
						"data.rediscloud_acl_role.test", "id", regexp.MustCompile("^\\d*$")),
					resource.TestCheckResourceAttr("data.rediscloud_acl_role.test", "name", testRoleName),
					resource.TestCheckResourceAttr("data.rediscloud_acl_role.test", "rule.#", "1"),
					resource.TestCheckResourceAttr("data.rediscloud_acl_role.test", "rule.0.name", exampleRuleName),
					resource.TestCheckResourceAttr("data.rediscloud_acl_role.test", "rule.0.database.#", "1"),
					resource.TestMatchResourceAttr("data.rediscloud_acl_role.test", "rule.0.database.0.subscription", regexp.MustCompile("^\\d*$")),
					resource.TestMatchResourceAttr("data.rediscloud_acl_role.test", "rule.0.database.0.database", regexp.MustCompile("^\\d*$")),
					resource.TestCheckResourceAttr("data.rediscloud_acl_role.test", "rule.0.database.0.regions.#", "0"),
				),
			},
			// Test role update
			{
				Config: testUpdateTerraform,
				Check: resource.ComposeTestCheckFunc(
					// Test the resource
					resource.TestCheckResourceAttr("rediscloud_acl_role.test", "name", testRoleNameUpdated),
					resource.TestCheckResourceAttr("rediscloud_acl_role.test", "rule.#", "1"),
					resource.TestCheckResourceAttr("rediscloud_acl_role.test", "rule.0.name", exampleRuleName),
					resource.TestCheckResourceAttr("rediscloud_acl_role.test", "rule.0.database.#", "1"),
					resource.TestMatchResourceAttr("rediscloud_acl_role.test", "rule.0.database.0.subscription", regexp.MustCompile("^\\d*$")),
					resource.TestMatchResourceAttr("rediscloud_acl_role.test", "rule.0.database.0.database", regexp.MustCompile("^\\d*$")),
					resource.TestCheckResourceAttr("rediscloud_acl_role.test", "rule.0.database.0.regions.#", "0"),

					// Test the datasource
					resource.TestMatchResourceAttr(
						"data.rediscloud_acl_role.test", "id", regexp.MustCompile("^\\d*$")),
					resource.TestCheckResourceAttr("data.rediscloud_acl_role.test", "name", testRoleNameUpdated),
					resource.TestCheckResourceAttr("data.rediscloud_acl_role.test", "rule.#", "1"),
					resource.TestCheckResourceAttr("data.rediscloud_acl_role.test", "rule.0.name", exampleRuleName),
					resource.TestCheckResourceAttr("data.rediscloud_acl_role.test", "rule.0.database.#", "1"),
					resource.TestMatchResourceAttr("data.rediscloud_acl_role.test", "rule.0.database.0.subscription", regexp.MustCompile("^\\d*$")),
					resource.TestMatchResourceAttr("data.rediscloud_acl_role.test", "rule.0.database.0.database", regexp.MustCompile("^\\d*$")),
					resource.TestCheckResourceAttr("data.rediscloud_acl_role.test", "rule.0.database.0.regions.#", "0"),
				),
			},
			// Test that the role is imported successfully
			{
				Config:            fmt.Sprintf(testRole, testRoleNameUpdated),
				ResourceName:      "rediscloud_acl_role.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

const referencableRule = `
resource "rediscloud_acl_rule" "example" {
    name = "%s"
    rule = "+@all"
}
`

const testRole = `
resource "rediscloud_acl_role" "test" {
	name = "%s"
	rule {
		name = rediscloud_acl_rule.example.name
		database {
			subscription = rediscloud_flexible_subscription.example.id
			database = rediscloud_flexible_database.example.db_id
		}
	}
}

data "rediscloud_acl_role" "test" {
	name = rediscloud_acl_role.test.name
}
`
