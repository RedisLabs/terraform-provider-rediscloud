package provider

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceRedisCloudActiveActivePrivateServiceConnectEndpoint_CRUDI(t *testing.T) {

	utils.TestAccRequiresEnvVar(t, "EXECUTE_TESTS")

	baseName := acctest.RandomWithPrefix(utils.TestResourcePrefix) + "-pro-psce"

	const resourceName = "rediscloud_active_active_private_service_connect_endpoint.psce"
	const datasourceName = "data.rediscloud_active_active_private_service_connect_endpoints.psce"
	gcpProjectId := os.Getenv("GCP_PROJECT_ID")
	gcpVPCName := fmt.Sprintf("%s-network", baseName)
	gcpVPCSubnetName := fmt.Sprintf("%s-subnet", baseName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { utils.TestAccPreCheck(t); utils.TestAccGcpProjectPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudActiveActivePrivateServiceConnectEndpointProStep1, baseName, gcpProjectId, gcpVPCName, gcpVPCSubnetName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "private_service_connect_endpoint_id"),
				),
			},
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudActiveActivePrivateServiceConnectEndpointProStep2, baseName, gcpProjectId, gcpVPCName, gcpVPCSubnetName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(datasourceName, "id"),
					resource.TestCheckResourceAttr(datasourceName, "endpoints.#", "1"),
					resource.TestCheckResourceAttr(datasourceName, "endpoints.0.gcp_project_id", gcpProjectId),
					resource.TestCheckResourceAttr(datasourceName, "endpoints.0.gcp_vpc_name", gcpVPCName),
					resource.TestCheckResourceAttr(datasourceName, "endpoints.0.gcp_vpc_subnet_name", gcpVPCSubnetName),
					resource.TestCheckResourceAttrWith(datasourceName, "endpoints.0.endpoint_connection_name", func(value string) error {
						if !strings.HasPrefix(value, "redis-") {
							return fmt.Errorf("expected %s to have prefix 'redis-'", value)
						}
						return nil
					}),
					resource.TestCheckResourceAttr(datasourceName, "endpoints.0.service_attachments.#", "1"),
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

const testAccResourceRedisCloudActiveActivePrivateServiceConnectEndpointProStep1 = `
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

resource "rediscloud_active_active_private_service_connect_endpoint" "psce" {
  subscription_id = rediscloud_active_active_subscription.subscription_resource.id
  region_id = one([for r in rediscloud_active_active_subscription_regions.regions_resource.region : r.region_id if r.region == "us-central1"])
  private_service_connect_service_id = rediscloud_active_active_private_service_connect.psc.private_service_connect_service_id
  gcp_project_id = "%s"
  gcp_vpc_name = "%s"
  gcp_vpc_subnet_name = "%s"
  endpoint_connection_name = "redis-${rediscloud_active_active_subscription.subscription_resource.id}"
}
`

const testAccResourceRedisCloudActiveActivePrivateServiceConnectEndpointProStep2 = testAccResourceRedisCloudActiveActivePrivateServiceConnectEndpointProStep1 + `

data "rediscloud_active_active_private_service_connect_endpoints" "psce" {
  subscription_id = rediscloud_active_active_subscription.subscription_resource.id
  region_id = one([for r in rediscloud_active_active_subscription_regions.regions_resource.region : r.region_id if r.region == "us-central1"])
  private_service_connect_service_id = rediscloud_active_active_private_service_connect.psc.private_service_connect_service_id
}
`
