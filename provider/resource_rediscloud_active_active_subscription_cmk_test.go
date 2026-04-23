package provider

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

<<<<<<< Updated upstream
// TestAccResourceRedisCloudActiveActiveSubscription_CMK is a fully automated CMK test
=======
// TestAccResourceRedisCloudActiveActiveSubscription_CMK is a semi-automated test that requires the user to pause midway through
// to give the CMK the necessary permissions.
func TestAccResourceRedisCloudActiveActiveSubscription_CMK(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")
	utils.AccRequiresEnvVar(t, "GCP_CMK_RESOURCE_NAME")

	name := testRandomWithPrefix()
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
					_, _ = bufio.NewReader(os.Stdin).ReadBytes('\n')
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

// TestAccResourceRedisCloudActiveActiveSubscription_CMK_AWS is a semi-automated test for AWS CMK support.
// It requires the user to grant KMS key policy permissions between steps.
func TestAccResourceRedisCloudActiveActiveSubscription_CMK_AWS(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")
	utils.AccRequiresEnvVar(t, "AWS_CMK_KEY_ARN")

	name := testRandomWithPrefix()
	const resourceName = "rediscloud_active_active_subscription.example"
	awsCmkKeyArn := os.Getenv("AWS_CMK_KEY_ARN")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				Config:             fmt.Sprintf(activeActiveCmkAwsStep1Config, name),
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttrSet(resourceName, "customer_managed_key_aws_role_arn"),
					resource.TestCheckResourceAttr(resourceName, "payment_method", "credit-card"),
					resource.TestCheckResourceAttrSet(resourceName, "payment_method_id"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider"),
					resource.TestCheckResourceAttrSet(resourceName, "creation_plan.0.dataset_size_in_gb"),
					resource.TestCheckResourceAttr(resourceName, "customer_managed_key_enabled", "true"),
				),
			},
			{
				Config:             fmt.Sprintf(activeActiveCmkAwsStep2Config, name, awsCmkKeyArn),
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttrSet(resourceName, "customer_managed_key_aws_role_arn"),
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

const activeActiveCmkAwsStep1Config = `

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
  cloud_provider = "AWS"

  creation_plan {
    memory_limit_in_gb = 1
    quantity = 1
    region {
      region = "us-east-1"
      networking_deployment_cidr = "192.168.0.0/24"
      write_operations_per_second = 1000
      read_operations_per_second = 1000
    }
    region {
      region = "us-east-2"
      networking_deployment_cidr = "10.0.1.0/24"
      write_operations_per_second = 1000
      read_operations_per_second = 1000
    }
  }
}
`

const activeActiveCmkAwsStep2Config = `

locals {
resource_name = "%s"
customer_managed_key_arn = "%s"
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
  cloud_provider               = "AWS"

  customer_managed_key {
    resource_name = local.customer_managed_key_arn
	region = "us-east-1"
  }

  customer_managed_key {
    resource_name = local.customer_managed_key_arn
	region = "us-east-2"
  }

  creation_plan {
    memory_limit_in_gb = 1
    quantity           = 1
    region {
      region                      = "us-east-1"
      networking_deployment_cidr  = "192.168.0.0/24"
      write_operations_per_second = 1000
      read_operations_per_second  = 1000
    }
    region {
      region                      = "us-east-2"
      networking_deployment_cidr  = "10.0.1.0/24"
      write_operations_per_second = 1000
      read_operations_per_second  = 1000
    }
  }
}

`

// TestAccResourceRedisCloudActiveActiveSubscription_CMK_Automated is a fully automated CMK test
>>>>>>> Stashed changes
// that uses the GCP provider to create KMS keys and grant IAM permissions automatically.
func TestAccResourceRedisCloudActiveActiveSubscription_CMK(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	name := testRandomWithPrefix()
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
		},
		CheckDestroy: testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				// Step 1: Create subscription with CMK enabled (enters encryption_key_pending state)
				// Also creates GCP KMS key and grants IAM permissions to the Redis service account
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
				// Step 2: Add CMK blocks to activate encryption
				// This triggers the UpdateCmk code path which should also clean up creation plan databases
				Config:             utils.RenderTestConfig(t, "./activeactive/testdata/cmk_step2.tf", placeholders),
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttrSet(resourceName, "customer_managed_key_redis_service_account"),
					resource.TestCheckResourceAttr(resourceName, "payment_method", "credit-card"),
					resource.TestCheckResourceAttrSet(resourceName, "payment_method_id"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider"),
					resource.TestCheckResourceAttr(resourceName, "customer_managed_key_enabled", "true"),
					testAccCheckNoCreationPlanDatabases(resourceName),
				),
			},
		},
	})
}

// testAccCheckNoCreationPlanDatabases verifies that no databases remain in the subscription after
// CMK activation. This confirms that the creation plan databases were properly cleaned up.
func testAccCheckNoCreationPlanDatabases(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		r, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		subId, err := strconv.Atoi(r.Primary.ID)
		if err != nil {
			return fmt.Errorf("could not parse subscription ID: %s", r.Primary.ID)
		}

		return utils.CheckNoDatabasesForSubscription(context.TODO(), subId)
	}
}
