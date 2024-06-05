package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceRedisCloudProSubscription_basic(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-test")

	resourceName := "rediscloud_subscription.example"
	dataSourceName := "data.rediscloud_subscription.example"
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDatasourceRedisCloudProSubscription, testCloudAccountName, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(resourceName, "name", regexp.MustCompile(name)),
				),
			},
			{
				Config: fmt.Sprintf(testAccDatasourceRedisCloudProSubscriptionDataSource, name) + fmt.Sprintf(testAccDatasourceRedisCloudProSubscription, testCloudAccountName, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(dataSourceName, "name", regexp.MustCompile(name)),
					resource.TestCheckResourceAttr(dataSourceName, "payment_method", "credit-card"),
					resource.TestCheckResourceAttrSet(dataSourceName, "payment_method_id"),
					resource.TestMatchResourceAttr(dataSourceName, "memory_storage", regexp.MustCompile("ram")),
					resource.TestCheckResourceAttr(dataSourceName, "number_of_databases", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "cloud_provider.0.provider", "AWS"),
					resource.TestCheckResourceAttrSet(dataSourceName, "cloud_provider.0.cloud_account_id"),
					resource.TestCheckResourceAttr(dataSourceName, "cloud_provider.0.region.0.region", "eu-west-1"),
					resource.TestCheckResourceAttr(dataSourceName, "cloud_provider.0.region.0.networks.0.networking_deployment_cidr", "10.0.0.0/24"),
					resource.TestCheckResourceAttr(dataSourceName, "status", "active"),

					resource.TestCheckResourceAttr(dataSourceName, "pricing.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "pricing.0.type", "Shards"),
					resource.TestCheckResourceAttr(dataSourceName, "pricing.0.type_details", "micro"),
					resource.TestCheckResourceAttr(dataSourceName, "pricing.0.quantity", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "pricing.0.quantity_measurement", "shards"),
					resource.TestCheckResourceAttrSet(dataSourceName, "pricing.0.price_per_unit"),
					resource.TestCheckResourceAttr(dataSourceName, "pricing.0.price_currency", "USD"),
					resource.TestCheckResourceAttr(dataSourceName, "pricing.0.price_period", "hour"),
				),
			},
		},
	})
}

func TestAccDataSourceRedisCloudProSubscription_ignoresAA(t *testing.T) {
	name := acctest.RandomWithPrefix(testResourcePrefix)
	password := acctest.RandString(20)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(testAccDatasourceRedisCloudAADatabaseWithProDataSource, name+"-subscription", name+"-database", password),
				ExpectError: regexp.MustCompile("Your query returned no results. Please change your search criteria and try again."),
			},
		},
	})
}

const testAccDatasourceRedisCloudProSubscription = `
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
  payment_method = "credit-card"
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
    memory_limit_in_gb           = 1
    quantity                     = 1
    replication                  = false
    support_oss_cluster_api      = false
    throughput_measurement_by    = "operations-per-second"
    throughput_measurement_value = 1000
    modules = []
  }
}
resource "rediscloud_subscription_database" "example" {
    subscription_id              = rediscloud_subscription.example.id
	name                         = "tf-database"
    protocol                     = "redis"
    memory_limit_in_gb           = 1
    data_persistence             = "none"
    throughput_measurement_by    = "operations-per-second"
    throughput_measurement_value = 1000
}
`

const testAccDatasourceRedisCloudProSubscriptionDataSource = `
data "rediscloud_subscription" "example" {
  name = "%s"
}
`

const testAccDatasourceRedisCloudAADatabaseWithProDataSource = `
data "rediscloud_payment_method" "card" {
	card_type = "Visa"
}
resource "rediscloud_active_active_subscription" "example" {
	name = "%s"
	payment_method_id = data.rediscloud_payment_method.card.id
	cloud_provider = "AWS"
	creation_plan {
		memory_limit_in_gb = 1
		quantity = 1
		region {
			region = "us-east-1"
			networking_deployment_cidr = "192.168.0.0/24"
			write_operations_per_second = 1000
			read_operations_per_second = 1000
		}
		region {
			region = "us-east-2"
			networking_deployment_cidr = "10.0.1.0/24"
			write_operations_per_second = 1000
			read_operations_per_second = 1000
		}
	}
}
resource "rediscloud_active_active_subscription_database" "example" {
    subscription_id = rediscloud_active_active_subscription.example.id
    name = "%s"
    memory_limit_in_gb = 3
    support_oss_cluster_api = false
    external_endpoint_for_oss_cluster_api = false
	enable_tls = false

    global_data_persistence = "none"
    global_password = "%s"
    global_source_ips = ["192.168.0.0/16", "192.170.0.0/16"]
    global_alert {
		name = "dataset-size"
		value = 40
	}
	override_region {
		name = "us-east-1"
		override_global_data_persistence = "aof-every-write"
		override_global_source_ips = ["192.175.0.0/16"]
		override_global_password = "region-specific-password"
		override_global_alert {
			name = "dataset-size"
			value = 42
		}
	}
	override_region {
		name = "us-east-2"
	}
}
data "rediscloud_database" "example" {
  subscription_id = rediscloud_active_active_subscription.example.id
  name = rediscloud_active_active_subscription_database.example.name
}
`
