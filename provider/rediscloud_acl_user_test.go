package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func TestAccResourceRedisCloudAclUser_CRUDI(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

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

	testCreateTerraform := fmt.Sprintf(testAccResourceRedisCloudProDatabaseAcl, exampleCloudAccountName, exampleSubscriptionName, exampleDatabasePassword) +
		fmt.Sprintf(referencableRole, exampleRoleName) +
		fmt.Sprintf(testUser, testUserName, testUserPassword)

	// The User will be updated because the Role's name will have changed
	testUpdateTerraform := fmt.Sprintf(testAccResourceRedisCloudProDatabaseAcl, exampleCloudAccountName, exampleSubscriptionName, exampleDatabasePassword) +
		fmt.Sprintf(referencableRole, exampleRoleNameUpdated) +
		fmt.Sprintf(testUser, testUserName, testUserPassword)

	testNewNameTerraform := fmt.Sprintf(testAccResourceRedisCloudProDatabaseAcl, exampleCloudAccountName, exampleSubscriptionName, exampleDatabasePassword) +
		fmt.Sprintf(referencableRole, exampleRoleNameUpdated) +
		fmt.Sprintf(testUser, testUserNameUpdated, testUserPassword)

	testNewPasswordTerraform := fmt.Sprintf(testAccResourceRedisCloudProDatabaseAcl, exampleCloudAccountName, exampleSubscriptionName, exampleDatabasePassword) +
		fmt.Sprintf(referencableRole, exampleRoleNameUpdated) +
		fmt.Sprintf(testUser, testUserNameUpdated, testUserPasswordUpdated)

	identifier := ""

	const AclUserTest = "rediscloud_acl_user.test"
	const AclUserTestData = "data.rediscloud_acl_user.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             testAccCheckAclUserDestroy,
		Steps: []resource.TestStep{
			// Test user creation
			{
				Config: testCreateTerraform,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Test resource
					resource.TestCheckResourceAttr(AclUserTest, "name", testUserName),
					resource.TestCheckResourceAttr(AclUserTest, "role", exampleRoleName),

					// Take a snapshot of the ID
					func(s *terraform.State) error {
						r := s.RootModule().Resources[AclUserTest]
						identifier = r.Primary.ID
						return nil
					},

					// Test user exists
					func(s *terraform.State) error {
						r := s.RootModule().Resources[AclUserTest]

						id, err := strconv.Atoi(r.Primary.ID)
						if err != nil {
							return fmt.Errorf("couldn't parse the role ID: %s", redis.StringValue(&r.Primary.ID))
						}

						apiClient := sharedTestClient(t)
						user, err := apiClient.Client.Users.Get(context.TODO(), id)
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
						AclUserTestData, "id", regexp.MustCompile("^\\d*$")),
					resource.TestCheckResourceAttr(AclUserTestData, "name", testUserName),
					resource.TestCheckResourceAttr(AclUserTestData, "role", exampleRoleName),
				),
			},
			// Test user update, id should not have changed
			{
				Config: testUpdateTerraform,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Test resource
					resource.TestCheckResourceAttr(AclUserTest, "name", testUserName),
					resource.TestCheckResourceAttr(AclUserTest, "role", exampleRoleNameUpdated),

					func(s *terraform.State) error {
						r := s.RootModule().Resources[AclUserTest]
						if r.Primary.ID != identifier {
							return fmt.Errorf("entity should have the same identifier, but has changed from %s to %s", identifier, r.Primary.ID)
						}
						return nil
					},

					// Test datasource
					resource.TestMatchResourceAttr(
						AclUserTestData, "id", regexp.MustCompile("^\\d*$")),
					resource.TestCheckResourceAttr(AclUserTestData, "name", testUserName),
					resource.TestCheckResourceAttr(AclUserTestData, "role", exampleRoleNameUpdated),
				),
			},
			// Test user is updated successfully. A name change should forcibly generate a new entity with a new id
			// Take a snapshot of this new id
			{
				Config: testNewNameTerraform,
				Check: resource.ComposeAggregateTestCheckFunc(
					func(s *terraform.State) error {
						r := s.RootModule().Resources[AclUserTest]
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
				Check: resource.ComposeAggregateTestCheckFunc(
					func(s *terraform.State) error {
						r := s.RootModule().Resources[AclUserTest]
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
				ResourceName:            AclUserTest,
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

data "rediscloud_acl_user" "test" {
	name = rediscloud_acl_user.test.name
}
`

func testAccCheckAclUserDestroy(s *terraform.State) error {
	apiClient, err := getTestClient()
	if err != nil {
		return err
	}

	for _, r := range s.RootModule().Resources {
		if r.Type != "rediscloud_acl_user" {
			continue
		}

		id, err := strconv.Atoi(r.Primary.ID)
		if err != nil {
			return err
		}

		roles, err := apiClient.Client.Users.List(context.TODO())
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

const testAccResourceRedisCloudProDatabaseAcl = `
data "rediscloud_payment_method" "card" {
  card_type = "Visa"
  last_four_numbers = "5556"
}

data "rediscloud_cloud_account" "account" {
  exclude_internal_account = true
  provider_type = "AWS"
  name = "%s"
}

resource "rediscloud_subscription" "example" {
  name = "%s"
  payment_method_id = data.rediscloud_payment_method.card.id
  cloud_provider {
    provider = data.rediscloud_cloud_account.account.provider_type
    cloud_account_id = data.rediscloud_cloud_account.account.id
    region {
      region = "eu-west-1"
      networking_deployment_cidr = "10.0.0.0/24"
      preferred_availability_zones = ["eu-west-1a"]
    }
  }

  creation_plan {
    dataset_size_in_gb = 1
    quantity = 1
    replication = false
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 1000
  }
}

resource "rediscloud_subscription_database" "example" {
    subscription_id = rediscloud_subscription.example.id
    name = "example"
    protocol = "redis"
    dataset_size_in_gb = 3
    data_persistence = "none"
    data_eviction = "allkeys-random"
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 1000
    password = "%s"
    support_oss_cluster_api = false
    external_endpoint_for_oss_cluster_api = false
    replication = false
    average_item_size_in_bytes = 0
    client_ssl_certificate = ""
    periodic_backup_path = ""
	enable_default_user = true
    redis_version = "7.4"

    alert {
        name = "dataset-size"
        value = 1
    }

    modules = [
        {
          name = "RedisBloom"
        }
    ]

	tags = {
		"market" = "emea"
		"material" = "cardboard"
	}
}
`
