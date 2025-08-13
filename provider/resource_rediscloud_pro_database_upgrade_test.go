package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceRedisCloudProDatabase_Upgrade(t *testing.T) {

	testAccRequiresEnvVar(t, "EXECUTE_TESTS")

	name := acctest.RandomWithPrefix(testResourcePrefix)
	const resourceName = "rediscloud_subscription_database.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			// Test database and replica database creation
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudProDatabaseUpgrade, testCloudAccountName, name, "7.2"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "example"),
					resource.TestCheckResourceAttr(resourceName, "protocol", "redis"),
					resource.TestCheckResourceAttr(resourceName, "redis_version", "7.2"),
				),
			},
			// Test database is updated successfully
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudProDatabaseUpgrade, testCloudAccountName, name, "7.4"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "example"),
					resource.TestCheckResourceAttr(resourceName, "protocol", "redis"),
					resource.TestCheckResourceAttr(resourceName, "redis_version", "7.4"),
				),
			},
		},
	})
}

// Create and Read tests
// TF config for provisioning a new database
const testAccResourceRedisCloudProDatabaseUpgrade = `
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
  memory_storage = "ram"

  allowlist {
    cidrs = ["192.168.0.0/16"]
    security_group_ids = []
  }

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
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 1000
    quantity = 1
    replication=false
    support_oss_cluster_api=false
    modules = []
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
    support_oss_cluster_api = false 
    external_endpoint_for_oss_cluster_api = false
    replication = false
    average_item_size_in_bytes = 0
    client_ssl_certificate = "" 
    periodic_backup_path = ""
	enable_default_user = true
    redis_version = %s

    alert {
        name = "dataset-size"
        value = 1
    }

	tags = {
		"market" = "emea"
		"material" = "cardboard"
	}
} 
`
