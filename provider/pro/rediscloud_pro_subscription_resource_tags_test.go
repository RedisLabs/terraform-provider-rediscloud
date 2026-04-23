package pro_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/testhelpers"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

func TestAccResourceRedisCloudProSubscription_ResourceTags(t *testing.T) {

	utils.AccRequiresEnvVar(t, "EXECUTE_TESTS")
	byocCloudAccountName := utils.AccRequiresEnvVar(t, "AWS_TEST_BYOC_CLOUD_ACCOUNT_NAME")

	name := acctest.RandomWithPrefix("tf-test") + "-resource-tags"
	const resourceName = "rediscloud_subscription.example"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testhelpers.BasicPreCheck(t) },
		ProtoV5ProviderFactories: testhelpers.ProtoV5ProviderFactories(),
		CheckDestroy:             checkProSubscriptionDestroy,
		Steps: []resource.TestStep{
			{
				// Step 1: Create with resource tags
				ConfigFile: config.StaticFile("testdata/pro_subscription_resource_tags.tf"),
				ConfigVariables: config.Variables{
					"cloud_account_name": config.StringVariable(byocCloudAccountName),
					"subscription_name":  config.StringVariable(name),
					"resource_tags": config.MapVariable(map[string]config.Variable{
						"environment": config.StringVariable("staging"),
						"team":        config.StringVariable("platform"),
					}),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.resource_tags.environment", "staging"),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.resource_tags.team", "platform"),
				),
			},
			{
				// Step 2: Update tags (change value, add key)
				ConfigFile: config.StaticFile("testdata/pro_subscription_resource_tags.tf"),
				ConfigVariables: config.Variables{
					"cloud_account_name": config.StringVariable(byocCloudAccountName),
					"subscription_name":  config.StringVariable(name),
					"resource_tags": config.MapVariable(map[string]config.Variable{
						"environment": config.StringVariable("production"),
						"team":        config.StringVariable("platform"),
						"cost-centre": config.StringVariable("engineering"),
					}),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.resource_tags.environment", "production"),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.resource_tags.team", "platform"),
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.resource_tags.cost-centre", "engineering"),
				),
			},
			{
				// Step 3: Refresh-only step to prove the GET path now returns tags directly
				// from the API (no drift after the update in Step 2).
				ConfigFile: config.StaticFile("testdata/pro_subscription_resource_tags.tf"),
				ConfigVariables: config.Variables{
					"cloud_account_name": config.StringVariable(byocCloudAccountName),
					"subscription_name":  config.StringVariable(name),
					"resource_tags": config.MapVariable(map[string]config.Variable{
						"environment": config.StringVariable("production"),
						"team":        config.StringVariable("platform"),
						"cost-centre": config.StringVariable("engineering"),
					}),
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: false,
			},
			{
				// Step 4: Remove all tags
				ConfigFile: config.StaticFile("testdata/pro_subscription_resource_tags.tf"),
				ConfigVariables: config.Variables{
					"cloud_account_name": config.StringVariable(byocCloudAccountName),
					"subscription_name":  config.StringVariable(name),
					"resource_tags":      config.MapVariable(map[string]config.Variable{}),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "cloud_provider.0.resource_tags.%", "0"),
				),
			},
			{
				// Step 5: Import — with the GET path, tags survive a round-trip through import.
				// Re-apply tags first so there's something to verify.
				ConfigFile: config.StaticFile("testdata/pro_subscription_resource_tags.tf"),
				ConfigVariables: config.Variables{
					"cloud_account_name": config.StringVariable(byocCloudAccountName),
					"subscription_name":  config.StringVariable(name),
					"resource_tags": config.MapVariable(map[string]config.Variable{
						"environment": config.StringVariable("staging"),
					}),
				},
			},
			{
				ConfigFile: config.StaticFile("testdata/pro_subscription_resource_tags.tf"),
				ConfigVariables: config.Variables{
					"cloud_account_name": config.StringVariable(byocCloudAccountName),
					"subscription_name":  config.StringVariable(name),
					"resource_tags": config.MapVariable(map[string]config.Variable{
						"environment": config.StringVariable("staging"),
					}),
				},
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"creation_plan",
					"redis_version",
				},
			},
		},
	})
}

