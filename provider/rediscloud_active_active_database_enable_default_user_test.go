package provider

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// TestAccResourceRedisCloudActiveActiveDatabase_enableDefaultUser tests the enable_default_user field
// for global and regional override behavior, specifically testing for drift issues when:
// - Regions inherit from global (field NOT in override_region) - Steps 1, 5
// - Regions explicitly override global (field IS in override_region) - Steps 2, 3
// - User explicitly sets same value as global (field IS in override_region, tests explicit vs inherited) - Steps 4, 6
//
// Tests all 6 combinations:
//   Step 1: global=true,  both inherit
//   Step 2: global=true,  one region=false, one inherit
//   Step 3: global=false, one region=true, one inherit
//   Step 4: global=true,  region1=true (matches), region2=false
//   Step 5: global=false, both inherit
//   Step 6: global=false, region1=false (matches), one inherit
func TestAccResourceRedisCloudActiveActiveDatabase_enableDefaultUser(t *testing.T) {
	subscriptionName := os.Getenv("AWS_TEST_CLOUD_ACCOUNT_NAME") + "-enable-default-user"
	databaseName := "tf-test-enable-default-user"
	databasePassword := "ThisIs!ATestPassword123"

	const databaseResourceName = "rediscloud_active_active_subscription_database.example"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			// Step 1: global=true, both regions inherit (NO enable_default_user in override_region)
			{
				Config: fmt.Sprintf(
					utils.GetTestConfig(t, "./activeactive/testdata/enable_default_user_global_true_inherit.tf"),
					subscriptionName,
					databaseName,
					databasePassword,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Global setting
					resource.TestCheckResourceAttr(databaseResourceName, "global_enable_default_user", "true"),

					// Both regions should exist
					resource.TestCheckResourceAttr(databaseResourceName, "override_region.#", "2"),

					// Neither region should have enable_default_user in state (inheriting from global)
					resource.TestCheckNoResourceAttr(databaseResourceName, "override_region.0.enable_default_user"),
					resource.TestCheckNoResourceAttr(databaseResourceName, "override_region.1.enable_default_user"),

					// API check: Both regions should have true (inherited from global)
					testCheckEnableDefaultUserInAPI(databaseResourceName, true, map[string]*bool{
						"us-east-1": nil, // nil means inherits from global
						"us-east-2": nil,
					}),
				),
			},
			// Step 2: global=true, us-east-1 explicit false (field SHOULD appear in override_region)
			{
				Config: fmt.Sprintf(
					utils.GetTestConfig(t, "./activeactive/testdata/enable_default_user_global_true_region_false.tf"),
					subscriptionName,
					databaseName,
					databasePassword,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Global setting
					resource.TestCheckResourceAttr(databaseResourceName, "global_enable_default_user", "true"),

					// Two regions
					resource.TestCheckResourceAttr(databaseResourceName, "override_region.#", "2"),

					// us-east-1 has explicit false (differs from global=true)
					// us-east-2 inherits (no explicit field)
					// Use TestCheckTypeSetElemNestedAttrs to verify specific TypeSet element fields
					resource.TestCheckTypeSetElemNestedAttrs(databaseResourceName, "override_region.*", map[string]string{
						"name":                "us-east-1",
						"enable_default_user": "false",
					}),

					// API check: us-east-1=false (explicit), us-east-2=true (inherited)
					testCheckEnableDefaultUserInAPI(databaseResourceName, true, map[string]*bool{
						"us-east-1": redis.Bool(false), // Explicit override
						"us-east-2": nil,                // Inherits from global
					}),
				),
			},
			// Step 3: global=false, us-east-1 explicit true (field SHOULD appear in override_region)
			{
				Config: fmt.Sprintf(
					utils.GetTestConfig(t, "./activeactive/testdata/enable_default_user_global_false_region_true.tf"),
					subscriptionName,
					databaseName,
					databasePassword,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Global setting
					resource.TestCheckResourceAttr(databaseResourceName, "global_enable_default_user", "false"),

					// Two regions
					resource.TestCheckResourceAttr(databaseResourceName, "override_region.#", "2"),

					// us-east-1 has explicit true (differs from global=false)
					resource.TestCheckTypeSetElemNestedAttrs(databaseResourceName, "override_region.*", map[string]string{
						"name":                "us-east-1",
						"enable_default_user": "true",
					}),
					// us-east-2 inherits (no explicit field)

					// API check: us-east-1=true (explicit), us-east-2=false (inherited)
					testCheckEnableDefaultUserInAPI(databaseResourceName, false, map[string]*bool{
						"us-east-1": redis.Bool(true), // Explicit override
						"us-east-2": nil,               // Inherits from global
					}),
				),
			},
			// Step 4: global=true, both regions explicit (us-east-1=true, us-east-2=false)
			// This tests that explicit values matching global are still preserved
			{
				Config: fmt.Sprintf(
					utils.GetTestConfig(t, "./activeactive/testdata/enable_default_user_all_explicit.tf"),
					subscriptionName,
					databaseName,
					databasePassword,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Global setting
					resource.TestCheckResourceAttr(databaseResourceName, "global_enable_default_user", "true"),

					// Two regions
					resource.TestCheckResourceAttr(databaseResourceName, "override_region.#", "2"),

					// Both regions have explicit enable_default_user - both should be in state
					// us-east-1 has true (matches global but explicit)
					resource.TestCheckTypeSetElemNestedAttrs(databaseResourceName, "override_region.*", map[string]string{
						"name":                "us-east-1",
						"enable_default_user": "true",
					}),
					// us-east-2 has false (differs from global)
					resource.TestCheckTypeSetElemNestedAttrs(databaseResourceName, "override_region.*", map[string]string{
						"name":                "us-east-2",
						"enable_default_user": "false",
					}),

					// API check: us-east-1=true (explicit), us-east-2=false (explicit)
					testCheckEnableDefaultUserInAPI(databaseResourceName, true, map[string]*bool{
						"us-east-1": redis.Bool(true),  // Explicit (matches global)
						"us-east-2": redis.Bool(false), // Explicit (differs from global)
					}),
				),
			},
			// Step 5: global=false, both regions inherit (NO enable_default_user in override_region)
			// Mirror of Step 1 but with global=false
			{
				Config: fmt.Sprintf(
					utils.GetTestConfig(t, "./activeactive/testdata/enable_default_user_global_false_inherit.tf"),
					subscriptionName,
					databaseName,
					databasePassword,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Global setting
					resource.TestCheckResourceAttr(databaseResourceName, "global_enable_default_user", "false"),

					// Both regions should exist
					resource.TestCheckResourceAttr(databaseResourceName, "override_region.#", "2"),

					// Neither region should have enable_default_user in state (inheriting from global)
					resource.TestCheckNoResourceAttr(databaseResourceName, "override_region.0.enable_default_user"),
					resource.TestCheckNoResourceAttr(databaseResourceName, "override_region.1.enable_default_user"),

					// API check: Both regions should have false (inherited from global)
					testCheckEnableDefaultUserInAPI(databaseResourceName, false, map[string]*bool{
						"us-east-1": nil, // Inherits from global
						"us-east-2": nil,
					}),
				),
			},
			// Step 6: global=false, us-east-1 explicit false (field SHOULD appear in override_region)
			// Tests explicit false matching global false (vs inheriting)
			{
				Config: fmt.Sprintf(
					utils.GetTestConfig(t, "./activeactive/testdata/enable_default_user_global_false_region_false.tf"),
					subscriptionName,
					databaseName,
					databasePassword,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Global setting
					resource.TestCheckResourceAttr(databaseResourceName, "global_enable_default_user", "false"),

					// Two regions
					resource.TestCheckResourceAttr(databaseResourceName, "override_region.#", "2"),

					// us-east-1 has explicit false (matches global but is EXPLICIT)
					resource.TestCheckTypeSetElemNestedAttrs(databaseResourceName, "override_region.*", map[string]string{
						"name":                "us-east-1",
						"enable_default_user": "false",
					}),
					// us-east-2 inherits (no explicit field)

					// API check: us-east-1=false (explicit, matches global), us-east-2=false (inherited)
					testCheckEnableDefaultUserInAPI(databaseResourceName, false, map[string]*bool{
						"us-east-1": redis.Bool(false), // Explicit (matches global)
						"us-east-2": nil,                // Inherits from global
					}),
				),
			},
		},
	})
}

