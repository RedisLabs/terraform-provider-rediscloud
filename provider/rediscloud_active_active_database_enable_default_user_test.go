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
// - Regions inherit from global (field NOT in override_region)
// - Regions explicitly override global (field IS in override_region)
// - User explicitly sets same value as global (field IS in override_region, tests explicit vs inherited)
// - User removes explicit overrides (explicit → inherit transition)
// - User adds explicit overrides (inherit → explicit transition)
//
// Tests all 6 combinations of the behavior matrix with 3 regions:
//   Step 1: global=true,  region1=explicit true, region2=explicit false, region3=inherit
//   Step 2: global=false, region1=explicit true, region2=explicit false, region3=inherit
//   Step 3: global=false, all 3 regions inherit (tests REMOVAL: explicit → inherit)
//   Step 4: global=true,  region1=explicit false, region2&3=inherit (tests ADDITION: inherit → explicit)
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
			// Step 1: global=true, 3 regions with mixed explicit/inherit
			// Tests all 3 behaviors: explicit matching global, explicit differing, and inheritance
			{
				Config: fmt.Sprintf(
					utils.GetTestConfig(t, "./activeactive/testdata/enable_default_user_step1_global_true_mixed.tf"),
					subscriptionName,
					databaseName,
					databasePassword,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Global setting
					resource.TestCheckResourceAttr(databaseResourceName, "global_enable_default_user", "true"),

					// Three regions
					resource.TestCheckResourceAttr(databaseResourceName, "override_region.#", "3"),

					// us-east-1: explicit true (matches global)
					resource.TestCheckTypeSetElemNestedAttrs(databaseResourceName, "override_region.*", map[string]string{
						"name":                "us-east-1",
						"enable_default_user": "true",
					}),

					// us-east-2: explicit false (differs from global)
					resource.TestCheckTypeSetElemNestedAttrs(databaseResourceName, "override_region.*", map[string]string{
						"name":                "us-east-2",
						"enable_default_user": "false",
					}),

					// eu-west-2: inherits (no enable_default_user in state)
					// Note: We can't use TestCheckTypeSetElemNestedAttrs with absent fields
					// The API check below verifies inheritance works correctly

					// API check: verify actual values
					testCheckEnableDefaultUserInAPI(databaseResourceName, true, map[string]*bool{
						"us-east-1": redis.Bool(true),  // Explicit (matches global)
						"us-east-2": redis.Bool(false), // Explicit (differs from global)
						"eu-west-2": nil,                // Inherits from global=true
					}),
				),
			},
			// Step 2: global=false, 3 regions with mixed explicit/inherit
			// Tests global flip with same explicit/inherit pattern
			{
				Config: fmt.Sprintf(
					utils.GetTestConfig(t, "./activeactive/testdata/enable_default_user_step2_global_false_mixed.tf"),
					subscriptionName,
					databaseName,
					databasePassword,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Global setting
					resource.TestCheckResourceAttr(databaseResourceName, "global_enable_default_user", "false"),

					// Three regions
					resource.TestCheckResourceAttr(databaseResourceName, "override_region.#", "3"),

					// us-east-1: explicit true (differs from global)
					resource.TestCheckTypeSetElemNestedAttrs(databaseResourceName, "override_region.*", map[string]string{
						"name":                "us-east-1",
						"enable_default_user": "true",
					}),

					// us-east-2: explicit false (matches global)
					resource.TestCheckTypeSetElemNestedAttrs(databaseResourceName, "override_region.*", map[string]string{
						"name":                "us-east-2",
						"enable_default_user": "false",
					}),

					// eu-west-2: inherits (no enable_default_user in state)

					// API check: verify actual values
					testCheckEnableDefaultUserInAPI(databaseResourceName, false, map[string]*bool{
						"us-east-1": redis.Bool(true),  // Explicit (differs from global)
						"us-east-2": redis.Bool(false), // Explicit (matches global)
						"eu-west-2": nil,                // Inherits from global=false
					}),
				),
			},
			// Step 3: global=false, all 3 regions inherit
			// CRITICAL TEST: Removal scenario (explicit → inherit)
			// Verifies that removing explicit overrides from config:
			// - Removes fields from state
			// - Doesn't cause drift
			// - Regions correctly inherit from global
			{
				Config: fmt.Sprintf(
					utils.GetTestConfig(t, "./activeactive/testdata/enable_default_user_step3_all_inherit.tf"),
					subscriptionName,
					databaseName,
					databasePassword,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Global setting
					resource.TestCheckResourceAttr(databaseResourceName, "global_enable_default_user", "false"),

					// Three regions
					resource.TestCheckResourceAttr(databaseResourceName, "override_region.#", "3"),

					// All regions inherit - NO enable_default_user in state
					// We verify this by checking the API returns inherited values
					resource.TestCheckNoResourceAttr(databaseResourceName, "override_region.0.enable_default_user"),
					resource.TestCheckNoResourceAttr(databaseResourceName, "override_region.1.enable_default_user"),
					resource.TestCheckNoResourceAttr(databaseResourceName, "override_region.2.enable_default_user"),

					// API check: All regions inherit from global=false
					testCheckEnableDefaultUserInAPI(databaseResourceName, false, map[string]*bool{
						"us-east-1": nil, // Inherits from global
						"us-east-2": nil, // Inherits from global
						"eu-west-2": nil, // Inherits from global
					}),
				),
			},
			// Step 4: global=true, add explicit override to one region
			// Tests addition scenario (inherit → explicit)
			// Verifies that adding explicit override after removal works correctly
			{
				Config: fmt.Sprintf(
					utils.GetTestConfig(t, "./activeactive/testdata/enable_default_user_step4_one_explicit.tf"),
					subscriptionName,
					databaseName,
					databasePassword,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Global setting
					resource.TestCheckResourceAttr(databaseResourceName, "global_enable_default_user", "true"),

					// Three regions
					resource.TestCheckResourceAttr(databaseResourceName, "override_region.#", "3"),

					// us-east-1: explicit false (differs from global)
					resource.TestCheckTypeSetElemNestedAttrs(databaseResourceName, "override_region.*", map[string]string{
						"name":                "us-east-1",
						"enable_default_user": "false",
					}),

					// us-east-2 and eu-west-2: inherit (no enable_default_user in state)

					// API check: us-east-1 explicit, others inherit
					testCheckEnableDefaultUserInAPI(databaseResourceName, true, map[string]*bool{
						"us-east-1": redis.Bool(false), // Explicit (differs from global)
						"us-east-2": nil,                // Inherits from global=true
						"eu-west-2": nil,                // Inherits from global=true
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
				inheritStr := ""
				if hasExplicitOverride && expectedValue != nil {
					inheritStr = fmt.Sprintf(" (explicit override in config)")
				} else {
					inheritStr = fmt.Sprintf(" (should inherit from global=%v)", expectedGlobal)
				}

				// Build a detailed error message showing all regions
				errorMsg := fmt.Sprintf("API region %s enable_default_user mismatch:\n", regionName)
				errorMsg += fmt.Sprintf("  Expected: %v%s\n", expectedRegionValue, inheritStr)
				errorMsg += fmt.Sprintf("  Actual:   %v\n", actualRegionValue)
				errorMsg += fmt.Sprintf("\nGlobal enable_default_user: %v\n", actualGlobal)
				errorMsg += fmt.Sprintf("\nAll regions in API:")
				for _, r := range db.CrdbDatabases {
					rName := redis.StringValue(r.Region)
					rValue := "nil"
					if r.Security != nil && r.Security.EnableDefaultUser != nil {
						rValue = fmt.Sprintf("%v", redis.BoolValue(r.Security.EnableDefaultUser))
					}
					errorMsg += fmt.Sprintf("\n  - %s: %s", rName, rValue)
				}

				return fmt.Errorf(errorMsg)
			}
		}

		return nil
	}
}
