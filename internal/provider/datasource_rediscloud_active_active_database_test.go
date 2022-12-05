package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceRedisCloudActiveActiveDatabase_basic(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-test")
	password := acctest.RandString(20)
	testCloudAccountName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME")

	dataSourceName := "data.rediscloud_active_active_database.example"

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
					resource.TestCheckResourceAttr(dataSourceName, "global_data_persistence", "none"),
					resource.TestCheckResourceAttr(dataSourceName, "data_eviction", "volatile-lru"),
					resource.TestCheckResourceAttr(dataSourceName, "replication", "false"),
					resource.TestCheckResourceAttr(dataSourceName, "global_password", password),
					resource.TestCheckResourceAttrSet(dataSourceName, "public_endpoint"),
					resource.TestCheckResourceAttrSet(dataSourceName, "private_endpoint"),
				),
			},
		},
	})
}

const testAccDatasourceRedisCloudActiveActiveDatabase = `
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
    replication=false
    support_oss_cluster_api=true
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
}

resource "rediscloud_subscription_active_active_database" "example" {
    subscription_id              = rediscloud_subscription.example.id
    name                         = "tf-database"
    protocol                     = "redis"
    memory_limit_in_gb           = 1
    global_data_persistence      = "none"
	global_password              = "%s"
	support_oss_cluster_api	     = true
}

data "rediscloud_database" "example" {
  subscription_id = rediscloud_subscription.example.id
  name = rediscloud_subscription_database.example.name
}
`
