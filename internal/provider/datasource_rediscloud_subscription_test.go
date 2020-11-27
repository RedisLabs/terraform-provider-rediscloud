package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceRedisCloudSubscription_basic(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-test")
	password := acctest.RandString(20)

	resourceName := "rediscloud_subscription.example"
	dataSourceName := "data.rediscloud_subscription.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDatasourceRedisCloudSubscriptionOneDb, testCloudAccountName, name, 1, password),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(resourceName, "name", regexp.MustCompile(name)),
				),
			},
			{
				Config: fmt.Sprintf(testAccDatasourceRedisCloudSubscriptionDataSource, name) + fmt.Sprintf(testAccDatasourceRedisCloudSubscriptionOneDb, testCloudAccountName, name, 1, password),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(dataSourceName, "name", regexp.MustCompile(name)),
					resource.TestCheckResourceAttrSet(dataSourceName, "payment_method_id"),
					resource.TestMatchResourceAttr(dataSourceName, "memory_storage", regexp.MustCompile("ram")),
					resource.TestCheckResourceAttr(dataSourceName, "persistent_storage_encryption", "false"),
					resource.TestCheckResourceAttr(dataSourceName, "number_of_databases", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "cloud_provider.0.provider", "AWS"),
					resource.TestCheckResourceAttrSet(dataSourceName, "cloud_provider.0.cloud_account_id"),
					resource.TestCheckResourceAttr(dataSourceName, "cloud_provider.0.region.0.region", "eu-west-1"),
					resource.TestCheckResourceAttr(dataSourceName, "cloud_provider.0.region.0.networks.0.networking_deployment_cidr", "10.0.0.0/24"),
					resource.TestCheckResourceAttr(dataSourceName, "status", "active"),
				),
			},
		},
	})
}

const testAccDatasourceRedisCloudSubscriptionOneDb = `

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
  persistent_storage_encryption = false

  cloud_provider {
    provider = data.rediscloud_cloud_account.account.provider_type
    cloud_account_id = data.rediscloud_cloud_account.account.id
    region {
      region = "eu-west-1"
      networking_deployment_cidr = "10.0.0.0/24"
      preferred_availability_zones = ["eu-west-1a"]
    }
  }

  database {
    name = "tf-database"
    protocol = "redis"
    memory_limit_in_gb = %d
    support_oss_cluster_api = true
    data_persistence = "none"
    replication = false
    throughput_measurement_by = "operations-per-second"
    password = "%s"
    throughput_measurement_value = 10000
  }
}
`

const testAccDatasourceRedisCloudSubscriptionDataSource = `

data "rediscloud_subscription" "example" {
  name = "%s"
}
`
