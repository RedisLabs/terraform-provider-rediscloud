package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"os"
	"testing"
)

// Tests the multi-modules feature in a database resource.
func TestAccResourceRedisCloudDatabase_MultiModules(t *testing.T) {
	name := acctest.RandomWithPrefix(testResourcePrefix)
	dbName := "db-multi-modules"
	resourceName := "rediscloud_database.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudDatabaseMultiModules, testCloudAccountName, name, dbName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", dbName),
					resource.TestCheckResourceAttr(resourceName, "module.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "module.0.name", "RedisBloom"),
					resource.TestCheckResourceAttr(resourceName, "module.1.name", "RedisJSON"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

const multiModulesSubscriptionBoilerplate = `
data "rediscloud_payment_method" "card" {
  card_type = "Visa"
}

data "rediscloud_cloud_account" "account" {
  exclude_internal_account = true
  provider_type            = "AWS"
  name                     = "%s"
}

resource "rediscloud_subscription" "example" {

  name              = "%s"
  payment_method_id = data.rediscloud_payment_method.card.id
  memory_storage    = "ram"

  allowlist {
    cidrs = ["192.168.0.0/16"]
  }

  cloud_provider {
    provider         = data.rediscloud_cloud_account.account.provider_type
    cloud_account_id = data.rediscloud_cloud_account.account.id
    region {
      region                       = "eu-west-1"
      networking_deployment_cidr   = "10.0.0.0/24"
      preferred_availability_zones = ["eu-west-1a"]
    }
  }

  creation_plan {
    average_item_size_in_bytes   = 1
    memory_limit_in_gb           = 1
    quantity                     = 1
    replication                  = false
    support_oss_cluster_api      = false
    throughput_measurement_by    = "operations-per-second"
    throughput_measurement_value = 1000
    modules                      = ["RedisJSON", "RedisBloom"]
  }
}
`

const testAccResourceRedisCloudDatabaseMultiModules = multiModulesSubscriptionBoilerplate + `
resource "rediscloud_database" "example" {
    subscription_id              = rediscloud_subscription.example.id
	name                         = "%s"
    protocol                     = "redis"
    memory_limit_in_gb           = 1
    data_persistence             = "none"
    throughput_measurement_by    = "operations-per-second"
    throughput_measurement_value = 1000

    module {
      name  = "RedisJSON"
    }

    module {
      name  = "RedisBloom"
    }

    alert {
      name  = "latency"
      value = 11
    }
}
`
