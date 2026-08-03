package pro_test

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

// TestAccRedisCloudProSubscription_CMK is a fully automated CMK test that uses the GCP provider
// to create KMS keys and grant IAM permissions automatically.
func TestAccRedisCloudProSubscription_CMK(t *testing.T) {
	t.Skip("developer-only: GCP CMK is not supported in CI. The CI service account behind GOOGLE_CREDENTIALS lacks `cloudkms.keyRings.create` on the test project, so the in-fixture key ring cannot be created. Run locally with credentials holding roles/cloudkms.admin.")

	name := utils.RandomWithPrefix()
	const resourceName = "rediscloud_subscription.example"
	gcpProjectId := os.Getenv("GCP_PROJECT_ID")

	configVars := config.Variables{
		"name":           config.StringVariable(name),
		"gcp_project_id": config.StringVariable(gcpProjectId),
		"maintenance_windows": config.ListVariable(config.ObjectVariable(
			map[string]config.Variable{
				"mode": config.StringVariable("manual"),
				"window": config.ListVariable(config.ObjectVariable(
					map[string]config.Variable{
						"start_hour":        config.IntegerVariable(22),
						"duration_in_hours": config.IntegerVariable(8),
						"days":              config.ListVariable(config.StringVariable("Monday"), config.StringVariable("Thursday")),
					})),
			},
		)),
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:     envchecks.ComposePreChecks(t, envchecks.RedisCloudCheck, envchecks.GCPProviderCheck),
		CheckDestroy: checkProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				// Step 1: Create subscription with CMK enabled (enters encryption_key_pending state)
				// Also creates GCP KMS key and grants IAM permissions to the Redis service account
				ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
				ConfigFile:               config.StaticFile("./testdata/cmk_step1.tf"),
				ConfigVariables:          configVars,
				ExpectNonEmptyPlan:       true,
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
				ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
				ConfigFile:               config.StaticFile("./testdata/cmk_step2.tf"),
				ConfigVariables:          configVars,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectNonEmptyPlan(),
					},
					PostApplyPreRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				}, Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttrSet(resourceName, "customer_managed_key_redis_service_account"),
					resource.TestCheckResourceAttr(resourceName, "payment_method", "credit-card"),
					resource.TestCheckResourceAttrSet(resourceName, "payment_method_id"),
					resource.TestCheckResourceAttr(resourceName, "memory_storage", "ram"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider.0.provider"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider.0.region.#"),
					resource.TestCheckResourceAttr(resourceName, "customer_managed_key_enabled", "true"),
					checkNoCreationPlanDatabases(resourceName),
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.mode", "manual"),
					resource.TestCheckResourceAttrSet(resourceName, "maintenance_windows.0.window")),
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
