package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceRedisCloudAclUser_Default(t *testing.T) {

	prefix := acctest.RandomWithPrefix(testResourcePrefix)
	exampleCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")
	exampleSubscriptionName := prefix + "-subscription"
	exampleDatabaseName := prefix + "-database"
	exampleDatabasePassword := prefix + "aA.1"
	exampleRoleName := prefix + "-role"

	testName := prefix + "-test-user"
	testRoleName := exampleRoleName
	testPassword := prefix + "aA.1"

	createAndGetUserTerraform := fmt.Sprintf(
		testAccDatasourceRedisCloudUserDataSource,
		exampleCloudAccountName,
		exampleSubscriptionName,
		exampleDatabaseName,
		exampleDatabasePassword,
		exampleRoleName,
		testName,
		testPassword,
	)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckAclUserDestroy,
		Steps: []resource.TestStep{
			{
				Config: createAndGetUserTerraform,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"data.rediscloud_acl_user.test", "id", regexp.MustCompile("^\\d*$")),
					resource.TestCheckResourceAttr("data.rediscloud_acl_user.test", "name", testName),
					resource.TestCheckResourceAttr("data.rediscloud_acl_user.test", "role", testRoleName),
				),
			},
		},
	})
}

const testAccDatasourceRedisCloudUserDataSource = `
data "rediscloud_payment_method" "card" {
  card_type = "Visa"
}

data "rediscloud_cloud_account" "account" {
  exclude_internal_account = true
  provider_type = "AWS" 
  name = "%s"
}

resource "rediscloud_subscription" "example" {

  name = "%s"
  payment_method_id = data.rediscloud_payment_method.card.id
  memory_storage = "ram"

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
    memory_limit_in_gb = 1
    quantity = 1
    replication=false
    support_oss_cluster_api=true
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 1000
	modules = ["RedisJSON", "RedisBloom"]
  }
}

resource "rediscloud_subscription_database" "example" {
    subscription_id              = rediscloud_subscription.example.id
    name                         = "%s"
    protocol                     = "redis"
    memory_limit_in_gb           = 1
    data_persistence             = "none"
    throughput_measurement_by    = "operations-per-second"
    throughput_measurement_value = 1000
	password                     = "%s"
	support_oss_cluster_api	     = true
	replication				     = false
}

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

resource "rediscloud_acl_user" "test" {
	name = "%s"
	role = rediscloud_acl_role.example.name
	password = "%s"
}

data "rediscloud_acl_user" "test" {
	name = rediscloud_acl_user.test.name
}
`
