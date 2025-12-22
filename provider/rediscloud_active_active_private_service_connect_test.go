package provider

import (
	"fmt"
	"testing"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceRedisCloudActiveActivePrivateServiceConnect_CRUDI(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	baseName := acctest.RandomWithPrefix(testResourcePrefix) + "-pro-psc"

	const resourceName = "rediscloud_active_active_private_service_connect.psc"
	const datasourceName = "data.rediscloud_active_active_private_service_connect.psc"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudActiveActivePrivateServiceConnectProStep1, baseName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "region_id"),
					resource.TestCheckResourceAttrSet(resourceName, "subscription_id"),
					resource.TestCheckResourceAttrSet(resourceName, "private_service_connect_service_id"),
				),
			},
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudActiveActivePrivateServiceConnectProStep2, baseName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(datasourceName, "id"),
					resource.TestCheckResourceAttrSet(datasourceName, "region_id"),
					resource.TestCheckResourceAttrSet(datasourceName, "subscription_id"),
					resource.TestCheckResourceAttrSet(datasourceName, "private_service_connect_service_id"),
					resource.TestCheckResourceAttrSet(datasourceName, "connection_host_name"),
					resource.TestCheckResourceAttrSet(datasourceName, "service_attachment_name"),
					resource.TestCheckResourceAttr(datasourceName, "status", "active"),
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

const testAccResourceRedisCloudActiveActivePrivateServiceConnectProStep1 = `
data "rediscloud_payment_method" "card" {
	card_type = "Visa"
	last_four_numbers = "5556"
}

resource "rediscloud_active_active_subscription" "subscription_resource" {
  name = "%s"
  payment_method_id = data.rediscloud_payment_method.card.id
  cloud_provider = "GCP"

  creation_plan {
    memory_limit_in_gb = 1
    quantity = 1
    region {
      region = "us-central1"
      networking_deployment_cidr = "192.168.0.0/24"
      write_operations_per_second = 1000
      read_operations_per_second = 1000
    }
    region {
      region = "europe-west1"
      networking_deployment_cidr = "10.0.1.0/24"
      write_operations_per_second = 1000
      read_operations_per_second = 1000
    }
  }
}

resource "rediscloud_active_active_subscription_database" "database_resource" {
  subscription_id         = rediscloud_active_active_subscription.subscription_resource.id
  name                    = "db"
  memory_limit_in_gb      = 1
  global_data_persistence = "aof-every-1-second"
  global_password         = "some-password"
}

resource "rediscloud_active_active_subscription_regions" "regions_resource" {
  subscription_id = rediscloud_active_active_subscription.subscription_resource.id

  region {
    region = "us-central1"
    networking_deployment_cidr = "192.168.0.0/24"
    database {
      database_id                       = rediscloud_active_active_subscription_database.database_resource.db_id
      database_name                     = rediscloud_active_active_subscription_database.database_resource.name
      local_write_operations_per_second = 1000
      local_read_operations_per_second  = 1000
    }
  }
  
  region {
    region = "europe-west1"
    networking_deployment_cidr = "10.0.1.0/24"
    database {
      database_id                       = rediscloud_active_active_subscription_database.database_resource.db_id
      database_name                     = rediscloud_active_active_subscription_database.database_resource.name
      local_write_operations_per_second = 1000
      local_read_operations_per_second  = 1000
    }
  }  
}

resource "rediscloud_active_active_private_service_connect" "psc" {
  subscription_id = rediscloud_active_active_subscription.subscription_resource.id
  region_id = one([for r in rediscloud_active_active_subscription_regions.regions_resource.region : r.region_id if r.region == "us-central1"])
}
`

const testAccResourceRedisCloudActiveActivePrivateServiceConnectProStep2 = testAccResourceRedisCloudActiveActivePrivateServiceConnectProStep1 + `

data "rediscloud_active_active_private_service_connect" "psc" {
  subscription_id = rediscloud_active_active_subscription.subscription_resource.id
  region_id = one([for r in rediscloud_active_active_subscription_regions.regions_resource.region : r.region_id if r.region == "us-central1"])
}
`
