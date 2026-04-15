package pro_test

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

// TestAccRedisCloudProSubscription_CMK is a fully automated CMK test that uses the GCP provider
// to create KMS keys and grant IAM permissions automatically.
func TestAccRedisCloudProSubscription_CMK(t *testing.T) {

	testhelpers.RequireEnvVars(t, "EXECUTE_TESTS", "GCP_PROJECT_ID", "GOOGLE_CREDENTIALS")

	name := utils.RandomWithPrefix()
	const resourceName = "rediscloud_subscription.example"
	gcpProjectId := os.Getenv("GCP_PROJECT_ID")

	placeholders := map[string]string{
		"__NAME__":           name,
		"__GCP_PROJECT_ID__": gcpProjectId,
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testhelpers.BasicPreCheck(t) },
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		ExternalProviders: map[string]resource.ExternalProvider{
			"google": {
				Source:            "hashicorp/google",
				VersionConstraint: "~> 6.5",
			},
		},
		CheckDestroy: checkProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				// Step 1: Create subscription with CMK enabled (enters encryption_key_pending state)
				// Also creates GCP KMS key and grants IAM permissions to the Redis service account
				Config:             utils.RenderTestConfig(t, "testdata/cmk_step1.tf", placeholders),
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttrSet(resourceName, "customer_managed_key_redis_service_account"),
					resource.TestCheckResourceAttr(resourceName, "payment_method", "credit-card"),
					resource.TestCheckResourceAttrSet(resourceName, "payment_method_id"),
					resource.TestCheckResourceAttr(resourceName, "memory_storage", "ram"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider.0.provider"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider.0.region.#"),
					resource.TestCheckResourceAttr(resourceName, "customer_managed_key_enabled", "true"),
				),
			},
			{
				// Step 2: Add CMK blocks to activate encryption
				// This triggers the UpdateCmk code path which should also clean up creation plan databases
				Config:             utils.RenderTestConfig(t, "testdata/cmk_step2.tf", placeholders),
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttrSet(resourceName, "customer_managed_key_redis_service_account"),
					resource.TestCheckResourceAttr(resourceName, "payment_method", "credit-card"),
					resource.TestCheckResourceAttrSet(resourceName, "payment_method_id"),
					resource.TestCheckResourceAttr(resourceName, "memory_storage", "ram"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider.0.provider"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider.0.region.#"),
					resource.TestCheckResourceAttr(resourceName, "customer_managed_key_enabled", "true"),
					checkNoCreationPlanDatabases(resourceName),
				),
			},
		},
	})
}

// checkNoCreationPlanDatabases verifies that no databases remain in the subscription after
// CMK activation. This confirms that the creation plan databases were properly cleaned up.
func checkNoCreationPlanDatabases(resourceName string) resource.TestCheckFunc {
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
