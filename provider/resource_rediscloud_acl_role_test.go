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

	testCreateTerraform := fmt.Sprintf(testAccResourceRedisCloudSubscriptionDatabase, exampleCloudAccountName, exampleSubscriptionName, exampleDatabasePassword) +
		fmt.Sprintf(referencableRule, exampleRuleName) +
		fmt.Sprintf(testRole, testRoleName)

	testUpdateTerraform := fmt.Sprintf(testAccResourceRedisCloudSubscriptionDatabase, exampleCloudAccountName, exampleSubscriptionName, exampleDatabasePassword) +
		fmt.Sprintf(referencableRule, exampleRuleName) +
		fmt.Sprintf(testRole, testRoleName+"-updated")

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
				),
			},
			// Test role is updated successfully
			{
				Config: testUpdateTerraform,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("rediscloud_acl_role.test", "name", testRoleName+"-updated"),
					resource.TestCheckResourceAttr("rediscloud_acl_role.test", "rule.#", "1"),
					resource.TestCheckResourceAttr("rediscloud_acl_role.test", "rule.0.name", exampleRuleName),
					resource.TestCheckResourceAttr("rediscloud_acl_role.test", "rule.0.database.#", "1"),
					resource.TestMatchResourceAttr("rediscloud_acl_role.test", "rule.0.database.0.subscription", regexp.MustCompile("^\\d*$")),
					resource.TestMatchResourceAttr("rediscloud_acl_role.test", "rule.0.database.0.database", regexp.MustCompile("^\\d*$")),
					resource.TestCheckResourceAttr("rediscloud_acl_role.test", "rule.0.database.0.regions.#", "0"),
				),
			},
			// Test that the role is imported successfully
			{
				Config:            fmt.Sprintf(testRole, testRoleName+"_updated"),
				ResourceName:      "rediscloud_acl_role.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// Adds a considerable time overhead without giving us any useful information
//func TestAccCreateReadUpdateImportDeleteAclRole_ActiveActive(t *testing.T) {
//
//	prefix := acctest.RandomWithPrefix(testResourcePrefix)
//	exampleSubscriptionName := prefix + "-subscription"
//	exampleDatabaseName := prefix + "-database"
//	exampleDatabasePassword := prefix + "aA.1"
//
//	testRoleName := prefix + "-test-role"
//
//	testCreateTerraform := fmt.Sprintf(testAccResourceRedisCloudActiveActiveSubscriptionDatabase, exampleSubscriptionName, exampleDatabaseName, exampleDatabasePassword) +
//		fmt.Sprintf(testAADatabaseRole, testRoleName)
//
//	testUpdateTerraform := fmt.Sprintf(testAccResourceRedisCloudActiveActiveSubscriptionDatabase, exampleSubscriptionName, exampleDatabaseName, exampleDatabasePassword) +
//		fmt.Sprintf(testAADatabaseRole, testRoleName+"-updated")
//
//	resource.ParallelTest(t, resource.TestCase{
//		PreCheck:          func() { testAccPreCheck(t) },
//		ProviderFactories: providerFactories,
//		CheckDestroy:      testAccCheckAclRoleDestroy,
//		Steps: []resource.TestStep{
//			// Test role creation including association with AA database
//			{
//				Config: testCreateTerraform,
//				Check: resource.ComposeTestCheckFunc(
//					resource.TestCheckResourceAttr("rediscloud_acl_role.test", "name", testRoleName),
//					resource.TestCheckResourceAttr("rediscloud_acl_role.test", "rule.#", "1"),
//					resource.TestCheckResourceAttr("rediscloud_acl_role.test", "rule.0.name", "Read-Only"),
//					resource.TestCheckResourceAttr("rediscloud_acl_role.test", "rule.0.database.#", "1"),
//					resource.TestMatchResourceAttr("rediscloud_acl_role.test", "rule.0.database.0.subscription", regexp.MustCompile("^\\d*$")),
//					resource.TestMatchResourceAttr("rediscloud_acl_role.test", "rule.0.database.0.database", regexp.MustCompile("^\\d*$")),
//					resource.TestCheckResourceAttr("rediscloud_acl_role.test", "rule.0.database.0.regions.#", "2"),
//					resource.TestCheckTypeSetElemAttr("rediscloud_acl_role.test", "rule.0.database.0.regions.*", "us-east-1"),
//					resource.TestCheckTypeSetElemAttr("rediscloud_acl_role.test", "rule.0.database.0.regions.*", "us-east-2"),
//
//					// Test role exist
//					func(s *terraform.State) error {
//						r := s.RootModule().Resources["rediscloud_acl_role.test"]
//
//						id, err := strconv.Atoi(r.Primary.ID)
//						if err != nil {
//							return fmt.Errorf("couldn't parse the role ID: %s", redis.StringValue(&r.Primary.ID))
//						}
//
//						client := testProvider.Meta().(*apiClient)
//						role, err := client.client.Roles.Get(context.TODO(), id)
//						if err != nil {
//							return err
//						}
//
//						if redis.StringValue(role.Name) != testRoleName {
//							return fmt.Errorf("unexpected name value: %s", redis.StringValue(role.Name))
//						}
//
//						return nil
//					},
//				),
//			},
//			// Test role is updated successfully
//			{
//				Config: testUpdateTerraform,
//				Check: resource.ComposeTestCheckFunc(
//					resource.TestCheckResourceAttr("rediscloud_acl_role.test", "name", testRoleName+"-updated"),
//					resource.TestCheckResourceAttr("rediscloud_acl_role.test", "rule.#", "1"),
//					resource.TestCheckResourceAttr("rediscloud_acl_role.test", "rule.0.name", "Read-Only"),
//					resource.TestCheckResourceAttr("rediscloud_acl_role.test", "rule.0.database.#", "1"),
//					resource.TestMatchResourceAttr("rediscloud_acl_role.test", "rule.0.database.0.subscription", regexp.MustCompile("^\\d*$")),
//					resource.TestMatchResourceAttr("rediscloud_acl_role.test", "rule.0.database.0.database", regexp.MustCompile("^\\d*$")),
//					resource.TestCheckResourceAttr("rediscloud_acl_role.test", "rule.0.database.0.regions.#", "2"),
//					resource.TestCheckTypeSetElemAttr("rediscloud_acl_role.test", "rule.0.database.0.regions.*", "us-east-1"),
//					resource.TestCheckTypeSetElemAttr("rediscloud_acl_role.test", "rule.0.database.0.regions.*", "us-east-2"),
//				),
//			},
//			// Test that the role is imported successfully
//			{
//				Config:            fmt.Sprintf(testRole, testRoleName+"_updated"),
//				ResourceName:      "rediscloud_acl_role.test",
//				ImportState:       true,
//				ImportStateVerify: true,
//			},
//		},
//	})
//}

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
			subscription = rediscloud_subscription.example.id
			database = rediscloud_subscription_database.example.db_id
		}
	}
}
`

//const testAADatabaseRole = `
//resource "rediscloud_acl_role" "test" {
//	name = "%s"
//	rule {
//		name = "Read-Only"
//		database {
//			subscription = rediscloud_active_active_subscription.example.id
//			database = rediscloud_active_active_subscription_database.example.db_id
//			regions = [
//        		for r in rediscloud_active_active_subscription_database.example.override_region : r.name
//      		]
//		}
//	}
//}
//`

//func testAccCheckAclRoleDestroy(s *terraform.State) error {
//	client := testProvider.Meta().(*apiClient)
//
//	for _, r := range s.RootModule().Resources {
//		if r.Type != "rediscloud_acl_role" {
//			continue
//		}
//
//		id, err := strconv.Atoi(r.Primary.ID)
//		if err != nil {
//			return err
//		}
//
//		roles, err := client.client.Roles.List(context.TODO())
//		if err != nil {
//			return err
//		}
//
//		for _, role := range roles {
//			if redis.IntValue(role.ID) == id {
//				return fmt.Errorf("role %d still exists", id)
//			}
//		}
//	}
//
//	return nil
//}
