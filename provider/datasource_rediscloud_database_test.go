package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceRedisCloudDatabase_basic(t *testing.T) {
	name := acctest.RandomWithPrefix(testResourcePrefix)
	password := acctest.RandString(20)
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	dataSourceName := "data.rediscloud_database.example"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDatasourceRedisCloudDatabase, testCloudAccountName, name, password),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "name", "tf-database"),
					resource.TestCheckResourceAttr(dataSourceName, "protocol", "redis"),
					resource.TestCheckResourceAttr(dataSourceName, "region", "eu-west-1"),
					resource.TestCheckResourceAttr(dataSourceName, "memory_limit_in_gb", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "support_oss_cluster_api", "true"),
					resource.TestCheckResourceAttr(dataSourceName, "resp_version", "resp2"),
					resource.TestCheckResourceAttr(dataSourceName, "data_persistence", "none"),
					resource.TestCheckResourceAttr(dataSourceName, "data_eviction", "volatile-lru"),
					resource.TestCheckResourceAttr(dataSourceName, "replication", "false"),
					resource.TestCheckResourceAttr(dataSourceName, "throughput_measurement_by", "operations-per-second"),
					resource.TestCheckResourceAttr(dataSourceName, "throughput_measurement_value", "1000"),
					resource.TestCheckResourceAttr(dataSourceName, "password", password),
					resource.TestCheckResourceAttrSet(dataSourceName, "public_endpoint"),
					resource.TestCheckResourceAttrSet(dataSourceName, "private_endpoint"),
					resource.TestCheckResourceAttr(dataSourceName, "enable_default_user", "true"),
				),
			},
		},
	})
}

func TestAccDataSourceRedisCloudDatabase_filterAADatabases(t *testing.T) {
	name := acctest.RandomWithPrefix(testResourcePrefix)
	password := acctest.RandString(20)

	dataSourceName := "data.rediscloud_database.example"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDatasourceRedisCloudAADatabase, name+"-subscription", name+"-database", password),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "name", name+"-database"),
					resource.TestCheckResourceAttr(dataSourceName, "protocol", "redis"),
					resource.TestCheckResourceAttr(dataSourceName, "support_oss_cluster_api", "false"),
					resource.TestCheckResourceAttr(dataSourceName, "data_eviction", "volatile-lru"),
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

data "rediscloud_database" "example" {
  subscription_id = rediscloud_subscription.example.id
  name = rediscloud_subscription_database.example.name
}
`

const testAccDatasourceRedisCloudAADatabase = `
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
