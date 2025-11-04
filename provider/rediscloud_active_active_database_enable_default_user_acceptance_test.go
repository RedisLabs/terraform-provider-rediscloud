package provider

import (
	"fmt"
	"testing"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// TestAccResourceRedisCloudActiveActiveDatabase_enableDefaultUser tests the enable_default_user
// field behavior at both global and regional levels, ensuring proper three-state logic
func TestAccResourceRedisCloudActiveActiveDatabase_enableDefaultUser(t *testing.T) {
	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	subscriptionName := acctest.RandomWithPrefix(testResourcePrefix) + "-enable-default-user"
	databaseName := "test-enable-default-user"
	password := acctest.RandString(20)

	const resourceName = "rediscloud_active_active_subscription_database.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create with global=true, regions inherit (THE BUG SCENARIO)
			{
				Config: fmt.Sprintf(
					utils.GetTestConfig(t, "./activeactive/testdata/enable_default_user_global_true_inherit.tf"),
					subscriptionName, databaseName, password),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify Terraform state
					resource.TestCheckResourceAttr(resourceName, "name", databaseName),
					resource.TestCheckResourceAttr(resourceName, "global_enable_default_user", "true"),
					resource.TestCheckResourceAttr(resourceName, "override_region.#", "2"),

					// Both regions inherit global (no explicit enable_default_user set)
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "override_region.*", map[string]string{
						"name": "us-east-1",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "override_region.*", map[string]string{
						"name": "us-east-2",
					}),
				),
			},

			// Step 2: Update to mixed overrides - some regions inherit, some override
			{
				Config: fmt.Sprintf(
					utils.GetTestConfig(t, "./activeactive/testdata/enable_default_user_mixed_overrides.tf"),
					subscriptionName, databaseName, password),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "global_enable_default_user", "true"),
					resource.TestCheckResourceAttr(resourceName, "override_region.#", "2"),

					// us-east-1: explicitly false (override)
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "override_region.*", map[string]string{
						"name":                "us-east-1",
						"enable_default_user": "false",
					}),

					// us-east-2: inherits global (no enable_default_user set)
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "override_region.*", map[string]string{
						"name": "us-east-2",
					}),
				),
			},

			// Step 3: Global false, specific regions enable (CRITICAL USE CASE)
			{
				Config: fmt.Sprintf(
					utils.GetTestConfig(t, "./activeactive/testdata/enable_default_user_global_false_region_true.tf"),
					subscriptionName, databaseName, password),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "global_enable_default_user", "false"),
					resource.TestCheckResourceAttr(resourceName, "override_region.#", "2"),

					// us-east-1: explicitly true (override global false)
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "override_region.*", map[string]string{
						"name":                "us-east-1",
						"enable_default_user": "true",
					}),

					// us-east-2: inherits global false (no enable_default_user set)
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "override_region.*", map[string]string{
						"name": "us-east-2",
					}),
				),
			},

			// Step 4: Mixed - one region overrides to false
			{
				Config: fmt.Sprintf(
					utils.GetTestConfig(t, "./activeactive/testdata/enable_default_user_all_explicit.tf"),
					subscriptionName, databaseName, password),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "global_enable_default_user", "true"),
					resource.TestCheckResourceAttr(resourceName, "override_region.#", "2"),

					// us-east-1: explicitly true (but matches global, so won't be in state)
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "override_region.*", map[string]string{
						"name": "us-east-1",
					}),

					// us-east-2: explicitly false (differs from global, so IS in state)
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "override_region.*", map[string]string{
						"name":                "us-east-2",
						"enable_default_user": "false",
					}),
				),
			},
		},
	})
}
