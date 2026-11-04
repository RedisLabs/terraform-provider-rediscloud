package provider_test

import (
	"fmt"
	"testing"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccResourceRedisCloudPrivateServiceConnectEndpoint_CRUDI(t *testing.T) {

	baseName := testRandomWithPrefix() + "-pro-psce"

	const resourceName = "rediscloud_private_service_connect_endpoint.psce"
	gcpProjectId, gcpProjectCheck := envchecks.GCPProjectValueAndCheck()
	gcpVPCName := fmt.Sprintf("%s-network", baseName)
	gcpVPCSubnetName := fmt.Sprintf("%s-subnet", baseName)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, envchecks.RedisCloudCheck, gcpProjectCheck),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testhelpers.CheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudPrivateServiceConnectEndpointProStep1, baseName, gcpProjectId, gcpVPCName, gcpVPCSubnetName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "private_service_connect_endpoint_id"),
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
    dataset_size_in_gb           = 1
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
