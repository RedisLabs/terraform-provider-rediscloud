package provider

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

// TestAccResourceRedisCloudActiveActiveDatabase_enableDefaultUserInheritance tests the fix for
// the bug where regions that don't explicitly set enable_default_user were incorrectly getting
// true instead of inheriting from global_enable_default_user.
//
// Bug scenario:
// - global_enable_default_user = false
// - us-east-1: no enable_default_user set (should inherit false from global)
// - us-east-2: explicitly sets enable_default_user = false
//
// Bug behaviour: us-east-1 incorrectly got enable_default_user = true
// Fixed behaviour: us-east-1 should inherit false from global
func TestAccResourceRedisCloudActiveActiveDatabase_enableDefaultUserInheritance(t *testing.T) {
	subscriptionName := acctest.RandomWithPrefix(testResourcePrefix) + "-inherit-test"
	databaseName := acctest.RandomWithPrefix(testResourcePrefix) + "-database"
	password := acctest.RandString(20)

	const resourceName = "rediscloud_active_active_subscription_database.test"
	const subscriptionResourceName = "rediscloud_active_active_subscription.test"

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
			{
				Config: utils.RenderTestConfig(t, "./activeactive/testdata/enable_default_user_inheritance.tf", placeholders),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", databaseName),
					resource.TestCheckResourceAttr(resourceName, "global_enable_default_user", "false"),
					resource.TestCheckResourceAttr(resourceName, "override_region.#", "2"),
					// Verify via API that BOTH regions have enable_default_user = false
					// us-east-1 inherits from global, us-east-2 explicitly set
					checkEnableDefaultUserFromAPI(t, subscriptionResourceName, resourceName, map[string]bool{
						"us-east-1": false, // inherited from global_enable_default_user = false
						"us-east-2": false, // explicitly set to false
					}),
				),
			},
		},
	})
}

func TestAccResourceRedisCloudActiveActiveDatabase_enableDefaultUser(t *testing.T) {
	subscriptionName := acctest.RandomWithPrefix(testResourcePrefix) + "-enable-default-user"
	databaseName := acctest.RandomWithPrefix(testResourcePrefix) + "-database"
	password := acctest.RandString(20)

	const resourceName = "rediscloud_active_active_subscription_database.test"
	const subscriptionResourceName = "rediscloud_active_active_subscription.test"

	placeholders := map[string]string{
		"__SUBSCRIPTION_NAME__": subscriptionName,
		"__DATABASE_NAME__":     databaseName,
		"__PASSWORD__":          password,
	}

	// Track database ID to verify it's not recreated between steps
	var initialDbId string

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t); testAccAwsPreExistingCloudAccountPreCheck(t) },
		ProtoV5ProviderFactories: protoV5ProviderFactories,
		CheckDestroy:             testAccCheckActiveActiveSubscriptionDestroy,
		Steps: []resource.TestStep{
			// Step 1: global=true, both regions inherit
			{
				Config: utils.RenderTestConfig(t, "./activeactive/testdata/enable_default_user_global_true_inherit.tf", placeholders),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", databaseName),
					resource.TestCheckResourceAttr(resourceName, "global_enable_default_user", "true"),
					resource.TestCheckResourceAttr(resourceName, "override_region.#", "2"),
					// Both regions inherit global, so enable_default_user should NOT be in state
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "override_region.*", map[string]string{
						"name": "us-east-1",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "override_region.*", map[string]string{
						"name": "us-east-2",
					}),
					checkEnableDefaultUserFromAPI(t, subscriptionResourceName, resourceName, map[string]bool{
						"us-east-1": true, // inherited from global
						"us-east-2": true, // inherited from global
					}),
					// Capture the database ID for later verification
					captureDbId(resourceName, &initialDbId),
				),
			},
			// Step 2: global=true, us-east-1 overrides to false, us-east-2 inherits
			// Also tests override_global_password matching global_password doesn't cause drift
			{
				Config: utils.RenderTestConfig(t, "./activeactive/testdata/enable_default_user_mixed_overrides.tf", placeholders),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "global_enable_default_user", "true"),
					resource.TestCheckResourceAttr(resourceName, "override_region.#", "2"),
					// Explicit override, differs from global - should be in state
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "override_region.*", map[string]string{
						"name":                     "us-east-1",
						"enable_default_user":      "false",
						"override_global_password": password,
					}),
					// Inherits global - enable_default_user should not be in state
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "override_region.*", map[string]string{
						"name": "us-east-2",
					}),
					checkEnableDefaultUserFromAPI(t, subscriptionResourceName, resourceName, map[string]bool{
						"us-east-1": false, // explicit override
						"us-east-2": true,  // inherited from global
					}),
					verifyDbIdUnchanged(resourceName, &initialDbId),
				),
			},
			// Step 3: global=false, us-east-1 overrides to true, us-east-2 inherits false
			// This step catches the inheritance bug where us-east-2 incorrectly gets true
			{
				Config: utils.RenderTestConfig(t, "./activeactive/testdata/enable_default_user_global_false_region_true.tf", placeholders),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "global_enable_default_user", "false"),
					resource.TestCheckResourceAttr(resourceName, "override_region.#", "2"),
					// Explicitly true, differs from global, so IS in state
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "override_region.*", map[string]string{
						"name":                "us-east-1",
						"enable_default_user": "true",
					}),
					// Inherits global=false, so enable_default_user should NOT be in state
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "override_region.*", map[string]string{
						"name": "us-east-2",
					}),
					checkEnableDefaultUserFromAPI(t, subscriptionResourceName, resourceName, map[string]bool{
						"us-east-1": true,  // explicit override
						"us-east-2": false, // inherited from global=false
					}),
					// Verify database was NOT recreated
					verifyDbIdUnchanged(resourceName, &initialDbId),
				),
			},
			// Step 4: global=true, both regions explicit (us-east-1=true, us-east-2=false)
			{
				Config: utils.RenderTestConfig(t, "./activeactive/testdata/enable_default_user_all_explicit.tf", placeholders),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "global_enable_default_user", "true"),
					resource.TestCheckResourceAttr(resourceName, "override_region.#", "2"),
					// Explicitly true but matches global, so should NOT be in state
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "override_region.*", map[string]string{
						"name": "us-east-1",
					}),
					// Explicitly false, differs from global, so IS in state
					resource.TestCheckTypeSetElemNestedAttrs(resourceName, "override_region.*", map[string]string{
						"name":                "us-east-2",
						"enable_default_user": "false",
					}),
					checkEnableDefaultUserFromAPI(t, subscriptionResourceName, resourceName, map[string]bool{
						"us-east-1": true,  // explicit but matches global
						"us-east-2": false, // explicit override
					}),
					// Verify database was NOT recreated
					verifyDbIdUnchanged(resourceName, &initialDbId),
				),
			},
		},
	})
}

