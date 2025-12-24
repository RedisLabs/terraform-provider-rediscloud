package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

// TestAccResourceRedisCloudActiveActiveDatabase_enableDefaultUser tests the enable_default_user
// field behaviour at both global and regional levels, ensuring proper three-state logic:
// - Global true, regions inherit (no enable_default_user in override_region)
// - Global true, region explicitly overrides to false
// - Global false, region explicitly overrides to true
// - Mixed: some regions inherit, some override
func TestAccResourceRedisCloudActiveActiveDatabase_enableDefaultUser(t *testing.T) {
	subscriptionName := acctest.RandomWithPrefix(testResourcePrefix) + "-enable-default-user"
	databaseName := acctest.RandomWithPrefix(testResourcePrefix) + "-database"
	password := acctest.RandString(20)

	const resourceName = "rediscloud_active_active_subscription_database.test"

	placeholders := map[string]string{
		"__SUBSCRIPTION_NAME__": subscriptionName,
		"__DATABASE_NAME__":     databaseName,
		"__PASSWORD__":          password,
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create with global=true, regions inherit
			// Both regions should NOT have enable_default_user in state (inheriting from global)
			{
				Config: utils.RenderTestConfig(t, "./activeactive/testdata/enable_default_user_global_true_inherit.tf", placeholders),
				Check: resource.ComposeAggregateTestCheckFunc(
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

			// Step 2: Update to mixed overrides - us-east-1 overrides to false, us-east-2 inherits
			{
				Config: utils.RenderTestConfig(t, "./activeactive/testdata/enable_default_user_mixed_overrides.tf", placeholders),
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

			// Step 3: Global false, us-east-1 overrides to true, us-east-2 inherits false
			{
				Config: utils.RenderTestConfig(t, "./activeactive/testdata/enable_default_user_global_false_region_true.tf", placeholders),
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

			// Step 4: Both regions explicit - us-east-1 true (matches global), us-east-2 false (differs)
			// Key test: us-east-1's enable_default_user=true should NOT appear in state since it matches global
			{
				Config: utils.RenderTestConfig(t, "./activeactive/testdata/enable_default_user_all_explicit.tf", placeholders),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "global_enable_default_user", "true"),
					resource.TestCheckResourceAttr(resourceName, "override_region.#", "2"),

					// us-east-1: explicitly true but matches global, so should NOT be in state
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
