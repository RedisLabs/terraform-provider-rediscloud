package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"os"
	"testing"
)

// TestAccResourceRedisCloudActiveActiveSubscription_CMK is a semi-automated test that requires the user to pause midway through
// to give the CMK the necessary permissions.
func TestAccResourceRedisCloudActiveActiveSubscription_CMK(t *testing.T) {

	testAccRequiresEnvVar(t, "EXECUTE_TESTS")
	testAccRequiresEnvVar(t, "GCP_CMK_RESOURCE_NAME")

	name := acctest.RandomWithPrefix(testResourcePrefix)
	const resourceName = "rediscloud_active_active_subscription.example"
	gcpCmkResourceName := os.Getenv("GCP_CMK_RESOURCE_NAME")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config:             fmt.Sprintf(activeActiveCmkStep1Config, name),
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttrSet(resourceName, "customer_managed_key_redis_service_account"),
					resource.TestCheckResourceAttr(resourceName, "payment_method", "credit-card"),
					resource.TestCheckResourceAttrSet(resourceName, "payment_method_id"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider"),
					resource.TestCheckResourceAttrSet(resourceName, "creation_plan.0.dataset_size_in_gb"),
					resource.TestCheckResourceAttr(resourceName, "customer_managed_key_enabled", "true"),
				),
			},
			{
				Config:             fmt.Sprintf(activeActiveCmkStep2Config, name, gcpCmkResourceName),
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttrSet(resourceName, "customer_managed_key_redis_service_account"),
					resource.TestCheckResourceAttr(resourceName, "payment_method", "credit-card"),
					resource.TestCheckResourceAttrSet(resourceName, "payment_method_id"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider"),
					resource.TestCheckResourceAttrSet(resourceName, "creation_plan.0.dataset_size_in_gb"),
					resource.TestCheckResourceAttr(resourceName, "customer_managed_key_enabled", "true"),
				),
			},
		},
	})
}

const activeActiveCmkStep1Config = `


locals {
resource_name = "%s"
}

data "rediscloud_payment_method" "card" {
	card_type = "Visa"
	last_four_numbers = "5556"
}

resource "rediscloud_active_active_subscription" "example" {
  name = local.resource_name
  payment_method = "credit-card"
  payment_method_id = data.rediscloud_payment_method.card.id
  customer_managed_key_enabled = true
  cloud_provider = "GCP"

  creation_plan {
    memory_limit_in_gb = 1
    quantity = 1
    region {
      region = "europe-west1"
      networking_deployment_cidr = "192.168.0.0/24"
      write_operations_per_second = 1000
      read_operations_per_second = 1000
    }
    region {
      region = "europe-west2"
      networking_deployment_cidr = "10.0.1.0/24"
      write_operations_per_second = 1000
      read_operations_per_second = 1000
    }
  }
}
`

const activeActiveCmkStep2Config = `

locals {
resource_name = "%s"
customer_managed_key_resource_name = "%s"
}

data "rediscloud_payment_method" "card" {
	card_type = "Visa"
	last_four_numbers = "5556"
}

resource "rediscloud_active_active_subscription" "example" {
  name                         = local.resource_name 
  payment_method               = "credit-card"
  payment_method_id            = data.rediscloud_payment_method.card.id
  customer_managed_key_enabled = true
  cloud_provider               = "GCP"

  customer_managed_key {
    resource_name = local.customer_managed_key_resource_name
	region = "europe-west1"
  }

  customer_managed_key {
    resource_name = local.customer_managed_key_resource_name
	region = "europe-west2"
  }

  creation_plan {
    memory_limit_in_gb = 1
    quantity           = 1
    region {
      region                      = "europe-west1"
      networking_deployment_cidr  = "192.168.0.0/24"
      write_operations_per_second = 1000
      read_operations_per_second  = 1000
    }
    region {
      region                      = "europe-west2"
      networking_deployment_cidr  = "10.0.1.0/24"
      write_operations_per_second = 1000
      read_operations_per_second  = 1000
    }
  }
}

`
