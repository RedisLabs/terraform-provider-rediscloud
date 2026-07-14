package pro_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func TestAccRedisCloudProDatabase_Passwordless(t *testing.T) {

	const databaseResource = "rediscloud_subscription_database.example"
	const datasourceName = "data.rediscloud_database.example"
	subscriptionName := utils.RandomWithPrefix()
	password := acctest.RandString(20)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, envchecks.RedisCloudCheck),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             checkProSubscriptionDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create a passwordless database
			{
				ConfigFile: config.StaticFile("testdata/pro_database_passwordless.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(subscriptionName),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("rediscloud_subscription.example", "public_endpoint_access", "false"),
					resource.TestCheckResourceAttr(databaseResource, "name", subscriptionName),
					resource.TestCheckResourceAttr(databaseResource, "enable_passwordless", "true"),
					resource.TestCheckResourceAttr(databaseResource, "password", ""),
					resource.TestCheckResourceAttr(datasourceName, "enable_passwordless", "true"),
					resource.TestCheckResourceAttr(datasourceName, "password", ""),
				),
			},
			// Step 2: Update from passwordless to password-protected
			{
				ConfigFile: config.StaticFile("testdata/pro_database_password_only.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(subscriptionName),
					"password":          config.StringVariable(password),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(databaseResource, "enable_passwordless", "false"),
					resource.TestCheckResourceAttr(databaseResource, "password", password),
					resource.TestCheckResourceAttr(datasourceName, "enable_passwordless", "false"),
				),
			},
			// Step 3: Update back to passwordless
			{
				ConfigFile: config.StaticFile("testdata/pro_database_passwordless.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(subscriptionName),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(databaseResource, "enable_passwordless", "true"),
					resource.TestCheckResourceAttr(databaseResource, "password", ""),
					resource.TestCheckResourceAttr(datasourceName, "enable_passwordless", "true"),
				),
			},
			// Step 4: Import and verify state
			{
				ConfigFile: config.StaticFile("testdata/pro_database_passwordless.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(subscriptionName),
				},
				ResourceName:            databaseResource,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"creation_plan"},
			},
		},
	})
}

func TestAccRedisCloudProDatabase_ExplicitPasswordFalseWithPassword(t *testing.T) {

	const databaseResource = "rediscloud_subscription_database.example"
	const datasourceName = "data.rediscloud_database.example"
	subscriptionName := utils.RandomWithPrefix()
	password := acctest.RandString(20)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, envchecks.RedisCloudCheck),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             checkProSubscriptionDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create with enable_passwordless=false and password set explicitly
			{
				ConfigFile: config.StaticFile("testdata/pro_database_explicit_password.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(subscriptionName),
					"password":          config.StringVariable(password),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(databaseResource, "enable_passwordless", "false"),
					resource.TestCheckResourceAttr(databaseResource, "password", password),
					resource.TestCheckResourceAttr(datasourceName, "enable_passwordless", "false"),
				),
			},
			// Step 2: Transition to passwordless (omitting password from config)
			{
				ConfigFile: config.StaticFile("testdata/pro_database_passwordless.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(subscriptionName),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(databaseResource, "enable_passwordless", "true"),
					resource.TestCheckResourceAttr(databaseResource, "password", ""),
					resource.TestCheckResourceAttr(datasourceName, "enable_passwordless", "true"),
				),
			},
		},
	})
}

func TestAccRedisCloudProDatabase_PasswordToPasswordless(t *testing.T) {

	const databaseResource = "rediscloud_subscription_database.example"
	const datasourceName = "data.rediscloud_database.example"
	subscriptionName := utils.RandomWithPrefix()
	password := acctest.RandString(20)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, envchecks.RedisCloudCheck),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             checkProSubscriptionDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create with password only (no enable_passwordless field)
			{
				ConfigFile: config.StaticFile("testdata/pro_database_password_only.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(subscriptionName),
					"password":          config.StringVariable(password),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(databaseResource, "enable_passwordless", "false"),
					resource.TestCheckResourceAttr(databaseResource, "password", password),
					resource.TestCheckResourceAttr(datasourceName, "enable_passwordless", "false"),
				),
			},
			// Step 2: Transition to passwordless (omitting password, setting enable_passwordless=true)
			{
				ConfigFile: config.StaticFile("testdata/pro_database_passwordless.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(subscriptionName),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(databaseResource, "enable_passwordless", "true"),
					resource.TestCheckResourceAttr(databaseResource, "password", ""),
					resource.TestCheckResourceAttr(datasourceName, "enable_passwordless", "true"),
				),
			},
		},
	})
}

func TestAccRedisCloudProDatabase_PasswordlessDisableWithoutPassword(t *testing.T) {

	subscriptionName := utils.RandomWithPrefix()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, envchecks.RedisCloudCheck),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             checkProSubscriptionDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create a passwordless database
			{
				ConfigFile: config.StaticFile("testdata/pro_database_passwordless.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(subscriptionName),
				},
			},
			// Step 2: Try to disable passwordless without providing a password — expect error
			{
				ConfigFile: config.StaticFile("testdata/pro_database_passwordless_disabled_no_password.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(subscriptionName),
				},
				ExpectError: regexp.MustCompile(`when disabling passwordless mode, you must provide a 'password'`),
			},
		},
	})
}

func TestAccRedisCloudProDatabase_PasswordlessWithPasswordConflict(t *testing.T) {

	subscriptionName := utils.RandomWithPrefix()
	password := acctest.RandString(20)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, envchecks.RedisCloudCheck),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             checkProSubscriptionDestroy,
		Steps: []resource.TestStep{
			// Try to create with both enable_passwordless=true and password — expect error at plan time
			{
				ConfigFile: config.StaticFile("testdata/pro_database_passwordless_with_password.tf"),
				ConfigVariables: config.Variables{
					"subscription_name": config.StringVariable(subscriptionName),
					"password":          config.StringVariable(password),
				},
				ExpectError: regexp.MustCompile(`"enable_passwordless" cannot be true when "password" is set`),
			},
		},
	})
}
