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
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// TestAccResourceRedisCloudActiveActiveDatabase_enableDefaultUserDebug is a DEBUG version
// that reuses an existing subscription to speed up testing during development.
//
// SETUP:
// 1. Set DEBUG_SUBSCRIPTION_ID environment variable to an existing AA subscription ID
// 2. The subscription must have us-east-1 and us-east-2 regions
// 3. Run with: DEBUG_SUBSCRIPTION_ID=12345 EXECUTE_TESTS=true make testacc TESTARGS='-run=TestAccResourceRedisCloudActiveActiveDatabase_enableDefaultUserDebug'
//
// This test will:
// - Create a new database in the existing subscription
// - Update it through 2 test steps (global=true variants)
// - Delete the database at the end
func TestAccResourceRedisCloudActiveActiveDatabase_enableDefaultUserDebug(t *testing.T) {
	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	subscriptionID := os.Getenv("DEBUG_SUBSCRIPTION_ID")
	if subscriptionID == "" {
		t.Skip("DEBUG_SUBSCRIPTION_ID not set - skipping debug test")
	}

	databaseName := acctest.RandomWithPrefix("debug-enable-default-user")
	databasePassword := acctest.RandString(20)

	const databaseResourceName = "rediscloud_active_active_subscription_database.example"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		CheckDestroy:      testAccCheckActiveActiveDatabaseDestroy,
		Steps: []resource.TestStep{
			// Step 1: global=true, both regions inherit
			{
				PreConfig: func() {
					t.Logf("DEBUG Step 1: global=true, both inherit (subscription: %s, database: %s)", subscriptionID, databaseName)
				},
				Config: fmt.Sprintf(
					utils.GetTestConfig(t, "./activeactive/testdata/enable_default_user_debug_step1.tf"),
					subscriptionID,
					databaseName,
					databasePassword,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Global setting
					resource.TestCheckResourceAttr(databaseResourceName, "global_enable_default_user", "true"),
					resource.TestCheckResourceAttr(databaseResourceName, "subscription_id", subscriptionID),

					// Both regions should exist
					resource.TestCheckResourceAttr(databaseResourceName, "override_region.#", "2"),

					// Neither region should have enable_default_user in state
					resource.TestCheckNoResourceAttr(databaseResourceName, "override_region.0.enable_default_user"),
					resource.TestCheckNoResourceAttr(databaseResourceName, "override_region.1.enable_default_user"),

					// API check
					testCheckEnableDefaultUserInAPIDebug(databaseResourceName, true, map[string]*bool{
						"us-east-1": nil,
						"us-east-2": nil,
					}),
				),
			},
			// Step 2: global=true, us-east-1 explicit false
			{
				PreConfig: func() {
					t.Logf("DEBUG Step 2: global=true, us-east-1 explicit false")
				},
				Config: fmt.Sprintf(
					utils.GetTestConfig(t, "./activeactive/testdata/enable_default_user_debug_step2.tf"),
					subscriptionID,
					databaseName,
					databasePassword,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Global setting
					resource.TestCheckResourceAttr(databaseResourceName, "global_enable_default_user", "true"),

					// Two regions
					resource.TestCheckResourceAttr(databaseResourceName, "override_region.#", "2"),

					// us-east-1 has explicit false
					resource.TestCheckTypeSetElemNestedAttrs(databaseResourceName, "override_region.*", map[string]string{
						"name":                "us-east-1",
						"enable_default_user": "false",
					}),

					// API check
					testCheckEnableDefaultUserInAPIDebug(databaseResourceName, true, map[string]*bool{
						"us-east-1": redis.Bool(false),
						"us-east-2": nil,
					}),
				),
			},
		},
	})
}

// testCheckEnableDefaultUserInAPIDebug is identical to the regular version but for the debug test
func testCheckEnableDefaultUserInAPIDebug(resourceName string, expectedGlobal bool, expectedRegions map[string]*bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

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

		apiClient, err := client.NewClient()
		if err != nil {
			return fmt.Errorf("failed to get API client: %v", err)
		}

		ctx := context.Background()
		db, err := apiClient.Client.Database.GetActiveActive(ctx, subId, dbId)
		if err != nil {
			return fmt.Errorf("failed to get database from API: %v", err)
		}

		if db.GlobalEnableDefaultUser == nil {
			return fmt.Errorf("API returned nil for GlobalEnableDefaultUser")
		}
		actualGlobal := redis.BoolValue(db.GlobalEnableDefaultUser)
		if actualGlobal != expectedGlobal {
			return fmt.Errorf("API global_enable_default_user: expected %v, got %v", expectedGlobal, actualGlobal)
		}

		for _, regionDb := range db.CrdbDatabases {
			regionName := redis.StringValue(regionDb.Region)

			if regionDb.Security == nil || regionDb.Security.EnableDefaultUser == nil {
				return fmt.Errorf("API returned nil for region %s EnableDefaultUser", regionName)
			}

			actualRegionValue := redis.BoolValue(regionDb.Security.EnableDefaultUser)

			expectedValue, hasExplicitOverride := expectedRegions[regionName]

			var expectedRegionValue bool
			if hasExplicitOverride && expectedValue != nil {
				expectedRegionValue = *expectedValue
			} else {
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

// testAccCheckActiveActiveDatabaseDestroy verifies the database was destroyed (subscription remains)
func testAccCheckActiveActiveDatabaseDestroy(s *terraform.State) error {
	apiClient, err := client.NewClient()
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "rediscloud_active_active_subscription_database" {
			continue
		}

		subId, err := strconv.Atoi(rs.Primary.Attributes["subscription_id"])
		if err != nil {
			continue
		}

		dbId, err := strconv.Atoi(rs.Primary.Attributes["db_id"])
		if err != nil {
			continue
		}

		ctx := context.Background()
		db, err := apiClient.Client.Database.GetActiveActive(ctx, subId, dbId)
		if err != nil {
			// Database not found is expected
			if _, ok := err.(*redis.NotFound); ok {
				continue
			}
			return err
		}

		if db != nil {
			return fmt.Errorf("database %d still exists in subscription %d", dbId, subId)
		}
	}

	return nil
}
