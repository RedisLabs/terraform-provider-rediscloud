package provider

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

// TestAccResourceRedisCloudActiveActiveSubscription_CMK is a semi-automated test that requires the user to pause midway through
// to give the CMK the necessary permissions.
func TestAccResourceRedisCloudActiveActiveSubscription_CMK(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")
	utils.AccRequiresEnvVar(t, "GCP_CMK_RESOURCE_NAME")

	name := acctest.RandomWithPrefix(testResourcePrefix)
	const resourceName = "rediscloud_active_active_subscription.example"
	gcpCmkResourceName := os.Getenv("GCP_CMK_RESOURCE_NAME")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             testAccCheckActiveActiveSubscriptionDestroy,
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
				PreConfig: func() {
					fmt.Println("\n" + strings.Repeat("=", 60))
					fmt.Println("MANUAL STEP REQUIRED")
					fmt.Println(strings.Repeat("=", 60))
					fmt.Println("Grant these IAM roles to the Redis service account on your GCP KMS key:")
					fmt.Println("  - roles/cloudkms.cryptoKeyEncrypterDecrypter")
					fmt.Println("  - roles/cloudkms.viewer")
					fmt.Println(strings.Repeat("=", 60))
					fmt.Print("Press ENTER when ready to continue...")
					bufio.NewReader(os.Stdin).ReadBytes('\n')
				},
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

// TestAccResourceRedisCloudActiveActiveSubscription_CMK_Automated is a fully automated CMK test
// that uses the GCP provider to create KMS keys and grant IAM permissions automatically.
func TestAccResourceRedisCloudActiveActiveSubscription_CMK_Automated(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	name := acctest.RandomWithPrefix(testResourcePrefix)
	const resourceName = "rediscloud_active_active_subscription.example"
	gcpProjectId := os.Getenv("GCP_PROJECT_ID")

	placeholders := map[string]string{
		"__NAME__":           name,
		"__GCP_PROJECT_ID__": gcpProjectId,
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t); testAccGcpProjectPreCheck(t); testAccGcpCredentialsPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		ExternalProviders: map[string]resource.ExternalProvider{
			"google": {
				Source:            "hashicorp/google",
				VersionConstraint: "~> 6.5",
			},
			"time": {
				Source:            "hashicorp/time",
				VersionConstraint: "~> 0.9",
			},
		},
		CheckDestroy: testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config:             utils.RenderTestConfig(t, "./activeactive/testdata/cmk_step1.tf", placeholders),
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttrSet(resourceName, "customer_managed_key_redis_service_account"),
					resource.TestCheckResourceAttr(resourceName, "payment_method", "credit-card"),
					resource.TestCheckResourceAttrSet(resourceName, "payment_method_id"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider"),
					resource.TestCheckResourceAttr(resourceName, "customer_managed_key_enabled", "true"),
				),
			},
			{
				Config:             utils.RenderTestConfig(t, "./activeactive/testdata/cmk_step2.tf", placeholders),
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					func(s *terraform.State) error {
						time.Sleep(10 * time.Second)
						return nil
					},
				),
			},
			{
				Config:             utils.RenderTestConfig(t, "./activeactive/testdata/cmk_step3.tf", placeholders),
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttrSet(resourceName, "customer_managed_key_redis_service_account"),
					resource.TestCheckResourceAttr(resourceName, "payment_method", "credit-card"),
					resource.TestCheckResourceAttrSet(resourceName, "payment_method_id"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider"),
					resource.TestCheckResourceAttr(resourceName, "customer_managed_key_enabled", "true"),
				),
			},
		},
	})
}
