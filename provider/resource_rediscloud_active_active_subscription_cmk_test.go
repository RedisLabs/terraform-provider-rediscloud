package provider_test

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

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

// TestAccResourceRedisCloudActiveActiveSubscription_CMK is a fully automated CMK test
// that uses the GCP provider to create KMS keys and grant IAM permissions automatically.
func TestAccResourceRedisCloudActiveActiveSubscription_CMK(t *testing.T) {

	name := testRandomWithPrefix()
	const resourceName = "rediscloud_active_active_subscription.example"
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
		PreCheck:     func() { testAccPreCheck(t); testAccGcpProjectPreCheck(t); testAccGcpCredentialsPreCheck(t) },
		CheckDestroy: testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				// Step 1: Create subscription with CMK enabled (enters encryption_key_pending state)
				// Also creates GCP KMS key and grants IAM permissions to the Redis service account
				ProtoV5ProviderFactories: protoV5ProviderFactories,
				ConfigFile:               config.StaticFile("./activeactive/testdata/cmk_step1.tf"),
				ConfigVariables:          configVars,
				ExpectNonEmptyPlan:       true,
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
				ProtoV5ProviderFactories: protoV5ProviderFactories,
				ConfigFile:               config.StaticFile("./activeactive/testdata/cmk_step2.tf"),
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
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider"),
					resource.TestCheckResourceAttr(resourceName, "customer_managed_key_enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.mode", "manual"),
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.window.#", "1"),
					testAccCheckNoCreationPlanDatabases(resourceName),
				),
			},
		},
	})
}

// TestAccResourceRedisCloudActiveActiveSubscription_CMK_AWS is a fully automated AWS CMK test.
// It uses the hashicorp/aws external provider to create a multi-region KMS key
// (primary in us-east-1, replica in us-east-2) and key policies in-fixture,
// removing the need for a pre-existing AWS_CMK_KEY_ARN.
func TestAccResourceRedisCloudActiveActiveSubscription_CMK_AWS(t *testing.T) {

	name := testRandomWithPrefix() + "-aa-cmk-aws"
	const resourceName = "rediscloud_active_active_subscription.example"

	configVars := config.Variables{
		"name": config.StringVariable(name),
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
		PreCheck: func() {
			testAccPreCheck(t)
			testAccAwsPreExistingCloudAccountPreCheck(t)
			testAccAwsApiCredsPreCheck(t)
		},
		CheckDestroy: testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				// Step 1: subscription enters encryption_key_pending; both key policies
				// (primary + replica) reference the role ARN returned by the subscription.
				ProtoV5ProviderFactories: protoV5ProviderFactories,
				ConfigFile:               config.StaticFile("./activeactive/testdata/cmk_aws_step1.tf"),
				ConfigVariables:          configVars,
				ExpectNonEmptyPlan:       true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttrSet(resourceName, "customer_managed_key_aws_role_arn"),
					resource.TestCheckResourceAttr(resourceName, "payment_method", "credit-card"),
					resource.TestCheckResourceAttrSet(resourceName, "payment_method_id"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider"),
					resource.TestCheckResourceAttr(resourceName, "customer_managed_key_enabled", "true"),
				),
			},
			{
				// Step 2: subscription supplies the per-region KMS ARNs (primary + replica)
				// and transitions out of encryption_key_pending.
				ProtoV5ProviderFactories: protoV5ProviderFactories,
				ConfigFile:               config.StaticFile("./activeactive/testdata/cmk_aws_step2.tf"),
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
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttrSet(resourceName, "customer_managed_key_aws_role_arn"),
					resource.TestCheckResourceAttr(resourceName, "payment_method", "credit-card"),
					resource.TestCheckResourceAttrSet(resourceName, "payment_method_id"),
					resource.TestCheckResourceAttrSet(resourceName, "cloud_provider"),
					resource.TestCheckResourceAttr(resourceName, "customer_managed_key_enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.mode", "manual"),
					resource.TestCheckResourceAttr(resourceName, "maintenance_windows.0.window.#", "1"),
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
