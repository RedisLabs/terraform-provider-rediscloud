package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceRedisCloudFlexibleDatabase_basic(t *testing.T) {
	name := acctest.RandomWithPrefix(testResourcePrefix)
	password := acctest.RandString(20)
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	dataSourceById := "data.rediscloud_flexible_database.example-by-id"
	dataSourceByName := "data.rediscloud_flexible_database.example-by-name"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckFlexibleSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDatasourceRedisCloudFlexibleDatabase, testCloudAccountName, name, password),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceById, "name", "tf-database"),
					resource.TestCheckResourceAttr(dataSourceById, "protocol", "redis"),
					resource.TestCheckResourceAttr(dataSourceById, "region", "eu-west-1"),
					resource.TestCheckResourceAttr(dataSourceById, "memory_limit_in_gb", "1"),
					resource.TestCheckResourceAttr(dataSourceById, "support_oss_cluster_api", "true"),
					resource.TestCheckResourceAttr(dataSourceById, "resp_version", "resp2"),
					resource.TestCheckResourceAttr(dataSourceById, "data_persistence", "none"),
					resource.TestCheckResourceAttr(dataSourceById, "data_eviction", "volatile-lru"),
					resource.TestCheckResourceAttr(dataSourceById, "replication", "false"),
					resource.TestCheckResourceAttr(dataSourceById, "throughput_measurement_by", "operations-per-second"),
					resource.TestCheckResourceAttr(dataSourceById, "throughput_measurement_value", "1000"),
					resource.TestCheckResourceAttr(dataSourceById, "password", password),
					resource.TestCheckResourceAttrSet(dataSourceById, "public_endpoint"),
					resource.TestCheckResourceAttrSet(dataSourceById, "private_endpoint"),
					resource.TestCheckResourceAttr(dataSourceById, "enable_default_user", "true"),

					resource.TestCheckResourceAttr(dataSourceByName, "name", "tf-database"),
					resource.TestCheckResourceAttr(dataSourceByName, "protocol", "redis"),
					resource.TestCheckResourceAttr(dataSourceByName, "region", "eu-west-1"),
					resource.TestCheckResourceAttr(dataSourceByName, "memory_limit_in_gb", "1"),
					resource.TestCheckResourceAttr(dataSourceByName, "support_oss_cluster_api", "true"),
					resource.TestCheckResourceAttr(dataSourceByName, "resp_version", "resp2"),
					resource.TestCheckResourceAttr(dataSourceByName, "data_persistence", "none"),
					resource.TestCheckResourceAttr(dataSourceByName, "data_eviction", "volatile-lru"),
					resource.TestCheckResourceAttr(dataSourceByName, "replication", "false"),
					resource.TestCheckResourceAttr(dataSourceByName, "throughput_measurement_by", "operations-per-second"),
					resource.TestCheckResourceAttr(dataSourceByName, "throughput_measurement_value", "1000"),
					resource.TestCheckResourceAttr(dataSourceByName, "password", password),
					resource.TestCheckResourceAttrSet(dataSourceByName, "public_endpoint"),
					resource.TestCheckResourceAttrSet(dataSourceByName, "private_endpoint"),
					resource.TestCheckResourceAttr(dataSourceByName, "enable_default_user", "true"),
				),
			},
		},
	})
}

const testAccDatasourceRedisCloudFlexibleDatabase = `
data "rediscloud_payment_method" "card" {
  card_type = "Visa"
}
data "rediscloud_cloud_account" "account" {
  exclude_internal_account = true
  provider_type = "AWS" 
  name = "%s"
}
resource "rediscloud_flexible_subscription" "example" {
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
  }
}
resource "rediscloud_flexible_database" "example" {
    subscription_id              = rediscloud_flexible_subscription.example.id
    name                         = "tf-database"
    protocol                     = "redis"
    memory_limit_in_gb           = 1
    data_persistence             = "none"
    throughput_measurement_by    = "operations-per-second"
    throughput_measurement_value = 1000
	password                     = "%s"
	support_oss_cluster_api	     = true
	replication				     = false
    enable_default_user 		 = true
}

data "rediscloud_flexible_database" "example-by-id" {
  subscription_id = rediscloud_flexible_subscription.example.id
  db_id = rediscloud_flexible_database.example.db_id
}

data "rediscloud_flexible_database" "example-by-name" {
  subscription_id = rediscloud_flexible_subscription.example.id
  name = rediscloud_flexible_database.example.name
}
`
