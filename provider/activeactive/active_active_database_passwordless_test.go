package activeactive_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
)

// TestAccActiveActiveDatabase_Passwordless creates a passwordless AA database,
// transitions to password-protected, and transitions back.
func TestAccActiveActiveDatabase_Passwordless(t *testing.T) {

	const databaseResource = "rediscloud_active_active_subscription_database.example"
	const datasourceName = "data.rediscloud_active_active_subscription_database.example"
	subscriptionName := testRandomWithPrefix()
	password := acctest.RandString(20)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { envchecks.RedisCloudCheck(t) },
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             checkAASubscriptionDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create passwordless AA database
			{
				ConfigFile: config.StaticFile("testdata/aa_database_passwordless.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(subscriptionName),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("rediscloud_active_active_subscription.example", "public_endpoint_access", "false"),
					resource.TestCheckResourceAttr(databaseResource, "name", subscriptionName),
					resource.TestCheckResourceAttr(databaseResource, "global_enable_passwordless", "true"),
					resource.TestCheckResourceAttr(databaseResource, "global_password", ""),
					resource.TestCheckResourceAttr(datasourceName, "global_enable_passwordless", "true"),
				),
			},
			// Step 2: Transition to password-protected
			{
				ConfigFile: config.StaticFile("testdata/aa_database_passwordless_to_password.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(subscriptionName),
					"password":          config.StringVariable(password),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(databaseResource, "global_enable_passwordless", "false"),
					resource.TestCheckResourceAttr(databaseResource, "global_password", password),
					resource.TestCheckResourceAttr(datasourceName, "global_enable_passwordless", "false"),
				),
			},
			// Step 3: Transition back to passwordless
			{
				ConfigFile: config.StaticFile("testdata/aa_database_passwordless.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(subscriptionName),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(databaseResource, "global_enable_passwordless", "true"),
					resource.TestCheckResourceAttr(databaseResource, "global_password", ""),
					resource.TestCheckResourceAttr(datasourceName, "global_enable_passwordless", "true"),
				),
			},
		},
	})
}

// TestAccActiveActiveDatabase_PasswordlessWithPasswordConflict verifies that
// setting both global_enable_passwordless=true and global_password produces a plan error.
func TestAccActiveActiveDatabase_PasswordlessWithPasswordConflict(t *testing.T) {

	subscriptionName := testRandomWithPrefix()
	password := acctest.RandString(20)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { envchecks.RedisCloudCheck(t) },
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             checkAASubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("testdata/aa_database_passwordless_with_password.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(subscriptionName),
					"password":          config.StringVariable(password),
				},
				ExpectError: regexp.MustCompile(`'global_enable_passwordless' cannot be true when 'global_password' is set`),
			},
		},
	})
}

// TestAccActiveActiveDatabase_PasswordlessRegionOverride tests per-region passwordless
// override with global password set.
func TestAccActiveActiveDatabase_PasswordlessRegionOverride(t *testing.T) {

	const databaseResource = "rediscloud_active_active_subscription_database.example"
	const datasourceName = "data.rediscloud_active_active_subscription_database.example"
	subscriptionName := testRandomWithPrefix()
	password := acctest.RandString(20)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { envchecks.RedisCloudCheck(t) },
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             checkAASubscriptionDestroy,
		Steps: []resource.TestStep{
			// Create with global password + one region overriding to passwordless
			{
				ConfigFile: config.StaticFile("testdata/aa_database_passwordless_override_region.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(subscriptionName),
					"password":          config.StringVariable(password),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(databaseResource, "name", subscriptionName),
					// Global is NOT passwordless (has a password)
					resource.TestCheckResourceAttr(databaseResource, "global_enable_passwordless", "false"),
					resource.TestCheckResourceAttr(databaseResource, "global_password", password),
					// Verify per-region passwordless override is read back
					resource.TestCheckTypeSetElemNestedAttrs(databaseResource, "override_region.*", map[string]string{
						"name":                                "us-east-1",
						"override_global_enable_passwordless": "true",
					}),
					// Data source shows not passwordless globally
					resource.TestCheckResourceAttr(datasourceName, "global_enable_passwordless", "false"),
				),
			},
		},
	})
}

// TestAccActiveActiveDatabase_PasswordlessDisableWithoutPassword verifies that
// transitioning from passwordless to password-protected without providing a password produces an error.
func TestAccActiveActiveDatabase_PasswordlessDisableWithoutPassword(t *testing.T) {

	subscriptionName := testRandomWithPrefix()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { envchecks.RedisCloudCheck(t) },
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             checkAASubscriptionDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create passwordless AA database
			{
				ConfigFile: config.StaticFile("testdata/aa_database_passwordless.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(subscriptionName),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("rediscloud_active_active_subscription_database.example", "global_enable_passwordless", "true"),
				),
			},
			// Step 2: Disable passwordless without providing a password — should error
			{
				ConfigFile: config.StaticFile("testdata/aa_database_passwordless_disabled_no_password.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(subscriptionName),
				},
				ExpectError: regexp.MustCompile(`When disabling passwordless mode, you must provide a 'global_password'`),
			},
		},
	})
}

// TestAccActiveActiveDatabase_PasswordlessRegionOverrideWithPasswordConflict verifies that
// setting both override_global_enable_passwordless=true and override_global_password in the
// same region produces a plan error.
func TestAccActiveActiveDatabase_PasswordlessRegionOverrideWithPasswordConflict(t *testing.T) {

	subscriptionName := testRandomWithPrefix()
	password := acctest.RandString(20)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { envchecks.RedisCloudCheck(t) },
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             checkAASubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile: config.StaticFile("testdata/aa_database_passwordless_override_with_password.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(subscriptionName),
					"password":          config.StringVariable(password),
				},
				ExpectError: regexp.MustCompile(`'override_global_enable_passwordless' cannot be true when 'override_global_password' is set`),
			},
		},
	})
}
