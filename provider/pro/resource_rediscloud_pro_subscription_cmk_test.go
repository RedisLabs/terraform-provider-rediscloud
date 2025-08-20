package pro

import (
	"fmt"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"os"
	"testing"
)

// TestAccResourceRedisCloudProSubscription_CMK is a semi-automated test that requires the user to pause midway through
// to give the CMK the necessary permissions.
// TODO: integrate the GCP provider and set up these permissions automatically
func TestAccResourceRedisCloudProSubscription_CMK(t *testing.T) {

	utils.TestAccRequiresEnvVar(t, "EXECUTE_TESTS")
	utils.TestAccRequiresEnvVar(t, "GCP_CMK_RESOURCE_NAME")

	name := acctest.RandomWithPrefix(utils.TestResourcePrefix)
	const resourceName = "rediscloud_subscription.example"
	gcpCmkResourceName := os.Getenv("GCP_CMK_RESOURCE_NAME")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { utils.TestAccPreCheck(t); utils.TestAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: provider.ProviderFactories(t),
		CheckDestroy:      testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config:             fmt.Sprintf(proCmkStep1Config, name),
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttrSet(resourceName, "customer_managed_key_redis_service_account"),
					resource.TestCheckResourceAttr(resourceName, "payment_method", "credit-card"),
					resource.TestCheckResourceAttrSet(resourceName, "payment_method_id"),
					resource.TestCheckResourceAttr(resourceName, "memory_storage", "ram"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider.0.provider"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider.0.region.#"), // number of regions
					resource.TestCheckResourceAttrSet(resourceName, "creation_plan.0.dataset_size_in_gb"),
					resource.TestCheckResourceAttr(resourceName, "customer_managed_key_enabled", "true"),
				),
			},
			{
				Config:             fmt.Sprintf(proCmkStep2Config, name, gcpCmkResourceName),
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttrSet(resourceName, "customer_managed_key_redis_service_account"),
					resource.TestCheckResourceAttr(resourceName, "payment_method", "credit-card"),
					resource.TestCheckResourceAttrSet(resourceName, "payment_method_id"),
					resource.TestCheckResourceAttr(resourceName, "memory_storage", "ram"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider.0.provider"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider.0.region.#"),
					resource.TestCheckResourceAttrSet(resourceName, "creation_plan.0.dataset_size_in_gb"),
					resource.TestCheckResourceAttr(resourceName, "customer_managed_key_enabled", "true"),
				),
			},
		},
	})
}

const proCmkStep1Config = `
data "rediscloud_payment_method" "card" {
	card_type = "Visa"
	last_four_numbers = "5556"
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
`

const proCmkStep2Config = `
data "rediscloud_payment_method" "card" {
	card_type = "Visa"
	last_four_numbers = "5556"
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
