package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"testing"
)

func TestAccResourceRedisCloudProSubscription_CMEK(t *testing.T) {

	testAccRequiresEnvVar(t, "EXECUTE_TESTS")

	name := acctest.RandomWithPrefix(testResourcePrefix)
	const resourceName = "rediscloud_subscription.example"
	//gcpProjectId := os.Getenv("GCP_PROJECT_ID")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccResourceRedisCloudProSubscriptionCmekEnabled_create, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.region.0.preferred_availability_zones.#", "0"),
				),
			},
			//{
			//	Config: fmt.Sprintf(testAccResourceRedisCloudProSubscriptionCmekEnabled_update, gcpProjectId, name),
			//	Check: resource.ComposeAggregateTestCheckFunc(
			//		resource.TestCheckResourceAttr(resourceName, "name", name),
			//		resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.region.0.preferred_availability_zones.#", "1"),
			//	),
			//},
		},
	})
}

const testAccResourceRedisCloudProSubscriptionCmekEnabled_create = `
data "rediscloud_payment_method" "card" {
  card_type = "Visa"
}

resource "rediscloud_subscription" "example" {

  name              				= "%s"
  payment_method    				= "credit-card"
  payment_method_id 				= data.rediscloud_payment_method.card.id
  memory_storage    				= "ram"
  customer_managed_key_enabled      = true

  allowlist {
   cidrs = ["192.168.0.0/16"]
   security_group_ids = []
  }

  creation_plan {
    dataset_size_in_gb       = 1
    quantity                 = 1
    replication              = false
    support_oss_cluster_api  = false
    query_performance_factor = "4x"

    throughput_measurement_by    = "operations-per-second"
    throughput_measurement_value = 10000
    modules = ["RedisJSON", "RedisBloom", "RediSearch"]
  }

  cloud_provider {
    provider         = "GCP"
    region {
      region                     = "europe-west2"
      networking_deployment_cidr = "10.0.1.0/24"
    }
  }
}
`

const testAccResourceRedisCloudProSubscriptionCmekEnabled_update = `
data "rediscloud_payment_method" "card" {
  card_type = "Visa"
}

resource "rediscloud_subscription" "example" {

  cloud_provider {
    provider = "GCP"
    cloud_account_id = %s
    region {
      region                     = "europe-west2"
      networking_deployment_cidr = "10.0.1.0/24"
    }

  name = "%s"
  payment_method = "credit-card"
  payment_method_id = data.rediscloud_payment_method.card.id
  memory_storage = "ram"
  customer_managed_key_enabled = true

  allowlist {
    cidrs = ["192.168.0.0/16"]
    security_group_ids = []
  }

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
    dataset_size_in_gb = 1
    quantity = 1
    replication = false
    support_oss_cluster_api = false
	query_performance_factor = "4x"

    throughput_measurement_by = "operations-per-second"
    throughput_measurement_value = 10000
    modules = ["RedisJSON", "RedisBloom", "RediSearch"]
  }

  cmek_id = "???"


}
`
