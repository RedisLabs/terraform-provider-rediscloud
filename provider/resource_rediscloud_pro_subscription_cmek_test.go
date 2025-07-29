package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"os"
	"testing"
)

func TestAccResourceRedisCloudProSubscription_CMEK(t *testing.T) {

	testAccRequiresEnvVar(t, "EXECUTE_TESTS")
	testAccRequiresEnvVar(t, "GCP_CMK_RESOURCE_NAME")

	name := acctest.RandomWithPrefix(testResourcePrefix)
	const resourceName = "rediscloud_subscription.example"
	gcpCmkResourceName := os.Getenv("GCP_CMK_RESOURCE_NAME")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config:             fmt.Sprintf(step1Config, name),
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					//resource.TestCheckResourceAttrSet(resourceName, "customer_managed_key_redis_service_account"),
				),
			},
			{
				Config:             fmt.Sprintf(step2Config, name, gcpCmkResourceName),
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
				),
			},
		},
	})
}

const step1Config = `
data "rediscloud_payment_method" "card" {
  card_type = "Visa"
}

resource "rediscloud_subscription" "example" {
  name = "%s"
  payment_method = "credit-card"
  payment_method_id = data.rediscloud_payment_method.card.id
  memory_storage = "ram"
  customer_managed_key_enabled = true

  cloud_provider {
    provider = "GCP"
    region {
      region                     = "europe-west2"
      networking_deployment_cidr = "10.0.1.0/24"
    }
  }

  creation_plan {
    dataset_size_in_gb = 1
    quantity = 1
    replication = false
    support_oss_cluster_api = false
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 10000
  }
}

output "rediscloud_service_account" {
  value = rediscloud_subscription.example.customer_managed_key_redis_service_account
}
`

const step2Config = `
data "rediscloud_payment_method" "card" {
  card_type = "Visa"
}

resource "rediscloud_subscription" "example" {
  name = "%s"
  payment_method = "credit-card"
  payment_method_id = data.rediscloud_payment_method.card.id
  memory_storage = "ram"
  customer_managed_key_enabled = true

  cloud_provider {
    provider = "GCP"
    region {
      region                     = "europe-west2"
      networking_deployment_cidr = "10.0.1.0/24"
    }
  }

  creation_plan {
    dataset_size_in_gb = 1
    quantity = 1
    replication = false
    support_oss_cluster_api = false
    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 10000
  }

  customer_managed_key {
    resource_name = "%s"
  }
}
`
