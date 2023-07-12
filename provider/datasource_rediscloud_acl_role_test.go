package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceRedisCloudAclRole_Default(t *testing.T) {

	prefix := acctest.RandomWithPrefix(testResourcePrefix)
	exampleCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")
	exampleSubscriptionName := prefix + "-subscription"
	exampleDatabaseName := prefix + "-database"
	exampleDatabasePassword := prefix + "aA.1"

	testName := prefix + "-test-role"

	createAndGetRoleTerraform := fmt.Sprintf(
		testAccDatasourceRedisCloudRoleDataSource,
		exampleCloudAccountName,
		exampleSubscriptionName,
		exampleDatabaseName,
		exampleDatabasePassword,
		testName,
	)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      nil, // test doesn't create a resource at the moment, so don't need to check anything
		Steps: []resource.TestStep{
			{
				Config: createAndGetRoleTerraform,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(
						"data.rediscloud_acl_role.test", "id", regexp.MustCompile("^\\d*$")),
					resource.TestCheckResourceAttr("data.rediscloud_acl_role.test", "name", testName),
					resource.TestCheckResourceAttr("data.rediscloud_acl_role.test", "rules.#", "1"),
					resource.TestCheckResourceAttr("data.rediscloud_acl_role.test", "rules.0.name", "Read-Only"),
					resource.TestCheckResourceAttr("data.rediscloud_acl_role.test", "rules.0.databases.#", "1"),
					resource.TestMatchResourceAttr("data.rediscloud_acl_role.test", "rules.0.databases.0.subscription", regexp.MustCompile("^\\d*$")),
					resource.TestMatchResourceAttr("data.rediscloud_acl_role.test", "rules.0.databases.0.database", regexp.MustCompile("^\\d*$")),
					resource.TestCheckResourceAttr("data.rediscloud_acl_role.test", "rules.0.databases.0.regions.#", "0"),
				),
			},
		},
	})
}

const testAccDatasourceRedisCloudRoleDataSource = `
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

resource "rediscloud_acl_role" "test" {
	name = "%s"
	rules {
		name = "Read-Only"
		databases {
			subscription = rediscloud_subscription.example.id
			database = rediscloud_subscription_database.example.db_id
		}
	}
}

data "rediscloud_acl_role" "test" {
	name = rediscloud_acl_role.test.name
}
`
