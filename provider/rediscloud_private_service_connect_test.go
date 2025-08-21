package provider

import (
	"fmt"
	"testing"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceRedisCloudPrivateServiceConnect_CRUDI(t *testing.T) {

	utils.TestAccRequiresEnvVar(t, "EXECUTE_TESTS")

	baseName := acctest.RandomWithPrefix(utils.TestResourcePrefix) + "-pro-psc"

	const resourceName = "rediscloud_private_service_connect.psc"
	const datasourceName = "data.rediscloud_private_service_connect.psc"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { utils.TestAccPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudPrivateServiceConnectProStep1, baseName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "subscription_id"),
					resource.TestCheckResourceAttrSet(resourceName, "private_service_connect_service_id"),
				),
			},
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudPrivateServiceConnectProStep2, baseName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(datasourceName, "id"),
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

const testAccResourceRedisCloudPrivateServiceConnectProStep1 = `
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
`

const testAccResourceRedisCloudPrivateServiceConnectProStep2 = testAccResourceRedisCloudPrivateServiceConnectProStep1 + `

data "rediscloud_private_service_connect" "psc" {
  subscription_id = rediscloud_subscription.subscription_resource.id
}
`
