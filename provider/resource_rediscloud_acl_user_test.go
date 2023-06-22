package provider

import (
	"context"
	"fmt"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"os"
	"strconv"
	"testing"
)

func TestAccResourceRedisCloudAclUser_CRUDI(t *testing.T) {

	prefix := acctest.RandomWithPrefix(testResourcePrefix)
	exampleCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")
	exampleSubscriptionName := prefix + "-subscription"
	exampleDatabasePassword := prefix + "aA.1"
	exampleRoleName := prefix + "-role"

	testName := prefix + "-test-user"
	testPassword := prefix + "aA.1"

	testCreateTerraform := fmt.Sprintf(testAccResourceRedisCloudSubscriptionDatabase, exampleCloudAccountName, exampleSubscriptionName, exampleDatabasePassword) +
		fmt.Sprintf(referencableRole, exampleRoleName) +
		fmt.Sprintf(testUser, testName, testPassword)

	testUpdateTerraform := fmt.Sprintf(testAccResourceRedisCloudSubscriptionDatabase, exampleCloudAccountName, exampleSubscriptionName, exampleDatabasePassword) +
		fmt.Sprintf(referencableRole, exampleRoleName) +
		fmt.Sprintf(testUser, testName, testPassword+"-updated")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckAclUserDestroy,
		Steps: []resource.TestStep{
			// Test user creation
			{
				Config: testCreateTerraform,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("rediscloud_acl_user.test", "name", testName),
					resource.TestCheckResourceAttr("rediscloud_acl_user.test", "role", exampleRoleName),

					// Test user exists
					func(s *terraform.State) error {
						r := s.RootModule().Resources["rediscloud_acl_user.test"]

						id, err := strconv.Atoi(r.Primary.ID)
						if err != nil {
							return fmt.Errorf("couldn't parse the role ID: %s", redis.StringValue(&r.Primary.ID))
						}

						client := testProvider.Meta().(*apiClient)
						user, err := client.client.Users.Get(context.TODO(), id)
						if err != nil {
							return err
						}

						if redis.StringValue(user.Name) != testName {
							return fmt.Errorf("unexpected name value: %s", redis.StringValue(user.Name))
						}
						if redis.StringValue(user.Role) != exampleRoleName {
							return fmt.Errorf("unexpected role value: %s", redis.StringValue(user.Role))
						}

						return nil
					},
				),
			},
			// Test user is updated successfully
			{
				Config: testUpdateTerraform,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("rediscloud_acl_user.test", "name", testName),
					resource.TestCheckResourceAttr("rediscloud_acl_user.test", "role", exampleRoleName),
					// (can't really tell about password, no feedback on password change)
				),
			},
			// Test that the user is imported successfully
			{
				Config:                  fmt.Sprintf(testUser, testName+"_updated", testPassword+"-updated"),
				ResourceName:            "rediscloud_acl_user.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
		},
	})
}

const referencableRole = `
resource "rediscloud_acl_role" "example" {
    name = "%s"
	rules {
		name = "Read-Only"
		databases {
			subscription = rediscloud_subscription.example.id
			database = rediscloud_subscription_database.example.db_id
		}
	}
}
`

const testUser = `
resource "rediscloud_acl_user" "test" {
	name = "%s"
	role = rediscloud_acl_role.example.name
	password = "%s"
}
`

func testAccCheckAclUserDestroy(s *terraform.State) error {
	client := testProvider.Meta().(*apiClient)

	for _, r := range s.RootModule().Resources {
		if r.Type != "rediscloud_acl_user" {
			continue
		}

		id, err := strconv.Atoi(r.Primary.ID)
		if err != nil {
			return err
		}

		roles, err := client.client.Users.List(context.TODO())
		if err != nil {
			return err
		}

		for _, role := range roles {
			if redis.IntValue(role.ID) == id {
				return fmt.Errorf("user %d still exists", id)
			}
		}
	}

	return nil
}