// testCheckEnableDefaultUserInAPI verifies the enable_default_user values in the actual Redis Cloud API
// expectedGlobal: expected value for global_enable_default_user
// expectedRegions: map[regionName]expectedValue (nil means should inherit from global)
func testCheckEnableDefaultUserInAPI(resourceName string, expectedGlobal bool, expectedRegions map[string]*bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		// Parse subscription_id and db_id from resource
		subIdStr := rs.Primary.Attributes["subscription_id"]
		dbIdStr := rs.Primary.Attributes["db_id"]

		subId, err := strconv.Atoi(subIdStr)
		if err != nil {
			return fmt.Errorf("failed to parse subscription_id: %v", err)
		}

		dbId, err := strconv.Atoi(dbIdStr)
		if err != nil {
			return fmt.Errorf("failed to parse db_id: %v", err)
		}

		// Get API client
		apiClient, err := client.NewClient()
		if err != nil {
			return fmt.Errorf("failed to get API client: %v", err)
		}

		// Fetch database from API
		ctx := context.Background()
		db, err := apiClient.Client.Database.GetActiveActive(ctx, subId, dbId)
		if err != nil {
			return fmt.Errorf("failed to get database from API: %v", err)
		}

		// Check global enable_default_user
		if db.GlobalEnableDefaultUser == nil {
			return fmt.Errorf("API returned nil for GlobalEnableDefaultUser")
		}
		actualGlobal := redis.BoolValue(db.GlobalEnableDefaultUser)
		if actualGlobal != expectedGlobal {
			return fmt.Errorf("API global_enable_default_user: expected %v, got %v", expectedGlobal, actualGlobal)
		}

		// Check regional enable_default_user values
		for _, regionDb := range db.CrdbDatabases {
			regionName := redis.StringValue(regionDb.Region)

			if regionDb.Security == nil || regionDb.Security.EnableDefaultUser == nil {
				return fmt.Errorf("API returned nil for region %s EnableDefaultUser", regionName)
			}

			actualRegionValue := redis.BoolValue(regionDb.Security.EnableDefaultUser)

			// Get expected value for this region
			expectedValue, hasExplicitOverride := expectedRegions[regionName]

			var expectedRegionValue bool
			if hasExplicitOverride && expectedValue != nil {
				// Region has explicit override
				expectedRegionValue = *expectedValue
			} else {
				// Region inherits from global
				expectedRegionValue = expectedGlobal
			}

			if actualRegionValue != expectedRegionValue {
				return fmt.Errorf("API region %s enable_default_user: expected %v, got %v",
					regionName, expectedRegionValue, actualRegionValue)
			}
		}

		return nil
	}
}
