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

func TestAccResourceRedisCloudAclUser_CRUDI(t *testing.T) {

	prefix := acctest.RandomWithPrefix(testResourcePrefix)
	exampleCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")
	exampleSubscriptionName := prefix + "-subscription"
	exampleDatabasePassword := prefix + "aA.1"

	exampleRoleName := prefix + "-role"
	exampleRoleNameUpdated := exampleRoleName + "-updated"

	testUserName := prefix + "-test-user"
	testUserNameUpdated := testUserName + "-updated"
	testUserPassword := prefix + "aA.1"
	testUserPasswordUpdated := testUserPassword + "-updated"

	testCreateTerraform := fmt.Sprintf(testAccResourceRedisCloudFlexibleDatabase, exampleCloudAccountName, exampleSubscriptionName, exampleDatabasePassword) +
		fmt.Sprintf(referencableRole, exampleRoleName) +
		fmt.Sprintf(testUser, testUserName, testUserPassword)

	// The User will be updated because the Role's name will have changed
	testUpdateTerraform := fmt.Sprintf(testAccResourceRedisCloudFlexibleDatabase, exampleCloudAccountName, exampleSubscriptionName, exampleDatabasePassword) +
		fmt.Sprintf(referencableRole, exampleRoleNameUpdated) +
		fmt.Sprintf(testUser, testUserName, testUserPassword)

	testNewNameTerraform := fmt.Sprintf(testAccResourceRedisCloudFlexibleDatabase, exampleCloudAccountName, exampleSubscriptionName, exampleDatabasePassword) +
		fmt.Sprintf(referencableRole, exampleRoleNameUpdated) +
		fmt.Sprintf(testUser, testUserNameUpdated, testUserPassword)

	testNewPasswordTerraform := fmt.Sprintf(testAccResourceRedisCloudFlexibleDatabase, exampleCloudAccountName, exampleSubscriptionName, exampleDatabasePassword) +
		fmt.Sprintf(referencableRole, exampleRoleNameUpdated) +
		fmt.Sprintf(testUser, testUserNameUpdated, testUserPasswordUpdated)

	identifier := ""

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckAclUserDestroy,
		Steps: []resource.TestStep{
			// Test user creation
			{
				Config: testCreateTerraform,
				Check: resource.ComposeTestCheckFunc(
					// Test resource
					resource.TestCheckResourceAttr("rediscloud_acl_user.test", "name", testUserName),
					resource.TestCheckResourceAttr("rediscloud_acl_user.test", "role", exampleRoleName),

					// Take a snapshot of the ID
					func(s *terraform.State) error {
						r := s.RootModule().Resources["rediscloud_acl_user.test"]
						identifier = r.Primary.ID
						return nil
					},

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

						if redis.StringValue(user.Name) != testUserName {
							return fmt.Errorf("unexpected name value: %s", redis.StringValue(user.Name))
						}
						if redis.StringValue(user.Role) != exampleRoleName {
							return fmt.Errorf("unexpected role value: %s", redis.StringValue(user.Role))
						}

						return nil
					},

					// Test datasource
					resource.TestMatchResourceAttr(
						"data.rediscloud_acl_user.test", "id", regexp.MustCompile("^\\d*$")),
					resource.TestCheckResourceAttr("data.rediscloud_acl_user.test", "name", testUserName),
					resource.TestCheckResourceAttr("data.rediscloud_acl_user.test", "role", exampleRoleName),
				),
			},
			// Test user update, id should not have changed
			{
				Config: testUpdateTerraform,
				Check: resource.ComposeTestCheckFunc(
					// Test resource
					resource.TestCheckResourceAttr("rediscloud_acl_user.test", "name", testUserName),
					resource.TestCheckResourceAttr("rediscloud_acl_user.test", "role", exampleRoleNameUpdated),

					func(s *terraform.State) error {
						r := s.RootModule().Resources["rediscloud_acl_user.test"]
						if r.Primary.ID != identifier {
							return fmt.Errorf("entity should have the same identifier, but has changed from %s to %s", identifier, r.Primary.ID)
						}
						return nil
					},

					// Test datasource
					resource.TestMatchResourceAttr(
						"data.rediscloud_acl_user.test", "id", regexp.MustCompile("^\\d*$")),
					resource.TestCheckResourceAttr("data.rediscloud_acl_user.test", "name", testUserName),
					resource.TestCheckResourceAttr("data.rediscloud_acl_user.test", "role", exampleRoleNameUpdated),
				),
			},
			// Test user is updated successfully. A name change should forcibly generate a new entity with a new id
			// Take a snapshot of this new id
			{
				Config: testNewNameTerraform,
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						r := s.RootModule().Resources["rediscloud_acl_user.test"]
						if r.Primary.ID == identifier {
							return fmt.Errorf("entity should have a new identifier, but is still: %s", identifier)
						}
						identifier = r.Primary.ID
						return nil
					},
				),
			},
			// Test user is updated successfully. A password change should forcibly generate a new entity with a new id
			{
				Config: testNewPasswordTerraform,
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						r := s.RootModule().Resources["rediscloud_acl_user.test"]
						if r.Primary.ID == identifier {
							return fmt.Errorf("entity should have a new identifier, but is still: %s", identifier)
						}
						return nil
					},
				),
			},
			// Test that the user is imported successfully
			{
				Config:                  fmt.Sprintf(testUser, testUserNameUpdated, testUserPasswordUpdated),
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
	rule {
		name = "Read-Only"
		database {
			subscription = rediscloud_flexible_subscription.example.id
			database = rediscloud_flexible_database.example.db_id
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

data "rediscloud_acl_user" "test" {
	name = rediscloud_acl_user.test.name
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
