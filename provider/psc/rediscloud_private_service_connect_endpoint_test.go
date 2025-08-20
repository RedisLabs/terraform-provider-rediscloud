package psc

import (
	"fmt"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/pro"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceRedisCloudPrivateServiceConnectEndpoint_CRUDI(t *testing.T) {

	utils.TestAccRequiresEnvVar(t, "EXECUTE_TESTS")

	baseName := acctest.RandomWithPrefix(utils.TestResourcePrefix) + "-pro-psce"

	const resourceName = "rediscloud_private_service_connect_endpoint.psce"
	const datasourceName = "data.rediscloud_private_service_connect_endpoints.psce"
	gcpProjectId := os.Getenv("GCP_PROJECT_ID")
	gcpVPCName := fmt.Sprintf("%s-network", baseName)
	gcpVPCSubnetName := fmt.Sprintf("%s-subnet", baseName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { utils.TestAccPreCheck(t); provider.testAccGcpProjectPreCheck(t) },
		ProviderFactories: provider.ProviderFactories(t),
		CheckDestroy:      pro.testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudPrivateServiceConnectEndpointProStep1, baseName, gcpProjectId, gcpVPCName, gcpVPCSubnetName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "private_service_connect_endpoint_id"),
				),
			},
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudPrivateServiceConnectEndpointProStep2, baseName, gcpProjectId, gcpVPCName, gcpVPCSubnetName),
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

const testAccResourceRedisCloudPrivateServiceConnectEndpointProStep1 = `
data "rediscloud_payment_method" "card" {
	card_type = "Visa"
	last_four_numbers = "5556"
}

resource "rediscloud_subscription" "subscription_resource" {
  name = "%s"
  payment_method_id = data.rediscloud_payment_method.card.id

  cloud_provider {
    provider = "GCP"
    region {
      region                     = "us-central1"
      networking_deployment_cidr = "10.0.0.0/24"
    }
  }

  creation_plan {
    dataset_size_in_gb           = 15
    quantity                     = 1
    replication                  = true
    throughput_measurement_by    = "operations-per-second"
    throughput_measurement_value = 20000
  }
}

resource "rediscloud_private_service_connect" "psc" {
  subscription_id = rediscloud_subscription.subscription_resource.id
}

resource "rediscloud_private_service_connect_endpoint" "psce" {
  subscription_id = rediscloud_subscription.subscription_resource.id
  private_service_connect_service_id = rediscloud_private_service_connect.psc.private_service_connect_service_id
  gcp_project_id = "%s"
  gcp_vpc_name = "%s"
  gcp_vpc_subnet_name = "%s"
  endpoint_connection_name = "redis-${rediscloud_subscription.subscription_resource.id}"
}
`

const testAccResourceRedisCloudPrivateServiceConnectEndpointProStep2 = testAccResourceRedisCloudPrivateServiceConnectEndpointProStep1 + `

data "rediscloud_private_service_connect_endpoints" "psce" {
  subscription_id = rediscloud_subscription.subscription_resource.id
  private_service_connect_service_id = rediscloud_private_service_connect.psc.private_service_connect_service_id
}
`
