package provider

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/RedisLabs/rediscloud-go-api/service/databases"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// TestAccResourceRedisCloudActiveActiveDatabase_enableDefaultUser tests the enable_default_user
// field behavior at both global and regional levels, ensuring proper three-state logic
func TestAccResourceRedisCloudActiveActiveDatabase_enableDefaultUser(t *testing.T) {
	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")

	subscriptionName := acctest.RandomWithPrefix(testResourcePrefix) + "-enable-default-user"
	databaseName := "test-enable-default-user"
	password := acctest.RandString(20)

	const resourceName = "rediscloud_active_active_subscription_database.test"
	const subscriptionResourceName = "rediscloud_active_active_subscription.test"

	var subId int
	var dbId int

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

					// Capture subscription and database IDs for API verification
					func(s *terraform.State) error {
						r := s.RootModule().Resources[subscriptionResourceName]
						var err error
						subId, err = strconv.Atoi(r.Primary.ID)
						if err != nil {
							return fmt.Errorf("couldn't parse subscription ID: %s", r.Primary.ID)
						}

						dbResource := s.RootModule().Resources[resourceName]
						dbIdStr := dbResource.Primary.Attributes["db_id"]
						dbId, err = strconv.Atoi(dbIdStr)
						if err != nil {
							return fmt.Errorf("couldn't parse database ID: %s", dbIdStr)
						}

						return nil
					},

					// Verify API state - regions should inherit global (not send enableDefaultUser)
					func(s *terraform.State) error {
						apiClient := testProvider.Meta().(*client.ApiClient)
						db, err := apiClient.Client.Database.GetActiveActive(context.TODO(), subId, dbId)
						if err != nil {
							return fmt.Errorf("failed to get database from API: %w", err)
						}

						// Verify global setting
						if db.GlobalEnableDefaultUser == nil || !*db.GlobalEnableDefaultUser {
							return fmt.Errorf("expected GlobalEnableDefaultUser to be true, got: %v", db.GlobalEnableDefaultUser)
						}

						// Verify regions - they should have enableDefaultUser=true (effective value from global)
						// What's important is that all regions show true (not false from bad override)
						for _, regionDb := range db.CrdbDatabases {
							region := redis.StringValue(regionDb.Region)
							enableDefaultUser := redis.BoolValue(regionDb.Security.EnableDefaultUser)
							t.Logf("Region %s: EnableDefaultUser = %v", region, enableDefaultUser)

							// All regions should have effective value of true (from global)
							if !enableDefaultUser {
								return fmt.Errorf("region %s has enableDefaultUser=%v, expected true (inherited from global)",
									region, enableDefaultUser)
							}
						}

						return nil
					},
				),
			},

			// Step 2: Update to mixed overrides - some regions inherit, some override
			{
				Config: fmt.Sprintf(
					utils.GetTestConfig(t, "./activeactive/testdata/enable_default_user_mixed_overrides.tf"),
					subscriptionName, databaseName, password),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "global_enable_default_user", "true"),

					// us-east-1: explicitly false (override)
					// Note: Terraform state won't show enable_default_user for override_region unless we check differently

					// Verify API state
					func(s *terraform.State) error {
						apiClient := testProvider.Meta().(*client.ApiClient)
						db, err := apiClient.Client.Database.GetActiveActive(context.TODO(), subId, dbId)
						if err != nil {
							return fmt.Errorf("failed to get database from API: %w", err)
						}

						// Verify global is still true
						if !redis.BoolValue(db.GlobalEnableDefaultUser) {
							return fmt.Errorf("expected GlobalEnableDefaultUser=true")
						}

						// Verify us-east-1 is explicitly false
						usEast1 := findRegionInActiveActiveDB(db, "us-east-1")
						if usEast1 == nil {
							return fmt.Errorf("us-east-1 region not found")
						}
						if redis.BoolValue(usEast1.Security.EnableDefaultUser) != false {
							return fmt.Errorf("us-east-1 should have enableDefaultUser=false, got: %v",
								usEast1.Security.EnableDefaultUser)
						}

						// Verify us-east-2 inherits (true)
						usEast2 := findRegionInActiveActiveDB(db, "us-east-2")
						if usEast2 == nil {
							return fmt.Errorf("us-east-2 region not found")
						}
						if redis.BoolValue(usEast2.Security.EnableDefaultUser) != true {
							return fmt.Errorf("us-east-2 should inherit enableDefaultUser=true, got: %v",
								usEast2.Security.EnableDefaultUser)
						}

						return nil
					},
				),
			},

			// Step 3: Global false, specific regions enable (CRITICAL USE CASE)
			{
				Config: fmt.Sprintf(
					utils.GetTestConfig(t, "./activeactive/testdata/enable_default_user_global_false_region_true.tf"),
					subscriptionName, databaseName, password),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "global_enable_default_user", "false"),

					// Verify API state
					func(s *terraform.State) error {
						apiClient := testProvider.Meta().(*client.ApiClient)
						db, err := apiClient.Client.Database.GetActiveActive(context.TODO(), subId, dbId)
						if err != nil {
							return fmt.Errorf("failed to get database from API: %w", err)
						}

						// Verify global is false
						if redis.BoolValue(db.GlobalEnableDefaultUser) != false {
							return fmt.Errorf("expected GlobalEnableDefaultUser=false, got: %v", db.GlobalEnableDefaultUser)
						}

						// Verify us-east-1 overrides to true
						usEast1 := findRegionInActiveActiveDB(db, "us-east-1")
						if usEast1 == nil {
							return fmt.Errorf("us-east-1 region not found")
						}
						if redis.BoolValue(usEast1.Security.EnableDefaultUser) != true {
							return fmt.Errorf("us-east-1 should override to enableDefaultUser=true, got: %v",
								usEast1.Security.EnableDefaultUser)
						}

						// Verify us-east-2 inherits false
						usEast2 := findRegionInActiveActiveDB(db, "us-east-2")
						if usEast2 == nil {
							return fmt.Errorf("us-east-2 region not found")
						}
						if redis.BoolValue(usEast2.Security.EnableDefaultUser) != false {
							return fmt.Errorf("us-east-2 should inherit enableDefaultUser=false, got: %v",
								usEast2.Security.EnableDefaultUser)
						}

						return nil
					},
				),
			},

			// Step 4: All explicit values (both true and false overrides)
			{
				Config: fmt.Sprintf(
					utils.GetTestConfig(t, "./activeactive/testdata/enable_default_user_all_explicit.tf"),
					subscriptionName, databaseName, password),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "global_enable_default_user", "true"),

					// Verify API state
					func(s *terraform.State) error {
						apiClient := testProvider.Meta().(*client.ApiClient)
						db, err := apiClient.Client.Database.GetActiveActive(context.TODO(), subId, dbId)
						if err != nil {
							return fmt.Errorf("failed to get database from API: %w", err)
						}

						usEast1 := findRegionInActiveActiveDB(db, "us-east-1")
						if redis.BoolValue(usEast1.Security.EnableDefaultUser) != true {
							return fmt.Errorf("us-east-1 should be true")
						}

						usEast2 := findRegionInActiveActiveDB(db, "us-east-2")
						if redis.BoolValue(usEast2.Security.EnableDefaultUser) != false {
							return fmt.Errorf("us-east-2 should be false")
						}

						return nil
					},
				),
			},
		},
	})
}

// Helper function to find a specific region in the ActiveActive database API response
func findRegionInActiveActiveDB(db *databases.ActiveActiveDatabase, regionName string) *databases.CrdbDatabase {
	if db == nil {
		return nil
	}

	for _, regionDb := range db.CrdbDatabases {
		if redis.StringValue(regionDb.Region) == regionName {
			return regionDb
		}
	}
	return nil
}
