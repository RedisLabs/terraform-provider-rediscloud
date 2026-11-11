package essentialssubscription_test

import (
	"flag"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/envchecks"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

var essentialsMarketplaceFlag = flag.Bool("essentialsMarketplace", false,
	"Add this flag '-essentialsMarketplace' to run tests for marketplace associated accounts")

func subscriptionVariables(name string) config.Variables {
	return config.Variables{"subscription_name": config.StringVariable(name)}
}

func checkSubscription(name string, extra ...resource.TestCheckFunc) resource.TestCheckFunc {
	checks := append([]resource.TestCheckFunc{
		resource.TestCheckResourceAttrSet(resourceName, "id"),
		resource.TestCheckResourceAttr(resourceName, "name", name),
		resource.TestCheckResourceAttr(resourceName, "status", "active"),
		resource.TestCheckResourceAttrSet(resourceName, "plan_id"),
		resource.TestCheckResourceAttrSet(resourceName, "creation_date"),
	}, extra...)
	return resource.ComposeAggregateTestCheckFunc(checks...)
}

func TestAccResourceRedisCloudEssentialsSubscription_Free_CRUDI(t *testing.T) {
	name := utils.RandomWithPrefix()
	nameUpdated := name + "-updated"
	configFile := config.StaticFile("testdata/free/subscription.tf")

	resource.Test(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, testhelpers.PreCheckNoEssentialsSubscription),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testhelpers.CheckEssentialsSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile:      configFile,
				ConfigVariables: subscriptionVariables(name),
				Check:           checkSubscription(name, resource.TestCheckResourceAttr(resourceName, "payment_method_id", "0")),
			},
			{
				ConfigFile:      configFile,
				ConfigVariables: subscriptionVariables(nameUpdated),
				Check:           checkSubscription(nameUpdated, resource.TestCheckResourceAttr(resourceName, "payment_method_id", "0")),
			},
			{
				ConfigFile:        configFile,
				ConfigVariables:   subscriptionVariables(nameUpdated),
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				// timeouts is config-only and never returned on import.
				ImportStateVerifyIgnore: []string{"timeouts"},
			},
		},
	})
}

func TestAccResourceRedisCloudEssentialsSubscription_Paid_CreditCard_CRUDI(t *testing.T) {
	name := utils.RandomWithPrefix()
	nameUpdated := name + "-updated"
	configFile := config.StaticFile("testdata/paid_by_name/subscription.tf")

	resource.Test(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, testhelpers.PreCheckNoEssentialsSubscription),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testhelpers.CheckEssentialsSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile:      configFile,
				ConfigVariables: subscriptionVariables(name),
				Check: checkSubscription(name,
					resource.TestCheckResourceAttr(resourceName, "payment_method", "credit-card"),
					resource.TestCheckResourceAttrSet(resourceName, "payment_method_id"),
				),
			},
			{
				ConfigFile:      configFile,
				ConfigVariables: subscriptionVariables(nameUpdated),
				Check:           checkSubscription(nameUpdated, resource.TestCheckResourceAttrSet(resourceName, "payment_method_id")),
			},
			{
				ConfigFile:              configFile,
				ConfigVariables:         subscriptionVariables(nameUpdated),
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"timeouts"},
			},
		},
	})
}

func TestAccResourceRedisCloudEssentialsSubscription_Paid_NoPaymentType_CRUDI(t *testing.T) {
	name := utils.RandomWithPrefix()
	nameUpdated := name + "-updated"
	configFile := config.StaticFile("testdata/paid_no_payment_type_by_name/subscription.tf")

	resource.Test(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, testhelpers.PreCheckNoEssentialsSubscription),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testhelpers.CheckEssentialsSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile:      configFile,
				ConfigVariables: subscriptionVariables(name),
				Check: checkSubscription(name,
					resource.TestCheckResourceAttrSet(resourceName, "payment_method"),
					resource.TestCheckResourceAttrSet(resourceName, "payment_method_id"),
				),
			},
			{
				ConfigFile:      configFile,
				ConfigVariables: subscriptionVariables(nameUpdated),
				Check:           checkSubscription(nameUpdated, resource.TestCheckResourceAttrSet(resourceName, "payment_method_id")),
			},
			{
				ConfigFile:              configFile,
				ConfigVariables:         subscriptionVariables(nameUpdated),
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"timeouts"},
			},
		},
	})
}

func TestAccResourceRedisCloudEssentialsSubscription_Paid_Marketplace_CRUDI(t *testing.T) {
	// Only the qa environment has access to the marketplace, so this test is kept for manual runs.
	if !*essentialsMarketplaceFlag {
		t.Skip("The '-essentialsMarketplace' parameter wasn't provided in the test command.")
	}

	name := utils.RandomWithPrefix()
	nameUpdated := name + "-updated"
	configFile := config.StaticFile("testdata/marketplace/subscription.tf")

	resource.Test(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, testhelpers.PreCheckNoEssentialsSubscription),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testhelpers.CheckEssentialsSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile:      configFile,
				ConfigVariables: subscriptionVariables(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkSubscription(name, resource.TestCheckResourceAttr(resourceName, "payment_method", "marketplace")),
					resource.TestCheckResourceAttrSet(datasourceName, "id"),
					resource.TestCheckResourceAttr(datasourceName, "name", name),
					resource.TestCheckResourceAttr(datasourceName, "status", "active"),
					resource.TestCheckResourceAttrSet(datasourceName, "plan_id"),
					resource.TestCheckResourceAttrSet(datasourceName, "creation_date"),
				),
			},
			{
				ConfigFile:      configFile,
				ConfigVariables: subscriptionVariables(nameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkSubscription(nameUpdated),
					resource.TestCheckResourceAttr(datasourceName, "name", nameUpdated),
				),
			},
			{
				ConfigFile:        configFile,
				ConfigVariables:   subscriptionVariables(nameUpdated),
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				// The API does not always echo the marketplace payment method.
				ImportStateVerifyIgnore: []string{"timeouts", "payment_method"},
			},
		},
	})
}

func TestAccResourceRedisCloudEssentialsSubscription_Incorrect_PaymentIdForType(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 envchecks.ComposePreChecks(t, envchecks.RedisCloudCheck),
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             testhelpers.CheckEssentialsSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile:      config.StaticFile("testdata/incorrect_payment_id/subscription.tf"),
				ConfigVariables: subscriptionVariables(utils.RandomWithPrefix()),
				ExpectError:     regexp.MustCompile("payment methods aside from credit-card cannot have a payment ID"),
			},
		},
	})
}

// State written by the last SDKv2 release must plan cleanly with the framework implementation.
// Step 1 pins that release in its terraform block, since ExternalProviders cannot be combined with ConfigFile.
func TestAccResourceRedisCloudEssentialsSubscription_UpgradeFromSDK(t *testing.T) {
	name := utils.RandomWithPrefix()

	resource.Test(t, resource.TestCase{
		PreCheck:     envchecks.ComposePreChecks(t, testhelpers.PreCheckNoEssentialsSubscription),
		CheckDestroy: testhelpers.CheckEssentialsSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				ConfigFile:      config.StaticFile("testdata/upgrade_from_sdk/sdk_v2_20_0.tf"),
				ConfigVariables: subscriptionVariables(name),
				Check:           checkSubscription(name),
			},
			{
				ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
				ConfigFile:               config.StaticFile("testdata/upgrade_from_sdk/registry_source.tf"),
				ConfigVariables:          subscriptionVariables(name),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
				Check: checkSubscription(name),
			},
		},
	})
}
