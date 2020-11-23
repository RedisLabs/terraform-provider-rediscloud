package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"testing"
)

func TestAccDataSourceRedisCloudDatabase_basic(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-test")
	password := acctest.RandString(20)

	dataSourceName := "data.rediscloud_database.example"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDatasourceRedisCloudDatabase, name, password),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "name", "tf-database"),
					resource.TestCheckResourceAttr(dataSourceName, "protocol", "redis"),
					resource.TestCheckResourceAttr(dataSourceName, "region", "eu-west-1"),
					resource.TestCheckResourceAttr(dataSourceName, "memory_limit_in_gb", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "support_oss_cluster_api", "true"),
					resource.TestCheckResourceAttr(dataSourceName, "data_persistence", "none"),
					resource.TestCheckResourceAttr(dataSourceName, "replication", "none"),
					resource.TestCheckResourceAttr(dataSourceName, "throughput_measurement_by", "operations-per-second"),
					resource.TestCheckResourceAttr(dataSourceName, "throughput_measurement_value", "10000"),
					resource.TestCheckResourceAttr(dataSourceName, "password", password),
					resource.TestCheckResourceAttrSet(dataSourceName, "public_endpoint"),
					resource.TestCheckResourceAttrSet(dataSourceName, "private_endpoint"),
				),
			},
		},
	})
}

const testAccDatasourceRedisCloudDatabase = `
data "rediscloud_payment_method" "card" {
  card_type = "Visa"
}

data "rediscloud_cloud_account" "account" {
  exclude_internal_account = true
  provider_type = "AWS" 
}

resource "rediscloud_subscription" "example" {

  name = "%s"
  payment_method_id = data.rediscloud_payment_method.card.id
  memory_storage = "ram"
  persistent_storage_encryption = false

  cloud_provider {
    provider = data.rediscloud_cloud_account.account.provider_type
    cloud_account_id = data.rediscloud_cloud_account.account.id
    region {
      region = "eu-west-1"
      networking_deployment_cidr = "10.0.0.0/24"
    }
  }

  database {
    name = "tf-database"
    protocol = "redis"
    memory_limit_in_gb = 1
    support_oss_cluster_api = true
    data_persistence = "none"
    replication = false
    throughput_measurement_by = "operations-per-second"
    password = "%s"
    throughput_measurement_value = 10000
  }
}

data "rediscloud_database" "example" {
  subscription_id = rediscloud_subscription.example.id
  name = "tf-database"
}
`