// checkEnableDefaultUserFromAPI returns a test check function that verifies the actual
// enable_default_user values from the Redis Cloud API for each region.
// This catches bugs where the provider sends incorrect values to the API that wouldn't
// be detected by only checking Terraform state.
func checkEnableDefaultUserFromAPI(t *testing.T, subscriptionResourceName, databaseResourceName string, expectedValues map[string]bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		subResource := s.RootModule().Resources[subscriptionResourceName]
		if subResource == nil {
			return fmt.Errorf("subscription resource %s not found", subscriptionResourceName)
		}
		subId, err := strconv.Atoi(subResource.Primary.ID)
		if err != nil {
			return fmt.Errorf("couldn't parse subscription ID: %s", subResource.Primary.ID)
		}

		dbResource := s.RootModule().Resources[databaseResourceName]
		if dbResource == nil {
			return fmt.Errorf("database resource %s not found", databaseResourceName)
		}
		dbId, err := strconv.Atoi(dbResource.Primary.Attributes["db_id"])
		if err != nil {
			return fmt.Errorf("couldn't parse database ID: %s", dbResource.Primary.Attributes["db_id"])
		}

		apiClient := sharedTestClient(t)
		db, err := apiClient.Client.Database.GetActiveActive(context.TODO(), subId, dbId)
		if err != nil {
			return fmt.Errorf("failed to get database from API: %w", err)
		}

		// Check each region's enable_default_user value
		for _, regionDb := range db.CrdbDatabases {
			regionName := redis.StringValue(regionDb.Region)
			expectedValue, ok := expectedValues[regionName]
			if !ok {
				continue // Skip regions not in expected values
			}

			actualValue := true // Default if not set
			if regionDb.Security != nil && regionDb.Security.EnableDefaultUser != nil {
				actualValue = redis.BoolValue(regionDb.Security.EnableDefaultUser)
			}

			if actualValue != expectedValue {
				return fmt.Errorf(
					"region %s: enable_default_user mismatch - expected %v from API, got %v",
					regionName, expectedValue, actualValue,
				)
			}
		}

		return nil
	}
}

// captureDbId captures the database ID from the first test step for later verification.
// This is used to ensure the database is not recreated between test steps.
func captureDbId(resourceName string, dbId *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs := s.RootModule().Resources[resourceName]
		if rs == nil {
			return fmt.Errorf("resource %s not found", resourceName)
		}
		*dbId = rs.Primary.Attributes["db_id"]
		if *dbId == "" {
			return fmt.Errorf("db_id attribute is empty")
		}
		return nil
	}
}

// verifyDbIdUnchanged verifies that the database ID has not changed from the initial value.
// If the ID changed, the database was recreated, which indicates a bug (e.g., state drift).
func verifyDbIdUnchanged(resourceName string, initialDbId *string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs := s.RootModule().Resources[resourceName]
		if rs == nil {
			return fmt.Errorf("resource %s not found", resourceName)
		}
		currentDbId := rs.Primary.Attributes["db_id"]
		if currentDbId != *initialDbId {
			return fmt.Errorf(
				"database was recreated! initial db_id=%s, current db_id=%s - this indicates state drift or unexpected replacement",
				*initialDbId, currentDbId,
			)
		}
		return nil
	}
}
