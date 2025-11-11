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

// TestAccResourceRedisCloudActiveActiveDatabase_enableDefaultUserImport is a DEBUG version
// that imports and modifies an existing database to speed up testing during development.
//
// SETUP:
// 1. Set DEBUG_SUBSCRIPTION_ID and DEBUG_DATABASE_ID environment variables
// 2. The database must have us-east-1 and us-east-2 regions
// 3. Run with: DEBUG_SUBSCRIPTION_ID=124134 DEBUG_DATABASE_ID=4923 EXECUTE_TESTS=true make testacc TESTARGS='-run=TestAccResourceRedisCloudActiveActiveDatabase_enableDefaultUserImport'
//
// This test will:
// - Import the existing database
// - Update it through 3 test steps to test enable_default_user drift detection
// - Leave the database in place (no destroy)
func TestAccResourceRedisCloudActiveActiveDatabase_enableDefaultUserImport(t *testing.T) {
	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	subscriptionID := os.Getenv("DEBUG_SUBSCRIPTION_ID")
	databaseID := os.Getenv("DEBUG_DATABASE_ID")

	if subscriptionID == "" || databaseID == "" {
		t.Skip("DEBUG_SUBSCRIPTION_ID and DEBUG_DATABASE_ID must be set - skipping import debug test")
	}

	// Get the actual database name and password from API
	apiClient, err := client.NewClient()
	if err != nil {
		t.Fatalf("Failed to create API client: %v", err)
	}

	subId, _ := strconv.Atoi(subscriptionID)
	dbId, _ := strconv.Atoi(databaseID)
	ctx := context.Background()

	db, err := apiClient.Client.Database.GetActiveActive(ctx, subId, dbId)
	if err != nil {
		t.Fatalf("Failed to fetch database %s/%s: %v", subscriptionID, databaseID, err)
	}

	databaseName := redis.StringValue(db.Name)
	databasePassword := acctest.RandString(20) // Use new password for testing

	const databaseResourceName = "rediscloud_active_active_subscription_database.example"
	importID := fmt.Sprintf("%s/%s", subscriptionID, databaseID)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			// Step 0: Import existing database
			{
				PreConfig: func() {
					t.Logf("DEBUG Step 0: Importing database %s (sub: %s, db: %s)", databaseName, subscriptionID, databaseID)
				},
				Config: fmt.Sprintf(
					utils.GetTestConfig(t, "./activeactive/testdata/enable_default_user_debug_import_step1.tf"),
					subscriptionID,
					databasePassword,
				),
				ResourceName:      databaseResourceName,
				ImportState:       true,
				ImportStateId:     importID,
				ImportStateVerify: false, // Don't verify all fields, just get it into state
			},
			// Step 1: global=true, both regions inherit
			{
				PreConfig: func() {
					t.Logf("DEBUG Step 1: global=true, both inherit")
				},
				Config: fmt.Sprintf(
					utils.GetTestConfig(t, "./activeactive/testdata/enable_default_user_debug_import_step1.tf"),
					subscriptionID,
					databasePassword,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Global setting
					resource.TestCheckResourceAttr(databaseResourceName, "global_enable_default_user", "true"),
					resource.TestCheckResourceAttr(databaseResourceName, "subscription_id", subscriptionID),
					resource.TestCheckResourceAttr(databaseResourceName, "db_id", databaseID),

					// Both regions should exist
					resource.TestCheckResourceAttr(databaseResourceName, "override_region.#", "2"),

					// Neither region should have enable_default_user in state
					resource.TestCheckNoResourceAttr(databaseResourceName, "override_region.0.enable_default_user"),
					resource.TestCheckNoResourceAttr(databaseResourceName, "override_region.1.enable_default_user"),

					// API check
					testCheckEnableDefaultUserInAPIImport(databaseResourceName, true, map[string]*bool{
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
					utils.GetTestConfig(t, "./activeactive/testdata/enable_default_user_debug_import_step2.tf"),
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
					testCheckEnableDefaultUserInAPIImport(databaseResourceName, true, map[string]*bool{
						"us-east-1": redis.Bool(false),
						"us-east-2": nil,
					}),
				),
			},
			// Step 3: global=false, us-east-1 explicit true
			{
				PreConfig: func() {
					t.Logf("DEBUG Step 3: global=false, us-east-1 explicit true")
				},
				Config: fmt.Sprintf(
					utils.GetTestConfig(t, "./activeactive/testdata/enable_default_user_debug_import_step3.tf"),
					subscriptionID,
					databaseName,
					databasePassword,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Global setting
					resource.TestCheckResourceAttr(databaseResourceName, "global_enable_default_user", "false"),

					// Two regions
					resource.TestCheckResourceAttr(databaseResourceName, "override_region.#", "2"),

					// us-east-1 has explicit true
					resource.TestCheckTypeSetElemNestedAttrs(databaseResourceName, "override_region.*", map[string]string{
						"name":                "us-east-1",
						"enable_default_user": "true",
					}),

					// API check
					testCheckEnableDefaultUserInAPIImport(databaseResourceName, false, map[string]*bool{
						"us-east-1": redis.Bool(true),
						"us-east-2": nil,
					}),
				),
			},
		},
	})
}

// testCheckEnableDefaultUserInAPIImport is identical to the regular version but for the import test
func testCheckEnableDefaultUserInAPIImport(resourceName string, expectedGlobal bool, expectedRegions map[string]*bool) resource.TestCheckFunc {
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
